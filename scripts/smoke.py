#!/usr/bin/env python3
"""Black-box test of all four services. Run `make build` first."""
import concurrent.futures
import datetime
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]

def free_port():
    with socket.socket() as s:
        s.bind(('127.0.0.1', 0))
        return s.getsockname()[1]

ports = {name: free_port() for name in ('gateway', 'rental', 'inventory', 'notifier')}
base = f'http://127.0.0.1:{ports["gateway"]}/api/v1'
env = dict(os.environ, SERVICE_KEY='smoke-test-internal-service-key',
           ADMIN_EMAIL='admin@example.com', ADMIN_PASSWORD='test-admin-password',
           STORAGE=os.environ.get('RENT_TEST_STORAGE','memory'), SEED_DEMO='true', GATEWAY_LISTEN=f'127.0.0.1:{ports["gateway"]}')
for name in ('rental', 'inventory', 'notifier'):
    addr = f'127.0.0.1:{ports[name]}'
    env[name.upper() + '_LISTEN'] = addr
    env[name.upper() + '_ADDR'] = addr

def request(method, path, payload=None, token=None, expected=200):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    body = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(base + path, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=12) as response:
            code, raw = response.status, response.read()
    except urllib.error.HTTPError as error:
        code, raw = error.code, error.read()
    value = json.loads(raw)
    assert code == expected, (method, path, code, expected, value)
    return value

