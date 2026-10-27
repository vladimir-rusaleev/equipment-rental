const $ = (selector, root = document) => root.querySelector(selector);
const main = $('#main');
const modal = $('#modal');
const esc = (value = '') => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const money = value => new Intl.NumberFormat('ru-RU', {style:'currency',currency:'RUB',minimumFractionDigits:0,maximumFractionDigits:2}).format(Number(value || 0) / 100);
const categories = {'rotary-hammers':'Перфораторы',jigsaws:'Лобзики',drills:'Дрели и шуруповёрты',grinders:'Шлифмашины',saws:'Пилы',vacuums:'Пылесосы',levels:'Измерительный инструмент',welders:'Сварочные аппараты'};
const statuses = {pending:'Оформляется',confirmed:'Забронирован',active:'В аренде',completed:'Завершён',cancelled:'Отменён',rejected:'Не подтверждён',cancelling:'Отменяется',issuing:'Выдаётся',returning:'Принимается возврат'};
const roleNames = {customer:'Клиент',staff:'Сотрудник',admin:'Администратор'};
const stateNames = {ready:'Готов к выдаче',maintenance:'На обслуживании',retired:'Списан'};
const actionNames = {reserve:'Бронирование',cancel:'Отмена',issue:'Выдача',return:'Возврат'};
const read = (key, fallback) => {try {return JSON.parse(localStorage.getItem(key)) ?? fallback;} catch {return fallback;}};
const write = (key,value) => {try {localStorage.setItem(key,JSON.stringify(value));} catch {}};
const state = {token:read('rent.session',''), user:null, models:[], cart:[], orders:[], view:'catalog', filters:{query:'',category:'',start:'',end:''}, staffTab:'orders', loading:false, checkoutKey:read('rent.checkout',''), modalItem:null};
const staff = () => ['staff','admin'].includes(state.user?.role);
const admin = () => state.user?.role === 'admin';
const uid = () => crypto.randomUUID();
const date = value => value ? new Intl.DateTimeFormat('ru-RU',{day:'numeric',month:'short'}).format(new Date(value+'T12:00:00')) : '—';
const today = (offset=0) => {const d=new Date(); d.setDate(d.getDate()+offset); return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;};
const days = item => Math.round((Date.parse(item.end)-Date.parse(item.start))/86400000);
const validItem = item => item.model_id && Number.isInteger(Number(item.quantity)) && Number(item.quantity)>0 && Number(item.quantity)<=100 && Number.isFinite(days(item)) && days(item)>0 && days(item)<=366;
const estimate = item => Number(state.models.find(m=>m.id===item.model_id)?.daily_price || 0)*Number(item.quantity)*days(item);
const icon = name => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${({catalog:'<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',cart:'<path d="M3 3h2l3 12h10l3-9H6"/><circle cx="9" cy="20" r="1"/><circle cx="18" cy="20" r="1"/>',orders:'<rect x="5" y="4" width="14" height="18" rx="2"/><path d="M9 2h6v4H9zM9 11h6M9 15h6"/>',staff:'<path d="M12 3l8 4v6c0 4-8 8-8 8s-8-4-8-8V7z"/><path d="M8 12l3 3 5-6"/>'})[name] || ''}</svg>`;
function illustration(category='') {
 const shape = category==='jigsaws' || category==='saws'
  ? `<path d="M47 53V37c0-10 9-16 17-16h23c12 0 18 7 18 18v14" fill="none" stroke="#354650" stroke-width="9"/>
     <path d="M38 53h70l10 34-9 13H34V72z" fill="#337c86" stroke="#344b50" stroke-width="2"/>
     <path d="M48 55h48M47 64h20m-20 7h20m-20 7h20" stroke="#b1d0cc" stroke-width="3"/>
     <path d="M34 96h85v9H27v-5z" fill="#5c666b"/><path d="M102 96v25" stroke="#737d83" stroke-width="3"/>
     <path d="M33 88h42l8 8H33z" fill="#2d3c43"/><circle cx="100" cy="78" r="7" fill="#2d3c43"/>`
  : `<path d="M24 42l17-9h53l13 9v35H36L24 64z" fill="#3b7986" stroke="#324d56" stroke-width="2"/>
     <path d="M29 43h31v24H31z" fill="#326471"/><path d="M34 48h19m-19 6h19m-19 6h19" stroke="#a4c3c6" stroke-width="2"/>
     <path d="M105 45h13v27h-13z" fill="#2c3840"/><path d="M118 49h12v19h-12z" fill="#66727a"/>
     <path d="M130 56h22m-19-4 5 8m3-8 5 8" stroke="#7c878d" stroke-width="3"/>
     <path d="M48 77h28l-5 32H48z" fill="#293941"/><path d="M54 82h13l-3 19H53z" fill="#50646a"/>
     <path d="M40 108h40v11H37z" fill="#344a53"/><path d="M72 77v13h8V78" fill="none" stroke="#293941" stroke-width="4"/>
     <path d="M83 43h13v19H83z" fill="#e2e7dc"/>`;
 return `<svg viewBox="0 0 170 140" role="img" aria-label="Схематичное изображение инструмента">${shape}</svg>`;
}
function visual(model) {
 const url = model.image_urls?.find(u=>/^https:\/\//i.test(u));
 return `<div class="product-visual">${url ? `<img src="${esc(url)}" alt="${esc(model.name)}" referrerpolicy="no-referrer">` : illustration(model.category)}</div>`;
}
function toast(message, error=false) {const el=document.createElement('div');el.className='toast'+(error?' error':'');el.textContent=message;$('#toasts').append(el);setTimeout(()=>el.remove(),6000);}
const messages = {'invalid input':'Проверьте даты, количество и заполненные поля.','unauthenticated':'Проверьте почту и пароль или войдите заново.','forbidden':'Недостаточно прав для этого действия.','dependency unavailable':'Сервис временно недоступен. Попробуйте позже.','operation not allowed':'Действие недоступно для текущего состояния.','conflict':'Данные изменились или операция конфликтует с существующими записями.','not found':'Запись не найдена.','insufficient permissions':'Недостаточно прав для этого действия.'};
async function api(path, {method='GET',body,auth=true}={}) {
 const headers={'Accept':'application/json'};
 if (body!==undefined) headers['Content-Type']='application/json';
 if (auth && state.token) headers.Authorization='Bearer '+state.token;
 let response;
 try {response=await fetch('/api/v1'+path,{method,headers,body:body===undefined?undefined:JSON.stringify(body),signal:AbortSignal.timeout(12000)});} catch {throw new Error('Не удалось связаться с сервером. Попробуйте позже.');}
 let data;try {data=await response.json();} catch {throw new Error('Сервер вернул неожиданный ответ.');}
 if(!response.ok) {
  if(response.status===401 && auth && state.token){state.token='';state.user=null;write('rent.session','');navigation();}
  const raw=data.error||'';
  const text=messages[raw] || (raw.includes('insufficient equipment') ? 'На выбранные даты недостаточно свободных экземпляров. Обновите доступность.' : raw.includes('cancellation reason') ? 'Укажите причину отмены чужого заказа.' : raw.includes('outstanding reservations') ? 'У экземпляра есть незавершённые бронирования.' : raw.includes('not physically returned') ? 'Инструмент ещё не возвращён по предыдущему заказу.' : raw || `Ошибка запроса (${response.status}).`);
  const error=new Error(text);error.status=response.status;throw error;
 }
 return data;
}
function navigation() {
 const count=state.cart.reduce((sum,i)=>sum+Number(i.quantity),0);
 const links=[['catalog','Каталог'],['cart','Корзина'],['orders','Мои заказы'],...(staff()?[['staff','Панель проката']]:[])];
 $('#navigation').innerHTML=links.map(([id,label])=>`<a href="#${id}" class="${state.view===id?'active':''}">${label}${id==='cart'&&count?` (${count})`:''}</a>`).join('');
 $('#side-navigation').innerHTML=links.map(([id,label])=>`<a href="#${id}" class="side-link ${state.view===id?'active':''}">${icon(id)}${label}${id==='cart'&&count?`<span class="badge">${count}</span>`:''}</a>`).join('');
 $('#account').innerHTML=state.user?`<span class="account-name">${esc(state.user.email)} · ${roleNames[state.user.role]||''}</span><button class="account-button" data-action="logout">Выйти</button>`:'<button class="account-button" data-action="login">Войти в кабинет</button>';
}
function heading(eyebrow,title,subtitle='',extra='') {return `<div class="page-heading"><div><h1>${title}</h1>${subtitle?`<p>${subtitle}</p>`:""}</div>${extra}</div>`;}
function empty(title,text,action='catalog',label='В каталог') {return `<div class="empty">${icon('orders')}<h2>${title}</h2><p>${text}</p><a class="button" href="#${action}">${label}</a></div>`;}
function showModal(title,content) {modal.innerHTML=`<div class="dialog-top"><h2 id="modal-title">${title}</h2><button class="close" data-action="close" aria-label="Закрыть">×</button></div>${content}`;if(!modal.open)modal.showModal();}
function errorBox(message) {return `<div class="notice error">${esc(message)}</div>`;}
function authModal(register=false) {
 showModal(register?'Создать аккаунт':'Вход в личный кабинет',`<p class="hint">${register?'Сохраняйте бронирования и следите за своими заказами.':'Войдите, чтобы оформить аренду или открыть заказы.'}</p><form id="auth-form" class="form-stack" data-register="${register}"><div class="field"><label for="email">Электронная почта</label><input id="email" name="email" type="email" autocomplete="email" required></div><div class="field"><label for="password">Пароль${register?' · от 10 символов':''}</label><input id="password" name="password" type="password" autocomplete="${register?'new-password':'current-password'}" minlength="${register?10:1}" maxlength="72" required></div><div id="form-error"></div><button class="button full">${register?'Зарегистрироваться':'Войти'}</button></form><p class="hint">${register?'Уже есть аккаунт?':'Первый раз здесь?'} <button class="text-button" data-action="${register?'login':'register'}">${register?'Войти':'Зарегистрироваться'}</button></p>`);
}
async function loadCatalog() {
 const q=new URLSearchParams();Object.entries(state.filters).forEach(([key,value])=>{if(value)q.set(key,value);});
 state.models=(await api('/models?'+q,{auth:false})).items||[];
}
function renderCatalog() {
 const f=state.filters;
 const known=new Set([...Object.keys(categories),...state.models.map(m=>m.category)]);
 main.innerHTML=heading('','Каталог инструментов','Выберите инструмент и проверьте доступность на нужные даты.')+
 ` <form id="filters" class="filters"><div class="field"><label for="search">Поиск по каталогу</label><input id="search" name="query" placeholder="Найти инструмент…" value="${esc(f.query)}"></div><div class="field"><label for="category">Категория</label><select id="category" name="category"><option value="">Все инструменты</option>${[...known].map(c=>`<option value="${esc(c)}" ${f.category===c?'selected':''}>${esc(categories[c]||c)}</option>`).join('')}</select></div><div class="field"><label for="filter-start">Начало аренды</label><input id="filter-start" name="start" type="date" value="${esc(f.start)}"></div><div class="field"><label for="filter-end">Возврат</label><input id="filter-end" name="end" type="date" value="${esc(f.end)}"></div><button class="button">Найти</button></form>
 <div class="section-line"><span>Найдено моделей: <b>${state.models.length}</b></span><span>${f.start?`${date(f.start)} — ${date(f.end)} · дата возврата не оплачивается`:'Стоимость указана за сутки'}</span></div>
 ${state.models.length?`<div class="cards">${state.models.map(m=>`<article class="card" role="button" tabindex="0" data-action="product" data-id="${esc(m.id)}" aria-label="Открыть ${esc(m.name)}">${visual(m)}<div class="card-body"><div class="category">${esc(categories[m.category]||m.category)}</div><h3>${esc(m.name)}</h3><p class="card-description">${esc(m.description||'Описание не добавлено.')}</p><div class="availability"><span class="status-dot"></span>${f.start?`Свободно: ${m.available} шт.`:'Укажите даты для проверки наличия'}</div><div class="card-bottom"><span class="price">${money(m.daily_price)}<small> / сутки</small></span></div></div></article>`).join('')}</div>`:empty('Ничего не нашлось','Попробуйте изменить категорию, запрос или даты.','catalog','Сбросить фильтры').replace('class="button"','class="button" data-action="reset-filters"')}`;
}
async function productModal(id) {
 const model=state.models.find(m=>m.id===id)||await api('/models/'+id,{auth:false});
 state.modalItem=model;
 const start=state.filters.start||today(1),end=state.filters.end||today(3);
 showModal(esc(model.name),`<div class="dialog-product">${visual(model)}</div><p class="hint">${esc(model.description)}</p><p><b>${money(model.daily_price)}</b> <span class="muted">за сутки</span></p><form id="product-form" class="form-stack"><div class="form-row"><div class="field"><label for="product-start">Начало аренды</label><input id="product-start" type="date" name="start" value="${start}" required></div><div class="field"><label for="product-end">Возврат (не включается)</label><input id="product-end" type="date" name="end" value="${end}" required></div></div><div class="field"><label for="product-quantity">Количество</label><input id="product-quantity" type="number" name="quantity" min="1" max="100" value="1" required></div><div id="product-estimate" class="notice"></div><div><div class="calendar-head">Доступность по дням выбранного периода</div><div id="calendar" class="calendar"></div><div class="fineprint">Число под датой — свободные экземпляры. Окончательная проверка выполняется при оформлении всего заказа.</div></div><div id="form-error"></div><button class="button full">Добавить в корзину</button></form>`);
 updateProductEstimate();await updateCalendar();
}
function updateProductEstimate() {const form=$('#product-form');if(!form)return;const i=Object.fromEntries(new FormData(form));$('#product-estimate').textContent=validItem({...i,model_id:state.modalItem.id})?`${days(i)} сут. × ${i.quantity} шт. = ${money(days(i)*Number(i.quantity)*Number(state.modalItem.daily_price))}`:'Выберите корректные даты и количество.';}
let calendarSequence=0;
async function updateCalendar() {
 const form=$('#product-form');if(!form)return;const sequence=++calendarSequence;
 const i=Object.fromEntries(new FormData(form));if(!validItem({...i,model_id:state.modalItem.id})){$('#calendar').innerHTML='';return;}
 $('#calendar').innerHTML='<span class="hint">Загрузка…</span>';
 try {const data=await api(`/models/${state.modalItem.id}/calendar?`+new URLSearchParams({start:i.start,end:i.end}),{auth:false});if(sequence!==calendarSequence||!$('#calendar'))return;
 $('#calendar').innerHTML=data.days.map(d=>`<div class="calendar-day ${d.available<Number(i.quantity)?'unavailable':''}" title="${esc(d.date)}: свободно ${d.available}">${date(d.date)}<small>${d.available} шт.</small></div>`).join('');
 } catch(e) {if(sequence===calendarSequence&&$('#calendar'))$('#calendar').innerHTML=`<span class="hint">${esc(e.message)}</span>`;}
}
async function saveCart(items) {
 if(state.user){state.cart=(await api('/cart',{method:'PUT',body:{items}})).items||[];}else{state.cart=items;write('rent.guest-cart',items);}
 navigation();
}
function renderCart() {
 main.innerHTML=heading('СОБЕРИТЕ ВСЁ НУЖНОЕ','Ваша корзина','У каждого инструмента может быть свой период аренды.')+
 (state.cart.length?`<div class="notice">Корзина не резервирует инструмент. Доступность и окончательная стоимость проверяются при оформлении.</div><div class="cart-layout"><form id="cart-form" class="panel">${state.cart.map((i,index)=>{const m=state.models.find(m=>m.id===i.model_id);return `<div class="cart-row"><div><h3>${esc(m?.name||'Инструмент '+i.model_id.slice(0,8))}</h3><span class="hint">${m?money(m.daily_price)+' / сутки':'Тариф будет проверен при оформлении'}</span><div class="cart-fields"><div class="field"><label for="start-${index}">Начало</label><input id="start-${index}" name="start-${index}" type="date" value="${esc(i.start)}" required></div><div class="field"><label for="end-${index}">Возврат</label><input id="end-${index}" name="end-${index}" type="date" value="${esc(i.end)}" required></div><div class="field"><label for="qty-${index}">Кол-во</label><input id="qty-${index}" name="quantity-${index}" type="number" min="1" max="100" value="${Number(i.quantity)}" required></div></div></div><div><b data-estimate="${index}">${money(estimate(i))}</b><br><button type="button" class="text-button" data-action="remove-item" data-index="${index}">Убрать</button></div></div>`;}).join('')}<div class="actions"><button class="button secondary">Сохранить изменения</button></div></form><aside class="panel summary"><h2>Оформление аренды</h2><div class="summary-line"><span>Позиций</span><b>${state.cart.length}</b></div><div class="summary-line"><span>Получение</span><b>Самовывоз</b></div><div class="summary-total"><span>Итого ≈</span><span id="cart-total">${money(state.cart.reduce((s,i)=>s+estimate(i),0))}</span></div><button class="button full" data-action="checkout">${state.user?'Забронировать':'Войти и забронировать'}</button><p class="fineprint">Предварительный расчёт. Оплата в пункте проката. Период считается до даты возврата, не включая её.</p></aside></div>`:empty('Пока ничего не выбрано','Добавьте инструменты из каталога. Для каждой позиции можно выбрать свои даты.'));
}
function orderHTML(o,employee=false) {
 const pending=['pending','cancelling','issuing','returning'].includes(o.state);
 return `<article class="panel"><div class="order-header"><div><h3>Заказ <span class="order-id">#${esc(o.id.slice(0,8))}</span></h3>${employee?`<span class="hint">Клиент: <code>${esc(o.user_id)}</code></span>`:''}</div><span class="pill ${esc(o.state)}">${statuses[o.state]||esc(o.state)}</span></div>${pending?'<div class="notice">Сервисы согласовывают операцию. Статус обновится автоматически.</div>':''}${o.error?errorBox(messages[o.error]||o.error):''}<div class="order-items">${o.items.map(q=>`<div class="order-item"><div><b>${esc(q.name||'Инструмент '+q.item.model_id.slice(0,8))}</b><small>${date(q.item.start)} — ${date(q.item.end)} · ${q.item.quantity} шт. · ${money(q.daily_price)} / сутки</small>${employee&&o.state==='active'?q.unit_ids.filter(id=>!q.returned_unit_ids.includes(id)).map(id=>`<label class="unit-check"><input type="checkbox" data-return="${esc(o.id)}" value="${esc(id)}"> Экземпляр ${esc(id.slice(0,10))}</label>`).join(''):''}${q.returned_unit_ids?.length?`<small>Возвращено: ${q.returned_unit_ids.length} из ${q.unit_ids.length}</small>`:''}</div><b>${money(q.total)}</b></div>`).join('')}</div><div class="summary-line"><span>Стоимость заказа</span><b>${money(o.total)}</b></div><div class="actions">${o.state==='confirmed'?`<button class="button danger small" data-action="cancel-order" data-id="${esc(o.id)}">Отменить бронь</button>${employee?`<button class="button small" data-action="issue-order" data-id="${esc(o.id)}">Выдать инструмент</button>`:''}`:''}${employee&&o.state==='active'?`<button class="button small" data-action="return-order" data-id="${esc(o.id)}">Принять выбранные</button><button class="button secondary small" data-action="return-all" data-id="${esc(o.id)}">Принять всё</button>`:''}</div>${o.history.length?`<details class="order-history"><summary>История · ${o.history.length}</summary>${o.history.map(h=>`<p>${esc(actionNames[h.action]||h.action)} · ${new Date(h.occurred_at).toLocaleString('ru-RU')}${h.reason?' · '+esc(h.reason):''}<br>Автор: ${esc(h.actor_id)}</p>`).join('')}</details>`:''}</article>`;
}
async function loadOrders() {state.orders=(await api('/orders')).items||[];}
function renderOrders(employee=false) {
 const orders=employee?state.orders:state.orders.filter(o=>o.user_id===state.user.id);
 main.innerHTML=employee?staffHeading()+`<div class="section-line"><span>ВСЕ ЗАКАЗЫ / ${orders.length}</span><button class="text-button" data-action="refresh">Обновить</button></div>`:heading('ЛИЧНЫЙ КАБИНЕТ','Мои заказы','Бронирования, текущая аренда и история.', '<button class="button secondary small" data-action="refresh">Обновить</button>')+`<p class="hint">Ваш ID: <code>${esc(state.user.id)}</code></p>`;
 main.innerHTML+=orders.length?[...orders].reverse().map(o=>orderHTML(o,employee)).join(''):empty('Заказов пока нет',employee?'Новые бронирования появятся здесь.':'После оформления бронирования заказ появится здесь.');
}
function staffHeading() {return heading('РАБОЧЕЕ МЕСТО','Панель проката','Управление заказами и оборудованием.')+`<div class="tabs">${[['orders','Заказы'],['inventory','Оборудование'],...(admin()?[['users','Сотрудники']]:[])].map(([key,label])=>`<button class="tab ${state.staffTab===key?'active':''}" data-action="staff-tab" data-tab="${key}">${label}</button>`).join('')}</div>`;}
function renderInventory() {
 main.innerHTML=staffHeading()+`<div class="section-line"><span>МОДЕЛИ / ${state.models.length}</span>${admin()?'<button class="button small" data-action="new-model">Добавить модель +</button>':''}</div><div class="panel">${state.models.length?state.models.map(m=>`<div class="admin-row"><div><h3>${esc(m.name)}</h3><small>${esc(categories[m.category]||m.category)} · ${money(m.daily_price)} / сутки</small></div><div class="actions"><button class="button secondary small" data-action="units" data-id="${esc(m.id)}">Экземпляры</button>${admin()?`<button class="button secondary small" data-action="edit-model" data-id="${esc(m.id)}">Изменить</button>`:''}</div></div>`).join(''):'<p class="hint">Каталог пуст. Добавьте первую модель.</p>'}</div>`;
}
function renderUsers() {
 main.innerHTML=staffHeading()+`<div class="panel"><h2>Роль пользователя</h2><p class="hint">Укажите полный ID пользователя из заказа или его личного кабинета. Учётную запись сотрудник сначала создаёт через обычную регистрацию.</p><form id="role-form" class="form-stack"><div class="field"><label for="user-id">ID пользователя</label><input id="user-id" name="user_id" required placeholder="Полный идентификатор"></div><div class="field"><label for="role">Роль</label><select id="role" name="role"><option value="staff">Сотрудник</option><option value="admin">Администратор</option><option value="customer">Клиент</option></select></div><button class="button">Сохранить роль</button><div id="role-result"></div></form></div><div class="notice">Последнего администратора нельзя понизить. Изменения роли применяются и к уже открытой сессии.</div>`;
}
async function unitsModal(id) {
 const model=state.models.find(m=>m.id===id), data=await api('/models/'+id+'/units');
 showModal('Экземпляры инструмента',`<h3>${esc(model?.name)}</h3><div id="unit-list">${data.items.length?data.items.map(u=>`<div class="unit-row"><div><code>${esc(u.id)}</code><small class="hint"> · ${stateNames[u.state]||esc(u.state)}</small></div><select aria-label="Состояние экземпляра" data-unit="${esc(u.id)}" data-model="${esc(id)}">${Object.entries(stateNames).map(([key,label])=>`<option value="${key}" ${u.state===key?'selected':''}>${label}</option>`).join('')}</select></div>`).join(''):'<p class="hint">Экземпляров пока нет.</p>'}</div><div class="actions"><button class="button" data-action="add-unit" data-id="${esc(id)}">Добавить экземпляр +</button></div><p class="fineprint">Экземпляр с незавершёнными резервами нельзя отправить на обслуживание или списать.</p>`);
}
function modelModal(id) {
 const m=state.models.find(m=>m.id===id)||{};
 showModal(id?'Изменить модель':'Новая модель',`<form id="model-form" class="form-stack" data-id="${esc(id||'')}"><div class="field"><label for="model-name">Название</label><input id="model-name" name="name" value="${esc(m.name||'')}" required maxlength="150"></div><div class="field"><label for="model-category">Категория (код)</label><input id="model-category" name="category" list="categories" value="${esc(m.category||'drills')}" required><datalist id="categories">${Object.entries(categories).map(([key,label])=>`<option value="${key}">${label}</option>`).join('')}</datalist></div><div class="field"><label for="model-description">Описание</label><textarea id="model-description" name="description">${esc(m.description||'')}</textarea></div><div class="field"><label for="model-price">Стоимость за сутки, ₽</label><input id="model-price" name="price" type="number" min="0.01" max="1000000" step="0.01" value="${m.daily_price?Number(m.daily_price)/100:''}" required></div><div class="field"><label for="model-image">Изображение (HTTPS URL, необязательно)</label><input id="model-image" name="image" type="url" value="${esc(m.image_urls?.[0]||'')}" placeholder="https://…"></div><div id="form-error"></div><button class="button">Сохранить</button></form>`);
}
async function refreshUser() {if(state.token)state.user=await api('/me');}
async function render() {
 state.view=location.hash.slice(1)||'catalog';if(!['catalog','cart','orders','staff'].includes(state.view))state.view='catalog';navigation();
 const view=state.view;main.setAttribute('aria-busy','true');
 try {
  if(view==='catalog'){await loadCatalog();if(state.view!==view)return;renderCatalog();}
  if(view==='cart') {await loadAllModels();if(state.view!==view)return;renderCart();}
  if(view==='orders'||view==='staff') {
   if(!state.user){main.innerHTML=heading('ЛИЧНЫЙ КАБИНЕТ','Сначала войдите')+`<div class="panel"><p class="hint">Войдите, чтобы увидеть ваши заказы и доступные действия.</p><button class="button" data-action="login">Войти в кабинет</button></div>`;return;}
   await refreshUser();navigation();
   if(view==='staff'&&!staff()){main.innerHTML=empty('Нет доступа','Этот раздел доступен сотрудникам проката.');return;}
   if(view==='orders'||state.staffTab==='orders'){await loadOrders();if(state.view!==view)return;renderOrders(view==='staff');}
   else if(state.staffTab==='inventory'){await loadAllModels();if(state.view!==view)return;renderInventory();}
   else if(admin())renderUsers();else{state.staffTab='orders';await render();}
  }
 }catch(e){main.innerHTML=heading('ПРОКАТ ИНСТРУМЕНТА','Не получилось загрузить данные')+errorBox(e.message)+'<button class="button secondary" data-action="refresh">Попробовать снова</button>';}
 finally{main.removeAttribute('aria-busy');}
}
async function loadAllModels() {state.models=(await api('/models',{auth:false})).items||[];}
function cartFromForm() {const form=$('#cart-form');if(!form)return state.cart;const data=new FormData(form);return state.cart.map((i,index)=>({...i,start:data.get('start-'+index),end:data.get('end-'+index),quantity:Number(data.get('quantity-'+index))}));}
async function checkout() {
 if(!state.user){authModal();return;}
 const items=cartFromForm();if(!items.length||!items.every(validItem))throw new Error('Проверьте даты и количество в корзине.');
 if(!state.checkoutKey)await saveCart(items);
 if(!state.checkoutKey){state.checkoutKey=uid();write('rent.checkout',state.checkoutKey);}
 try {
  const order=await api('/orders',{method:'POST',body:{request_id:state.checkoutKey}});
  state.checkoutKey='';write('rent.checkout','');state.cart=(await api('/cart')).items||[];navigation();
  toast(order.state==='confirmed'?'Бронирование подтверждено.':'Заказ принят. Следите за его статусом.');location.hash='orders';await render();
 }catch(e){
  // Keep the key across an ambiguous network failure to prevent duplicate orders.
  if(e.status && e.status<500){state.checkoutKey='';write('rent.checkout','');}
  throw e;
 }
}
async function orderAction(action,id) {
 const order=state.orders.find(o=>o.id===id);if(!order)throw new Error('Обновите список заказов.');
 if(action==='cancel-order') {
  showModal('Отменить бронирование?',`<p class="hint">Резерв оборудования по заказу #${esc(id.slice(0,8))} будет освобождён.</p><form id="cancel-form" class="form-stack" data-id="${esc(id)}"><div class="field"><label for="reason">Причина ${order.user_id!==state.user.id?'(обязательно)':'(необязательно)'}</label><textarea id="reason" name="reason" ${order.user_id!==state.user.id?'required':''}></textarea></div><div id="form-error"></div><button class="button danger">Подтвердить отмену</button></form>`);return;
 }
 if(action==='issue-order'){await api(`/orders/${id}/issue`,{method:'POST',body:{}});toast('Выдача зарегистрирована.');}
 else {
  const ids=action==='return-all'?order.items.flatMap(q=>q.unit_ids.filter(id=>!q.returned_unit_ids.includes(id))):[...document.querySelectorAll('input[data-return]:checked')].filter(el=>el.dataset.return===id).map(el=>el.value);
  if(!ids.length)throw new Error('Отметьте экземпляры для возврата.');
  await api(`/orders/${id}/return`,{method:'POST',body:{unit_ids:[...new Set(ids)]}});toast('Возврат зарегистрирован.');
 }
 await render();
}
document.addEventListener('keydown',event=>{
 const card=event.target.closest('.card[data-action="product"]');
 if(card && (event.key==='Enter'||event.key===' ')){
  event.preventDefault();
  if(!event.repeat)card.click();
 }
});
document.addEventListener('click', async event => {
 const button=event.target.closest('[data-action]');if(!button)return;
 const {action,id,index,tab}=button.dataset;
 if(action==='close'){modal.close();return;}
 if(action==='login'||action==='register'){authModal(action==='register');return;}
 button.disabled=true;
 try {
  if(action==='logout'){state.token='';state.user=null;state.cart=read('rent.guest-cart',[]);state.checkoutKey='';write('rent.session','');write('rent.checkout','');navigation();await render();}
  else if(action==='product')await productModal(id);
  else if(action==='remove-item'){const items=cartFromForm();items.splice(Number(index),1);await saveCart(items);renderCart();}
  else if(action==='checkout')await checkout();
  else if(action==='refresh')await render();
  else if(action==='reset-filters'){event.preventDefault();state.filters={query:'',category:'',start:'',end:''};await render();}
  else if(['cancel-order','issue-order','return-order','return-all'].includes(action))await orderAction(action,id);
  else if(action==='staff-tab'){state.staffTab=tab;await render();}
  else if(action==='new-model')modelModal();
  else if(action==='edit-model')modelModal(id);
  else if(action==='units')await unitsModal(id);
  else if(action==='add-unit'){await api(`/models/${id}/units`,{method:'POST'});await unitsModal(id);toast('Экземпляр добавлен.');}
 }catch(e){toast(e.message,true);}finally{button.disabled=false;}
});
document.addEventListener('submit',async event=>{
 const form=event.target;if(!['auth-form','filters','product-form','cart-form','model-form','role-form','cancel-form'].includes(form.id))return;
 event.preventDefault();const submit=form.querySelector('button:not([type="button"])');if(submit)submit.disabled=true;const data=Object.fromEntries(new FormData(form));
 const error=$('#form-error',form);if(error)error.innerHTML='';
 try {
  if(form.id==='auth-form') {
   const guest=state.user?[]:state.cart;
   const session=await api('/auth/'+(form.dataset.register==='true'?'register':'login'),{method:'POST',body:data,auth:false});state.token=session.token;state.user=session.user;write('rent.session',session.token);
   const remote=(await api('/cart')).items||[];state.cart=remote;
   if(guest.length){await saveCart([...remote,...guest]);write('rent.guest-cart',[]);}
   modal.close();navigation();await render();toast('Вы вошли в кабинет.');
  }else if(form.id==='filters') {
   if(Boolean(data.start)!==Boolean(data.end)||data.start&&!(days(data)>0&&days(data)<=366))throw new Error('Укажите обе даты: возврат позже начала, период до 366 суток.');state.filters=data;await render();
  }else if(form.id==='product-form') {
   const item={model_id:state.modalItem.id,quantity:Number(data.quantity),start:data.start,end:data.end};if(!validItem(item))throw new Error('Проверьте период и количество.');await saveCart([...state.cart,item]);modal.close();toast('Инструмент добавлен в корзину.');
  }else if(form.id==='cart-form') {
   const items=cartFromForm();if(!items.every(validItem))throw new Error('Проверьте даты и количество.');await saveCart(items);renderCart();toast('Корзина сохранена.');
  }else if(form.id==='model-form') {
   if(data.image&&!/^https:\/\//i.test(data.image))throw new Error('Изображение должно быть доступно по HTTPS.');
   const payload={name:data.name,category:data.category,description:data.description,daily_price:String(Math.round(Number(data.price)*100)),image_urls:data.image?[data.image]:[]};const id=form.dataset.id;await api('/models'+(id?'/'+id:''),{method:id?'PUT':'POST',body:payload});modal.close();await render();toast('Модель сохранена.');
  }else if(form.id==='role-form') {
   const u=await api(`/users/${encodeURIComponent(data.user_id.trim())}/role`,{method:'PATCH',body:{role:data.role}});$('#role-result').textContent=`${u.email}: ${roleNames[u.role]}`;toast('Роль обновлена.');await refreshUser();navigation();if(!admin()){state.staffTab='orders';await render();}
  }else if(form.id==='cancel-form') {
   await api(`/orders/${form.dataset.id}/cancel`,{method:'POST',body:{reason:data.reason}});modal.close();await render();toast('Бронирование отменено.');
  }
 }catch(e){if(error&&error.isConnected)error.innerHTML=errorBox(e.message);else toast(e.message,true);}finally{if(submit)submit.disabled=false;}
});
document.addEventListener('input',event=>{
 if(event.target.closest('#product-form'))updateProductEstimate();
 if(event.target.closest('#cart-form')) {
  const items=cartFromForm();items.forEach((item,index)=>{const el=document.querySelector(`[data-estimate="${index}"]`);if(el)el.textContent=validItem(item)?money(estimate(item)):'—';});
  const total=$('#cart-total');if(total)total.textContent=items.every(validItem)?money(items.reduce((sum,item)=>sum+estimate(item),0)):'—';
 }
});
document.addEventListener('change',async event=>{
 if(event.target.closest('#product-form'))await updateCalendar();
 if(event.target.dataset.unit){const el=event.target;el.disabled=true;try{await api('/units/'+el.dataset.unit,{method:'PATCH',body:{state:el.value}});toast('Состояние обновлено.');}catch(e){toast(e.message,true);}finally{el.disabled=false;try{await unitsModal(el.dataset.model);}catch(e){toast(e.message,true);}}}
});
modal.addEventListener('click',event=>{if(event.target===modal){const r=modal.getBoundingClientRect();if(event.clientX<r.left||event.clientX>r.right||event.clientY<r.top||event.clientY>r.bottom)modal.close();}});
window.addEventListener('hashchange',()=>{if(state.view==='catalog'&&location.hash==='#catalog')state.filters={query:'',category:'',start:'',end:''};render();});
setInterval(async()=>{if(!state.user||modal.open||!['orders','staff'].includes(state.view)||state.view==='staff'&&state.staffTab!=='orders')return;if(!state.orders.some(o=>['pending','cancelling','issuing','returning'].includes(o.state)))return;try{await loadOrders();renderOrders(state.view==='staff');}catch{}},3000);
async function init(){state.cart=read('rent.guest-cart',[]);if(state.token){try{await refreshUser();state.cart=(await api('/cart')).items||[];}catch(e){toast(e.message,true);}}navigation();await render();}
init();
