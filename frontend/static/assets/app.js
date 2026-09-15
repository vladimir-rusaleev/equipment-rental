const $ = (selector, root = document) => root.querySelector(selector);
const main = $('#main');
const esc = (value = '') => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const money = value => new Intl.NumberFormat('ru-RU', {style:'currency',currency:'RUB',minimumFractionDigits:0,maximumFractionDigits:2}).format(Number(value || 0) / 100);
const categories = {'rotary-hammers':'Перфораторы',jigsaws:'Лобзики',drills:'Дрели и шуруповёрты',grinders:'Шлифмашины',saws:'Пилы',vacuums:'Пылесосы',levels:'Измерительный инструмент',welders:'Сварочные аппараты'};
const date = value => value ? new Intl.DateTimeFormat('ru-RU',{day:'numeric',month:'short'}).format(new Date(value+'T12:00:00')) : '—';
const state = {models:[], filters:{query:'',category:'',start:'',end:''}};
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
function heading(eyebrow,title,subtitle='',extra='') {return `<div class="page-heading"><div><h1>${title}</h1>${subtitle?`<p>${subtitle}</p>`:""}</div>${extra}</div>`;}
function renderCatalog() {
 const f=state.filters;
 const known=new Set([...Object.keys(categories),...state.models.map(m=>m.category)]);
 main.innerHTML=heading('','Каталог инструментов','Укажите даты, чтобы проверить наличие инструмента.')+
 ` <form id="filters" class="filters"><div class="field"><label for="search">Поиск по каталогу</label><input id="search" name="query" placeholder="Найти инструмент…" value="${esc(f.query)}"></div><div class="field"><label for="category">Категория</label><select id="category" name="category"><option value="">Все инструменты</option>${[...known].map(c=>`<option value="${esc(c)}" ${f.category===c?'selected':''}>${esc(categories[c]||c)}</option>`).join('')}</select></div><div class="field"><label for="filter-start">Начало аренды</label><input id="filter-start" name="start" type="date" value="${esc(f.start)}" ${f.end?'required':''}></div><div class="field"><label for="filter-end">Возврат</label><input id="filter-end" name="end" type="date" value="${esc(f.end)}" ${f.start?'required':''}></div><button class="button">Найти</button></form>
 <div class="section-line"><span>Найдено моделей: <b>${state.models.length}</b></span><span>${f.start?`${date(f.start)} — ${date(f.end)} · дата возврата не оплачивается`:'Стоимость указана за сутки'}</span></div>
 ${state.models.length?`<div class="cards">${state.models.map(m=>`<article class="card">${visual(m)}<div class="card-body"><div class="category">${esc(categories[m.category]||m.category)}</div><h3>${esc(m.name)}</h3><p class="card-description">${esc(m.description||'Описание не добавлено.')}</p><div class="availability"><span class="status-dot"></span>${f.start?`Свободно: ${m.available} шт.`:'Укажите даты для проверки наличия'}</div><div class="card-bottom"><span class="price">${money(m.daily_price)}<small> / сутки</small></span></div></div></article>`).join('')}</div>`:'<div class="empty"><h2>Ничего не нашлось</h2><p>Попробуйте изменить категорию, запрос или даты.</p></div>'}`;
}

let catalogSequence=0;
async function loadCatalog() {
 const sequence=++catalogSequence;
 const q=new URLSearchParams();
 Object.entries(state.filters).forEach(([key,value])=>{if(value)q.set(key,value);});
 try {
  const response=await fetch('/api/v1/models?'+q,{signal:AbortSignal.timeout(10000)});
  if(!response.ok)throw new Error('Ошибка загрузки каталога');
  const data=await response.json();
  if(sequence!==catalogSequence)return;
  state.models=data.items||[];
  renderCatalog();
 } catch {
  if(sequence!==catalogSequence)return;
  state.models=[];
  renderCatalog();
  $('#filters').insertAdjacentHTML('afterend','<p class="notice error">Не удалось загрузить каталог. Проверьте выбранные даты или попробуйте ещё раз.</p>');
 }
}

main.addEventListener('submit',event=>{
 if(event.target.id!=='filters')return;
 event.preventDefault();
 state.filters=Object.fromEntries(new FormData(event.target));
 loadCatalog();
});
main.addEventListener('change',event=>{
 if(!['filter-start','filter-end'].includes(event.target.id))return;
 const start=$('#filter-start'),end=$('#filter-end');
 start.required=Boolean(end.value);
 end.required=Boolean(start.value);
});
loadCatalog();