processes = []
with tempfile.TemporaryFile(mode='w+') as log:
    try:
        for name in ('inventory', 'notifier', 'rental', 'gateway'):
            processes.append(subprocess.Popen([str(ROOT / 'bin' / name)], env=env,
                                              stdout=log, stderr=log))
        for _ in range(100):
            try:
                models = request('GET', '/models')['items']
                admin = request('POST', '/auth/login', {'email': env['ADMIN_EMAIL'],
                                                       'password': env['ADMIN_PASSWORD']})['token']
                break
            except (OSError, AssertionError):
                time.sleep(.1)
        else:
            raise AssertionError('services did not become ready')
        with urllib.request.urlopen(base.removesuffix('/api/v1') + '/') as response:
            html = response.read().decode()
            assert response.status == 200 and 'lang="ru"' in html and '/assets/app.js' in html
            assert "default-src 'self'" in response.headers.get('Content-Security-Policy', '')
        for asset in ('app.js', 'styles.css'):
            with urllib.request.urlopen(base.removesuffix('/api/v1') + '/assets/' + asset) as response:
                assert response.status == 200 and len(response.read()) > 100
        assert len(models) == 2
        model = models[0]
        start = (datetime.date.today() + datetime.timedelta(days=30)).isoformat()
        end = (datetime.date.today() + datetime.timedelta(days=32)).isoformat()
        later = (datetime.date.today() + datetime.timedelta(days=35)).isoformat()
        cart = {'items': [{'model_id': model['id'], 'quantity': 2, 'start': start, 'end': end}]}
        users = []
        for n in range(2):
            users.append(request('POST', '/auth/register', {
                'email': f'client{n}@example.com', 'password': 'customer-password'}))
        token = users[0]['token']
        request('GET', '/cart', expected=401)
        request('POST', '/models', {'name': 'Forbidden', 'category': 'x', 'daily_price': '100'}, token, 403)
        request('PATCH', '/users/' + users[0]['user']['id'] + '/role', {'role': 'admin'}, token, 403)
        request('PUT', '/cart', {'items': [{'model_id': model['id'], 'quantity': 0,
                                          'start': start, 'end': end}]}, token, 400)
        for user in users:
            request('PUT', '/cart', cart, user['token'])
        def checkout(user):
            req = urllib.request.Request(base + '/orders',
                data=json.dumps({'request_id': 'concurrent'}).encode(),
                headers={'Content-Type': 'application/json', 'Authorization': 'Bearer ' + user['token']})
            try:
                with urllib.request.urlopen(req, timeout=12) as response:
                    return response.status, json.load(response)
            except urllib.error.HTTPError as e:
                return e.code, json.load(e)
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            outcomes = list(pool.map(checkout, users))
        assert sorted(code for code, _ in outcomes) == [200, 409], outcomes
        index = next(i for i, (code, _) in enumerate(outcomes) if code == 200)
        order = outcomes[index][1]
        owner = users[index]['token']
        other = users[1-index]['token']

        if env['STORAGE'] == 'postgres':
            # Recreate every application process while preserving only the PostgreSQL data.
            for process in processes:
                process.terminate()
            for process in processes:
                process.wait(timeout=10)
            processes.clear()
            for name in ('inventory', 'notifier', 'rental', 'gateway'):
                processes.append(subprocess.Popen([str(ROOT / 'bin' / name)], env=env,
                                                  stdout=log, stderr=log))
            for _ in range(100):
                try:
                    restored = request('GET', '/orders/' + order['id'], token=owner)
                    assert restored['state'] == 'confirmed'
                    restored_models = request('GET', '/models')['items']
                    assert [m['id'] for m in restored_models] == [m['id'] for m in models]
                    assert len(request('GET', '/cart', token=other)['items']) == 1
                    break
                except (OSError, AssertionError):
                    time.sleep(.1)
            else:
                raise AssertionError('state or sessions did not survive service restart')
            # Restart does not reset the administrator's credentials or duplicate the demo seed.
            request('POST', '/auth/login', {'email': env['ADMIN_EMAIL'], 'password': env['ADMIN_PASSWORD']})
            print('PASS: PostgreSQL restart preserves users, sessions, carts, orders, reservations and demo IDs')
        assert order['state'] == 'confirmed'
        assert int(order['total']) == int(model['daily_price']) * 4
        repeat = request('POST', '/orders', {'request_id': 'concurrent'}, owner)
        assert repeat['id'] == order['id']
        request('GET', '/orders/' + order['id'], token=other, expected=403)
        request('POST', '/orders/' + order['id'] + '/issue', {}, owner, 403)
        cal = request('GET', f'/models/{model["id"]}/calendar?start={start}&end={end}')
        assert all(day['available'] == 1 for day in cal['days'])
        request('POST', '/orders/' + order['id'] + '/issue', {}, admin)
        unit_ids = order['items'][0]['unit_ids']
        partial = request('POST', '/orders/' + order['id'] + '/return', {'unit_ids': unit_ids[:1]}, admin)
        assert partial['state'] == 'active'
        complete = request('POST', '/orders/' + order['id'] + '/return', {'unit_ids': unit_ids[1:]}, admin)
        assert complete['state'] == 'completed'
        request('POST', '/orders/' + order['id'] + '/return', {'unit_ids': unit_ids}, admin)
        # Different periods per cart position.
        request('PUT', '/cart', {'items': [
            {'model_id': model['id'], 'quantity': 1, 'start': end, 'end': later},
            {'model_id': models[1]['id'], 'quantity': 1, 'start': start, 'end': later}]}, owner)
        second = request('POST', '/orders', {'request_id': 'different-periods'}, owner)
        assert second['state'] == 'confirmed' and len(second['items']) == 2
        request('POST', '/orders/' + second['id'] + '/cancel', {}, admin, 400)
        cancelled = request('POST', '/orders/' + second['id'] + '/cancel', {'reason': 'Customer called'}, admin)
        assert cancelled['state'] == 'cancelled' and cancelled['history'][-1]['reason'] == 'Customer called'
        if env['STORAGE'] == 'postgres':
            # Persist a pending operation, restart Rental, then bring Inventory back.
            processes[0].terminate()
            processes[0].wait(timeout=10)
            pending_end = (datetime.date.fromisoformat(later) + datetime.timedelta(days=2)).isoformat()
            request('PUT', '/cart', {'items': [{'model_id': model['id'], 'quantity': 1,
                                              'start': later, 'end': pending_end}]}, owner)
            pending = request('POST', '/orders', {'request_id': 'resume-after-restart'}, owner)
            assert pending['state'] == 'pending', pending
            processes[2].terminate()
            processes[2].wait(timeout=10)
            processes[2] = subprocess.Popen([str(ROOT / 'bin' / 'rental')], env=env, stdout=log, stderr=log)
            processes[0] = subprocess.Popen([str(ROOT / 'bin' / 'inventory')], env=env, stdout=log, stderr=log)
            for _ in range(100):
                try:
                    recovered = request('GET', '/orders/' + pending['id'], token=owner)
                    if recovered['state'] == 'confirmed':
                        break
                except (OSError, AssertionError):
                    pass
                time.sleep(.1)
            else:
                raise AssertionError('pending order did not recover after restart')
            request('POST', '/orders/' + pending['id'] + '/cancel', {}, owner)
            print('PASS: pending checkout resumes from PostgreSQL after Rental/Inventory restart')

        # Staff and administrator responsibilities.
        staff_id = users[index]['user']['id']
        request('PATCH', '/users/' + staff_id + '/role', {'role': 'staff'}, admin)
        request('POST', '/models', {'name': 'Test drill', 'category': 'drills', 'daily_price': '50000'}, owner, 403)
        created = request('POST', '/models', {'name': 'Test drill', 'category': 'drills', 'daily_price': '50000'}, admin)
        unit = request('POST', '/models/' + created['id'] + '/units', token=owner)
        units = request('GET', '/models/' + created['id'] + '/units', token=owner)['items']
        assert len(units) == 1 and units[0]['id'] == unit['id']
        created['daily_price'] = '60000'
        created['image_urls'] = ['https://example.com/drill.png']
        updated = request('PUT', '/models/' + created['id'], created, admin)
        assert updated['daily_price'] == '60000' and len(updated['image_urls']) == 1
        request('PATCH', '/units/' + unit['id'], {'state': 'maintenance'}, owner)
        filtered = request('GET', '/models?category=drills&query=Test')['items']
        assert len(filtered) == 1
        print('PASS: four services, auth/RBAC, catalogue/calendar, concurrent atomic booking,')
        print('idempotent checkout, different dates, partial return, cancellation and staff actions')
    except BaseException:
        log.flush()
        log.seek(0)
        print(log.read())
        raise
    finally:
        for process in processes:
            process.terminate()
        for process in processes:
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
