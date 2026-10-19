#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
project="rent-tests-$$"
compose=(docker compose -p "$project" -f compose.test.yaml)
cleanup() { "${compose[@]}" down -v >/dev/null; }
trap cleanup EXIT
"${compose[@]}" up -d --wait postgres
address=$("${compose[@]}" port postgres 5432)
port=${address##*:}
for service in rental inventory notifier; do
 variable="${service^^}_TEST_DATABASE_URL"
 export "$variable=postgres://rent_$service:rent-test-password@127.0.0.1:$port/rent_${service}_tests?sslmode=disable&timezone=UTC"
 variable="${service^^}_DATABASE_URL"
 export "$variable=postgres://rent_$service:rent-test-password@127.0.0.1:$port/rent_$service?sslmode=disable&timezone=UTC"
done
"${GO:-go}" test -buildvcs=false -race ./...
RENT_TEST_STORAGE=postgres python3 scripts/smoke.py
