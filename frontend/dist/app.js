const state = {
  user: null, page: "dashboard", clients: [], requests: [], orders: [], interactions: [], users: [], report: null,
};
const pages = {
  dashboard: ["Обзор", "Сводка состояния клиентской базы"],
  clients: ["Клиенты", "Карточки, контакты и статусы клиентов"],
  requests: ["Заявки", "Обращения клиентов и контроль обработки"],
  orders: ["Заказы", "Оказанные услуги и учет выполнения"],
  interactions: ["Взаимодействия", "История контактов с клиентами"],
  reports: ["Отчеты", "Показатели работы за выбранный период"],
  users: ["Сотрудники", "Учетные записи и доступ к системе"],
};
const $ = (selector) => document.querySelector(selector);
const content = $("#content");

function api(name, ...args) {
  return window.go.main.App[name](...args);
}
function esc(value = "") {
  return String(value ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
}
function money(value) {
  return new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB", maximumFractionDigits: 2 }).format(value || 0);
}
function today() { return new Date().toISOString().slice(0, 10); }
function monthAgo() { const d = new Date(); d.setMonth(d.getMonth() - 1); return d.toISOString().slice(0, 10); }
function statusClass(value) {
  if (["Активный", "Выполнен", "Выполнена", "Закрыта"].includes(value)) return "success";
  if (["В работе", "Новый", "Новая"].includes(value)) return "warning";
  if (["Отменен", "Черный список"].includes(value)) return "danger";
  return "";
}
function badge(value) { return `<span class="status ${statusClass(value)}">${esc(value)}</span>`; }
function toast(text, error = false) {
  const item = document.createElement("div");
  item.className = `toast ${error ? "error" : ""}`;
  item.textContent = text;
  $("#toastRoot").appendChild(item);
  setTimeout(() => item.remove(), 3500);
}
async function guarded(action) {
  try { return await action(); } catch (error) { toast(String(error), true); throw error; }
}
function head(title, note, actions = "") {
  return `<div class="page-head"><div><h1>${title}</h1><p>${note}</p></div><div class="actions">${actions}</div></div>`;
}
function empty(columns, label = "Данные пока не добавлены") {
  return `<tr><td colspan="${columns}"><div class="empty">${label}</div></td></tr>`;
}

async function setPage(page) {
  state.page = page;
  document.querySelectorAll(".nav").forEach(item => item.classList.toggle("active", item.dataset.page === page));
  $("#pageTitle").textContent = pages[page][0];
  $("#pageDescription").textContent = pages[page][1];
  await renderPage();
}
async function renderPage() {
  const renderers = { dashboard: renderDashboard, clients: renderClients, requests: renderRequests, orders: renderOrders, interactions: renderInteractions, reports: renderReports, users: renderUsers };
  await guarded(renderers[state.page]);
}

async function renderDashboard() {
  const data = await api("Dashboard");
  const activity = (data.recentInteractions || []).map(i => `
    <div class="activity-row"><div><strong>${esc(i.type)} - ${esc(i.clientName)}</strong><span>${esc(i.employee)}</span></div><span>${esc(i.date)}</span></div>`).join("") || `<div class="empty">Контакты еще не зарегистрированы</div>`;
  const requests = (data.recentRequests || []).map(r => `
    <div class="activity-row"><div><strong>${esc(r.theme)}</strong><span>${esc(r.clientName)}</span></div>${badge(r.status)}</div>`).join("") || `<div class="empty">Активных заявок пока нет</div>`;
  content.innerHTML = head("Рабочий обзор", "Актуальная информация по работе с клиентами",
    `<button class="btn primary" onclick="openClientForm()">+ Новый клиент</button>`) + `
    <div class="metric-grid">
      ${metric("Клиенты", data.clients, "в базе", "◯")}
      ${metric("Открытые заявки", data.openRequests, "ожидают обработки", "✉")}
      ${metric("Заказы", data.orders, "зарегистрировано", "▣")}
      ${metric("Выручка", money(data.revenue), "выполненные заказы", "₽")}
    </div>
    <div class="grid-two">
      <section class="card"><div class="card-head"><div><h3>Последние заявки</h3><p>Требуют внимания менеджера</p></div></div><div class="activity">${requests}</div></section>
      <section class="card"><div class="card-head"><div><h3>История контактов</h3><p>Недавняя активность</p></div></div><div class="activity">${activity}</div></section>
    </div>`;
}
function metric(title, value, note, icon) {
  return `<div class="metric"><div class="metric-top"><span>${title}</span><span class="metric-icon">${icon}</span></div><div class="metric-value">${esc(value)}</div><div class="metric-foot">${note}</div></div>`;
}

async function renderClients(search = "", filter = "") {
  state.clients = (await api("Clients", search, filter)) || [];
  content.innerHTML = head("Клиенты", "Единая база контактных данных и статусов",
    `<button class="btn primary" onclick="openClientForm()">+ Добавить клиента</button>`) + `
    <div class="toolbar"><div class="filters">
      <input id="clientSearch" value="${esc(search)}" placeholder="Поиск по ФИО, телефону или e-mail" />
      <select id="clientStatus">${options(["", "Новый", "Активный", "Неактивный", "Черный список"], filter, "Все статусы")}</select>
    </div><span class="pill">${state.clients.length} записей</span></div>
    <div class="data-card"><table><thead><tr><th>КЛИЕНТ</th><th>КОНТАКТЫ</th><th>РЕГИСТРАЦИЯ</th><th>СТАТУС</th><th></th></tr></thead>
    <tbody>${state.clients.map(c => `<tr><td class="name"><strong>${esc(fullName(c))}</strong><small>${esc(c.address || "Адрес не указан")}</small></td><td>${esc(c.phone)}<br><span class="muted">${esc(c.email)}</span></td><td>${esc(c.registrationDate)}</td><td>${badge(c.status)}</td><td class="row-actions"><button class="mini" onclick="openClientForm(${c.id})">Открыть</button><button class="mini danger" onclick="removeClient(${c.id})">Удалить</button></td></tr>`).join("") || empty(5)}</tbody></table></div>`;
  $("#clientSearch").addEventListener("input", e => renderClients(e.target.value, $("#clientStatus").value));
  $("#clientStatus").addEventListener("change", e => renderClients($("#clientSearch").value, e.target.value));
}
function fullName(c) { return [c.lastName, c.firstName, c.patronymic].filter(Boolean).join(" "); }

async function renderRequests(filter = "") {
  state.requests = (await api("Requests", filter)) || [];
  content.innerHTML = head("Заявки", "Обрабатывайте обращения и фиксируйте результат",
    `<button class="btn primary" onclick="openRequestForm()">+ Новая заявка</button>`) + `
    <div class="toolbar"><div class="filters"><select id="requestStatus">${options(["", "Новая", "В работе", "Выполнена", "Закрыта"], filter, "Все статусы")}</select></div><span class="pill">${state.requests.length} заявок</span></div>
    <div class="data-card"><table><thead><tr><th>ТЕМА / КЛИЕНТ</th><th>ДАТА</th><th>СТАТУС</th><th>КОММЕНТАРИЙ</th><th></th></tr></thead>
    <tbody>${state.requests.map(r => `<tr><td class="name"><strong>${esc(r.theme)}</strong><small>${esc(r.clientName)}</small></td><td>${esc(r.createDate)}</td><td>${badge(r.status)}</td><td>${esc(r.employeeComment || "Нет комментария")}</td><td class="row-actions"><button class="mini" onclick="openRequestForm(${r.id})">Изменить</button><button class="mini danger" onclick="removeRequest(${r.id})">Удалить</button></td></tr>`).join("") || empty(5)}</tbody></table></div>`;
  $("#requestStatus").addEventListener("change", e => renderRequests(e.target.value));
}

async function renderOrders(filter = "") {
  state.orders = (await api("Orders", filter)) || [];
  content.innerHTML = head("Заказы", "Услуги, стоимость и контроль выполнения",
    `<button class="btn primary" onclick="openOrderForm()">+ Новый заказ</button>`) + `
    <div class="toolbar"><div class="filters"><select id="orderStatus">${options(["", "Новый", "В работе", "Выполнен", "Отменен"], filter, "Все статусы")}</select></div><span class="pill">${state.orders.length} заказов</span></div>
    <div class="data-card"><table><thead><tr><th>УСЛУГА / КЛИЕНТ</th><th>ДАТА</th><th>СТОИМОСТЬ</th><th>СТАТУС</th><th></th></tr></thead>
    <tbody>${state.orders.map(o => `<tr><td class="name"><strong>${esc(o.serviceName)}</strong><small>${esc(o.clientName)}</small></td><td>${esc(o.orderDate)}</td><td><strong>${money(o.amount)}</strong></td><td>${badge(o.status)}</td><td class="row-actions"><button class="mini" onclick="openOrderForm(${o.id})">Изменить</button><button class="mini danger" onclick="removeOrder(${o.id})">Удалить</button></td></tr>`).join("") || empty(5)}</tbody></table></div>`;
  $("#orderStatus").addEventListener("change", e => renderOrders(e.target.value));
}

async function renderInteractions() {
  state.interactions = (await api("Interactions")) || [];
  content.innerHTML = head("Взаимодействия", "Полная история коммуникаций с клиентами",
    `<button class="btn primary" onclick="openInteractionForm()">+ Добавить контакт</button>`) + `
    <div class="toolbar"><p class="muted">Звонки, консультации, встречи и переписка</p><span class="pill">${state.interactions.length} записей</span></div>
    <div class="data-card"><table><thead><tr><th>ТИП / КЛИЕНТ</th><th>ДАТА</th><th>СОТРУДНИК</th><th>РЕЗУЛЬТАТ</th><th></th></tr></thead>
    <tbody>${state.interactions.map(i => `<tr><td class="name"><strong>${esc(i.type)}</strong><small>${esc(i.clientName)}</small></td><td>${esc(i.date)}</td><td>${esc(i.employee)}</td><td>${esc(i.description)}</td><td class="row-actions"><button class="mini" onclick="openInteractionForm(${i.id})">Изменить</button><button class="mini danger" onclick="removeInteraction(${i.id})">Удалить</button></td></tr>`).join("") || empty(5)}</tbody></table></div>`;
}

async function renderReports() {
  const from = state.report?.from || monthAgo(), to = state.report?.to || today();
  content.innerHTML = head("Отчеты", "Формирование статистики и выгрузка данных") + `
  <div class="reports">
    <section class="card report-form"><h3>Период отчета</h3><label>Дата начала<input id="fromDate" type="date" value="${from}"></label><label>Дата окончания<input id="toDate" type="date" value="${to}"></label>
      <button class="btn primary block" onclick="generateReport()">Сформировать отчет</button>
      <button class="btn ghost block" onclick="exportReport()">Экспортировать CSV</button>
    </section>
    <section class="card report-view" id="reportResult">${reportMarkup(state.report)}</section>
  </div>`;
}
function reportMarkup(r) {
  if (!r) return `<div class="empty">Выберите период и сформируйте отчет</div>`;
  return `<div class="card-head"><div><h3>Сводка ClientSys</h3><p>${esc(r.from)} - ${esc(r.to)}</p></div></div>
    ${reportStat("Новые клиенты", r.clients)}${reportStat("Заявки всего", r.requests)}${reportStat("Открытые заявки", r.openRequests)}${reportStat("Заказы", r.orders)}${reportStat("Выполненные заказы", r.completedOrders)}${reportStat("Выручка", money(r.revenue))}${reportStat("Взаимодействия", r.interactions)}`;
}
function reportStat(title, value) { return `<div class="report-stat"><span>${title}</span><strong>${esc(value)}</strong></div>`; }
async function generateReport() {
  state.report = await guarded(() => api("Report", $("#fromDate").value, $("#toDate").value));
  $("#reportResult").innerHTML = reportMarkup(state.report);
}
async function exportReport() {
  if (!state.report) { toast("Сначала сформируйте отчет", true); return; }
  const path = await guarded(() => api("ExportReport", state.report));
  if (path) toast("Отчет сохранен");
}

async function renderUsers() {
  state.users = (await api("Users")) || [];
  const addButton = state.user.role === "admin" ? `<button class="btn primary" onclick="openRegister()">+ Добавить сотрудника</button>` : "";
  content.innerHTML = head("Сотрудники", "Управление учетными записями системы",
    addButton) + `
    <div class="data-card"><table><thead><tr><th>СОТРУДНИК</th><th>ЛОГИН</th><th>E-MAIL</th><th>РОЛЬ</th><th>СОЗДАН</th></tr></thead>
    <tbody>${state.users.map(u => `<tr><td class="name"><strong>${esc(u.fullName)}</strong></td><td>${esc(u.login)}</td><td>${esc(u.email)}</td><td>${badge(u.role)}</td><td>${esc(u.createdAt)}</td></tr>`).join("") || empty(5)}</tbody></table></div>`;
}
function options(items, selected, first) {
  return items.map((item, index) => `<option value="${esc(item)}" ${item === selected ? "selected" : ""}>${esc(index === 0 ? first : item)}</option>`).join("");
}

function openModal(title, fields, onSubmit, wide = false) {
  $("#modalRoot").innerHTML = `<div class="modal-backdrop"><div class="modal ${wide ? "wide" : ""}"><div class="modal-head"><h3>${title}</h3><button class="close" type="button" onclick="closeModal()">×</button></div><form id="modalForm"><div class="fields">${fields}</div><div class="modal-actions"><button class="btn ghost" type="button" onclick="closeModal()">Отмена</button><button class="btn primary" type="submit">Сохранить</button></div></form></div></div>`;
  $("#modalForm").addEventListener("submit", async event => {
    event.preventDefault();
    try { await onSubmit(new FormData(event.target)); closeModal(); toast("Данные сохранены"); await renderPage(); } catch (_) {}
  });
}
function closeModal() { $("#modalRoot").innerHTML = ""; }
function field(label, name, value = "", required = false, cls = "") {
  return `<label class="${cls}">${label}<input name="${name}" value="${esc(value)}" ${required ? "required" : ""}></label>`;
}
function passwordField(label, name, required = false, cls = "") {
  return `<label class="${cls}">${label}<input type="password" name="${name}" ${required ? "required" : ""}></label>`;
}
function selectField(label, name, values, selected, cls = "") {
  return `<label class="${cls}">${label}<select name="${name}">${values.map(v => `<option ${v === selected ? "selected" : ""}>${v}</option>`).join("")}</select></label>`;
}
function textField(label, name, value = "") {
  return `<label class="full">${label}<textarea name="${name}">${esc(value)}</textarea></label>`;
}
async function ensureClients() {
  state.clients = (await api("Clients", "", "")) || [];
  if (!state.clients.length) { toast("Сначала добавьте клиента", true); return false; }
  return true;
}
function clientSelect(current) {
  return `<label class="full">Клиент<select name="clientId" required>${state.clients.map(c => `<option value="${c.id}" ${c.id === current ? "selected" : ""}>${esc(fullName(c))}</option>`).join("")}</select></label>`;
}

function openClientForm(id) {
  const c = state.clients.find(item => item.id === id) || { registrationDate: today(), status: "Новый" };
  openModal(id ? "Карточка клиента" : "Новый клиент",
    field("Фамилия *", "lastName", c.lastName, true) + field("Имя *", "firstName", c.firstName, true) +
    field("Отчество", "patronymic", c.patronymic) + field("Телефон *", "phone", c.phone, true) +
    field("E-mail", "email", c.email) + field("Адрес", "address", c.address) +
    field("Дата регистрации", "registrationDate", c.registrationDate, true) +
    selectField("Статус", "status", ["Новый", "Активный", "Неактивный", "Черный список"], c.status) +
    textField("Дополнительная информация", "comment", c.comment),
    form => api("SaveClient", { ...c, ...Object.fromEntries(form) }), true);
}
async function removeClient(id) {
  if (!confirm("Удалить клиента вместе со связанными заявками и заказами?")) return;
  await guarded(() => api("DeleteClient", id)); toast("Клиент удален"); await renderClients();
}
async function openRequestForm(id) {
  if (!await ensureClients()) return;
  const r = state.requests.find(item => item.id === id) || { createDate: today(), status: "Новая" };
  openModal(id ? "Редактирование заявки" : "Новая заявка", clientSelect(r.clientId) +
    field("Тема обращения *", "theme", r.theme, true, "full") + field("Дата", "createDate", r.createDate, true) +
    selectField("Статус", "status", ["Новая", "В работе", "Выполнена", "Закрыта"], r.status) +
    textField("Описание проблемы", "description", r.description) + textField("Комментарий сотрудника", "employeeComment", r.employeeComment),
    form => api("SaveRequest", { ...r, ...Object.fromEntries(form), clientId: Number(form.get("clientId")) }), true);
}
async function removeRequest(id) {
  if (!confirm("Удалить заявку?")) return;
  await guarded(() => api("DeleteRequest", id)); toast("Заявка удалена"); await renderRequests();
}
async function openOrderForm(id) {
  if (!await ensureClients()) return;
  const o = state.orders.find(item => item.id === id) || { orderDate: today(), status: "Новый", amount: 0 };
  openModal(id ? "Редактирование заказа" : "Новый заказ", clientSelect(o.clientId) +
    field("Наименование услуги *", "serviceName", o.serviceName, true, "full") + field("Стоимость, руб. *", "amount", o.amount, true) +
    field("Дата оформления", "orderDate", o.orderDate, true) + selectField("Статус", "status", ["Новый", "В работе", "Выполнен", "Отменен"], o.status),
    form => api("SaveOrder", { ...o, ...Object.fromEntries(form), clientId: Number(form.get("clientId")), amount: Number(String(form.get("amount")).replace(",", ".")) }));
}
async function removeOrder(id) {
  if (!confirm("Удалить заказ?")) return;
  await guarded(() => api("DeleteOrder", id)); toast("Заказ удален"); await renderOrders();
}
async function openInteractionForm(id) {
  if (!await ensureClients()) return;
  const i = state.interactions.find(item => item.id === id) || { date: today(), type: "Звонок" };
  openModal(id ? "Редактирование контакта" : "Новый контакт", clientSelect(i.clientId) +
    selectField("Тип взаимодействия", "type", ["Звонок", "Письмо", "Консультация", "Встреча", "Комментарий"], i.type) +
    field("Дата", "date", i.date, true) + textField("Результат взаимодействия *", "description", i.description),
    form => api("SaveInteraction", { ...i, ...Object.fromEntries(form), clientId: Number(form.get("clientId")) }));
}
async function removeInteraction(id) {
  if (!confirm("Удалить запись взаимодействия?")) return;
  await guarded(() => api("DeleteInteraction", id)); toast("Запись удалена"); await renderInteractions();
}
function openRegister() {
  openModal("Регистрация сотрудника",
    field("ФИО *", "fullName", "", true, "full") + field("Логин *", "login", "", true) + field("E-mail *", "email", "", true) +
    passwordField("Пароль *", "password", true) + selectField("Роль", "role", ["manager", "admin"], "manager"),
    form => { const value = Object.fromEntries(form); return api("Register", { fullName: value.fullName, login: value.login, email: value.email, role: value.role }, value.password); });
}
function openPassword() {
  openModal("Смена пароля", passwordField("Текущий пароль", "current", true, "full") + passwordField("Новый пароль", "next", true) + passwordField("Повтор нового пароля", "repeat", true),
    form => { if (form.get("next") !== form.get("repeat")) throw new Error("Новые пароли не совпадают"); return api("ChangePassword", form.get("current"), form.get("next")); });
}

async function login(event) {
  event.preventDefault();
  state.user = await guarded(() => api("Login", $("#loginInput").value, $("#passwordInput").value));
  $("#loginScreen").classList.add("hidden");
  $("#workspace").classList.remove("hidden");
  $("#userName").textContent = state.user.fullName;
  $("#userRole").textContent = state.user.role;
  $("#avatar").textContent = state.user.fullName.slice(0, 1).toUpperCase();
  await setPage("dashboard");
}
async function logout() {
  await api("Logout");
  state.user = null;
  $("#workspace").classList.add("hidden");
  $("#loginScreen").classList.remove("hidden");
  $("#passwordInput").value = "";
}
async function waitForBridge() {
  if (window.go?.main?.App) return;
  await new Promise(resolve => setTimeout(resolve, 50));
  return waitForBridge();
}
async function init() {
  await waitForBridge();
  $("#loginForm").addEventListener("submit", login);
  $("#navigation").addEventListener("click", event => {
    const button = event.target.closest("[data-page]");
    if (button) setPage(button.dataset.page);
  });
  $("#refreshBtn").addEventListener("click", () => renderPage());
  $("#passwordBtn").addEventListener("click", openPassword);
  $("#logoutBtn").addEventListener("click", logout);
  $("#backupBtn").addEventListener("click", async () => {
    const result = await guarded(() => api("BackupDatabase"));
    if (result) toast("Резервная копия сохранена");
  });
}
init();
