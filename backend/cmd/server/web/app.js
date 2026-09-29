// Состояние приложения
const state = {
    token: localStorage.getItem('masterbook_token') || '',
    me: null,
    selectedMaster: null,
    services: [],
    appointmentsRaw: [],
    appointmentsFilter: 'active',
    masterAppointmentsRaw: [],
    masterAppointmentsFilter: 'active',
    adminMasters: [],
    selectedAdminMasterId: null,
    editingServiceId: null,
};

// Хелперы для выборки элементов
const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];

// Показать всплывающее сообщение
function flash(msg, type = 'success') {
    const el = $('#flash');
    el.className = `flash ${type}`;
    el.textContent = msg;

    setTimeout(() => {
        el.textContent = '';
        el.className = '';
    }, 3500);
}

// Универсальная обёртка над fetch
async function api(path, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...(options.headers || {}),
    };

    if (state.token) {
        headers.Authorization = `Bearer ${state.token}`;
    }

    const res = await fetch(path, { ...options, headers });
    const text = await res.text();

    let data = {};
    try {
        data = text ? JSON.parse(text) : {};
    } catch {
        data = { error: text };
    }

    if (!res.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
    }

    return data;
}

// Загрузка файла (multipart) — api() не подходит, она всегда шлёт JSON
async function uploadFile(path, formData, method = 'POST') {
    const headers = {};
    if (state.token) {
        headers.Authorization = `Bearer ${state.token}`;
    }

    const res = await fetch(path, { method, headers, body: formData });
    const text = await res.text();

    let data = {};
    try {
        data = text ? JSON.parse(text) : {};
    } catch {
        data = { error: text };
    }

    if (!res.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
    }

    return data;
}

// Статус записи -> под какую вкладку она попадает
function matchesStatusFilter(status, filter) {
    if (filter === 'active') return ['pending', 'confirmed'].includes(status);
    if (filter === 'archived') return ['completed', 'cancelled'].includes(status);
    return status === filter;
}

// Переключить экран: авторизация или приложение
function setLoggedIn() {
    const auth = $('#auth-view');
    const app = $('#app-view');

    if (state.token) {
        auth.classList.add('hidden');
        app.classList.remove('hidden');
    } else {
        auth.classList.remove('hidden');
        app.classList.add('hidden');
    }
}

// Навигация (имя пользователя + кнопка выхода)
function renderNav() {
    const nav = $('#nav');

    nav.innerHTML = state.me
        ? `<span>${state.me.name} · ${state.me.role}</span><button id="logout" class="secondary">Выйти</button>`
        : '';

    $('#logout')?.addEventListener('click', () => {
        state.token = '';
        state.me = null;
        localStorage.removeItem('masterbook_token');
        setLoggedIn();
        renderNav();
    });
}

// Загрузить текущего пользователя
async function loadMe() {
    if (!state.token) {
        setLoggedIn();
        return;
    }

    try {
        state.me = await api('/api/v1/me');
        setLoggedIn();
        renderNav();
        await refreshAll();
    } catch {
        state.token = '';
        localStorage.removeItem('masterbook_token');
        setLoggedIn();
    }
}

// Авторизация
async function login(e) {
    e.preventDefault();
    const f = new FormData(e.target);

    try {
        const r = await api('/api/v1/auth/login', {
            method: 'POST',
            body: JSON.stringify({
                email: f.get('email'),
                password: f.get('password'),
            }),
        });

        state.token = r.access_token;
        localStorage.setItem('masterbook_token', state.token);

        await loadMe();
        flash('Вы вошли в систему');
    } catch (err) {
        $('#auth-status').innerHTML = `<div class="flash error">${err.message}</div>`;
    }
}

// Регистрация
async function register(e) {
    e.preventDefault();
    const f = new FormData(e.target);

    try {
        await api('/api/v1/auth/register', {
            method: 'POST',
            body: JSON.stringify({
                name: f.get('name'),
                email: f.get('email'),
                password: f.get('password'),
            }),
        });

        flash('Аккаунт создан. Теперь войдите');
        $$('.tab').find(x => x.dataset.tab === 'login').click();
    } catch (err) {
        $('#auth-status').innerHTML = `<div class="flash error">${err.message}</div>`;
    }
}

// Список мастеров
async function loadMasters() {
    const masters = await api('/api/v1/masters');
    const box = $('#masters');

    box.innerHTML = masters.length
        ? masters.map(m => `
            <div class="item">
                <div class="item-head">
                    <strong>${escapeHtml(m.name)}</strong>
                    <span class="badge">ID ${m.id}</span>
                </div>
                <div class="meta">${escapeHtml(m.description || 'Описание отсутствует')}</div>
                <div class="actions">
                    <button class="primary" data-book="${m.id}">Записаться</button>
                </div>
            </div>
        `).join('')
        : '<div class="muted">Мастеров пока нет.</div>';

    $$('[data-book]').forEach(btn => {
        btn.onclick = () => openBooking(
            Number(btn.dataset.book),
            masters.find(m => m.id === Number(btn.dataset.book))
        );
    });
}

// Открыть форму записи к мастеру
async function openBooking(id, master) {
    state.selectedMaster = master;
    $('#booking-card').classList.remove('hidden');
    $('#selected-master').textContent = `Мастер: ${master.name}`;

    state.services = await api(`/api/v1/masters/${id}/services`);

    $('#service-select').innerHTML = state.services
        .map(s => `<option value="${s.id}">${escapeHtml(s.name)} — ${s.price} ₽ — ${s.duration_minutes} мин</option>`)
        .join('');

    // По умолчанию — завтра
    const d = new Date();
    d.setDate(d.getDate() + 1);
    $('#date-input').value = d.toISOString().slice(0, 10);

    await loadSlots();
}

// Свободные слоты на выбранную дату
async function loadSlots() {
    if (!state.selectedMaster) return;

    const serviceId = Number($('#service-select').value);
    const date = $('#date-input').value;
    if (!serviceId || !date) return;

    try {
        const slots = await api(
            `/api/v1/masters/${state.selectedMaster.id}/availability?date=${encodeURIComponent(date)}&service_id=${serviceId}`
        );

        const box = $('#slots');
        box.innerHTML = slots.length
            ? slots.map(s => `
                <button class="slot" data-slot="${s.start_time}">
                    ${new Date(s.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </button>
            `).join('')
            : '<div class="muted">Свободных слотов нет.</div>';

        $$('[data-slot]').forEach(b => {
            b.onclick = () => createAppointment(b.dataset.slot, serviceId);
        });
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Создать запись
async function createAppointment(startTime, serviceId) {
    try {
        await api('/api/v1/appointments', {
            method: 'POST',
            body: JSON.stringify({
                master_id: state.selectedMaster.id,
                service_id: serviceId,
                start_time: startTime,
            }),
        });

        flash('Запись создана');
        await refreshAppointments();
        await loadSlots();
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Мои записи
async function refreshAppointments() {
    state.appointmentsRaw = await api('/api/v1/appointments');
    renderAppointments();
}

function renderAppointments() {
    const items = state.appointmentsRaw.filter(a => matchesStatusFilter(a.status, state.appointmentsFilter));

    $('#appointments').innerHTML = items.length
        ? items.map(a => `
            <div class="item">
                <div class="item-head">
                    <strong>${escapeHtml(a.master_name)} · ${escapeHtml(a.service_name)}</strong>
                    <span class="badge">${a.status}</span>
                </div>
                <div class="meta">${new Date(a.start_time).toLocaleString()}</div>
                ${['pending', 'confirmed'].includes(a.status)
                    ? `<div class="actions"><button class="danger" data-cancel="${a.id}">Отменить</button></div>`
                    : ''}
            </div>
        `).join('')
        : '<div class="muted">Записей нет.</div>';

    $$('[data-cancel]').forEach(btn => {
        btn.onclick = async () => {
            try {
                await api(`/api/v1/appointments/${btn.dataset.cancel}/cancel`, { method: 'PATCH' });
                await refreshAppointments();
                flash('Запись отменена');
            } catch (err) {
                flash(err.message, 'error');
            }
        };
    });
}

// Уведомления
async function refreshNotifications() {
    const items = await api('/api/v1/notifications');

    $('#notifications').innerHTML = items.length
        ? items.map(n => `
            <div class="item">
                <div class="item-head">
                    <strong>${escapeHtml(n.title)}</strong>
                    <span class="badge">${n.is_read ? 'прочитано' : 'новое'}</span>
                </div>
                <div class="meta">${escapeHtml(n.message)}</div>
                ${!n.is_read
                    ? `<div class="actions"><button class="secondary" data-read="${n.id}">Отметить прочитанным</button></div>`
                    : ''}
            </div>
        `).join('')
        : '<div class="muted">Уведомлений нет.</div>';

    $$('[data-read]').forEach(btn => {
        btn.onclick = async () => {
            try {
                await api(`/api/v1/notifications/${btn.dataset.read}/read`, { method: 'PATCH' });
                await refreshNotifications();
            } catch (err) {
                flash(err.message, 'error');
            }
        };
    });
}

// Кабинет мастера
async function loadMasterArea() {
    if (!state.me || state.me.role !== 'master') return;

    $('#master-section').classList.remove('hidden');

    // Профиль мастера
    try {
        const p = await api('/api/v1/master/profile');
        $('#profile-form [name="description"]').value = p.description || '';
        $('#profile-form [name="photo_url"]').value = p.photo_url || '';
    } catch {}

    // Записи к мастеру
    try {
        state.masterAppointmentsRaw = await api('/api/v1/master/appointments');
        renderMasterAppointments();
    } catch (err) {
        flash(err.message, 'error');
    }
}

function renderMasterAppointments() {
    const appointments = state.masterAppointmentsRaw.filter(a => matchesStatusFilter(a.status, state.masterAppointmentsFilter));

    $('#master-appointments').innerHTML = appointments.length
        ? appointments.map(a => `
            <div class="item">
                <div class="item-head">
                    <strong>${escapeHtml(a.client_name)} · ${escapeHtml(a.service_name)}</strong>
                    <span class="badge">${a.status}</span>
                </div>
                <div class="meta">${new Date(a.start_time).toLocaleString()} · ${escapeHtml(a.client_email)}</div>
                <div class="actions">
                    ${a.status === 'pending' ? `<button class="primary" data-confirm="${a.id}">Подтвердить</button>` : ''}
                    ${a.status === 'confirmed' ? `<button class="primary" data-complete="${a.id}">Завершить</button>` : ''}
                    ${['pending', 'confirmed'].includes(a.status) ? `<button class="danger" data-master-cancel="${a.id}">Отменить</button>` : ''}
                </div>
            </div>
        `).join('')
        : '<div class="muted">Записей нет.</div>';

    $$('[data-confirm]').forEach(b => b.onclick = () => masterStatus(b.dataset.confirm, 'confirm'));
    $$('[data-complete]').forEach(b => b.onclick = () => masterStatus(b.dataset.complete, 'complete'));
    $$('[data-master-cancel]').forEach(b => b.onclick = () => masterStatus(b.dataset.masterCancel, 'cancel'));
}

// Действие мастера над записью (подтвердить/завершить/отменить)
async function masterStatus(id, action) {
    const url = `/api/v1/master/appointments/${id}/${action}`;

    try {
        await api(url, { method: 'PATCH' });
        await loadMasterArea();
        await refreshNotifications();
        flash('Статус записи обновлён');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Сохранить профиль мастера
async function saveProfile(e) {
    e.preventDefault();

    try {
        const f = new FormData(e.target);
        await api('/api/v1/master/profile', {
            method: 'PUT',
            body: JSON.stringify({
                description: f.get('description'),
                photo_url: f.get('photo_url') || null,
            }),
        });

        flash('Профиль сохранён');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Загрузить аватар
async function uploadAvatar(e) {
    e.preventDefault();

    try {
        const formData = new FormData(e.target);
        const p = await uploadFile('/api/v1/master/avatar', formData);
        $('#profile-form [name="photo_url"]').value = p.photo_url || '';
        e.target.reset();
        flash('Аватар загружен');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Добавить услугу
async function addService(e) {
    e.preventDefault();

    try {
        const f = new FormData(e.target);
        await api('/api/v1/master/services', {
            method: 'POST',
            body: JSON.stringify({
                name: f.get('name'),
                description: f.get('description'),
                price: Number(f.get('price')),
                duration_minutes: Number(f.get('duration_minutes')),
            }),
        });

        e.target.reset();
        flash('Услуга добавлена');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Сохранить график
async function saveSchedule(e) {
    e.preventDefault();

    try {
        const f = new FormData(e.target);
        await api('/api/v1/master/schedule', {
            method: 'PUT',
            body: JSON.stringify({
                day_of_week: Number(f.get('day_of_week')),
                start_time: f.get('start_time'),
                end_time: f.get('end_time'),
            }),
        });

        flash('График сохранён');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Кабинет администратора
async function loadAdminArea() {
    if (!state.me || state.me.role !== 'admin') return;

    $('#admin-section').classList.remove('hidden');

    state.adminMasters = await api('/api/v1/masters');
    const select = $('#admin-master-select');
    const current = select.value;
    select.innerHTML = '<option value="">— выберите мастера —</option>' +
        state.adminMasters.map(m => `<option value="${m.id}">${escapeHtml(m.name)}</option>`).join('');
    select.value = current;
}

// Создать нового мастера
async function adminCreateMaster(e) {
    e.preventDefault();

    try {
        const f = new FormData(e.target);
        await api('/api/v1/admin/masters', {
            method: 'POST',
            body: JSON.stringify({
                name: f.get('name'),
                email: f.get('email'),
                password: f.get('password'),
                description: f.get('description'),
                photo_url: f.get('photo_url') || null,
            }),
        });

        e.target.reset();
        await loadAdminArea();
        flash('Мастер создан');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Выбор редактируемого мастера
async function onAdminMasterChange(e) {
    const id = e.target.value;
    state.selectedAdminMasterId = id || null;
    $('#admin-master-area').classList.toggle('hidden', !id);
    if (!id) return;

    const master = state.adminMasters.find(m => String(m.id) === id);
    $('#admin-profile-form [name="description"]').value = master?.description || '';
    $('#admin-profile-form [name="photo_url"]').value = master?.photo_url || '';

    resetAdminServiceForm();
    await Promise.all([loadAdminServices(), loadAdminSchedule()]);
}

// Сохранить профиль выбранного мастера
async function saveAdminProfile(e) {
    e.preventDefault();
    if (!state.selectedAdminMasterId) return;

    try {
        const f = new FormData(e.target);
        await api(`/api/v1/admin/masters/${state.selectedAdminMasterId}/profile`, {
            method: 'PUT',
            body: JSON.stringify({
                description: f.get('description'),
                photo_url: f.get('photo_url') || null,
            }),
        });

        await loadAdminArea();
        flash('Профиль мастера сохранён');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Услуги выбранного мастера
async function loadAdminServices() {
    const items = await api(`/api/v1/masters/${state.selectedAdminMasterId}/services`);

    $('#admin-services').innerHTML = items.length
        ? items.map(s => `
            <div class="item">
                <div class="item-head">
                    <strong>${escapeHtml(s.name)}</strong>
                    <span class="badge">${s.price} ₽ · ${s.duration_minutes} мин</span>
                </div>
                <div class="meta">${escapeHtml(s.description || '')}</div>
                <div class="actions">
                    <button class="secondary" data-edit-service="${s.id}">Изменить</button>
                    <button class="danger" data-delete-service="${s.id}">Удалить</button>
                </div>
            </div>
        `).join('')
        : '<div class="muted">Услуг пока нет.</div>';

    $$('[data-edit-service]').forEach(b => {
        b.onclick = () => {
            const item = items.find(s => s.id === Number(b.dataset.editService));
            if (item) editAdminService(item);
        };
    });
    $$('[data-delete-service]').forEach(b => {
        b.onclick = () => deleteAdminService(b.dataset.deleteService);
    });
}

function editAdminService(item) {
    state.editingServiceId = item.id;
    const f = $('#admin-service-form');
    f.name.value = item.name;
    f.description.value = item.description || '';
    f.price.value = item.price;
    f.duration_minutes.value = item.duration_minutes;
    $('#admin-service-form-title').textContent = 'Изменить услугу';
    $('#admin-service-cancel-edit').classList.remove('hidden');
}

function resetAdminServiceForm() {
    state.editingServiceId = null;
    $('#admin-service-form').reset();
    $('#admin-service-form-title').textContent = 'Новая услуга';
    $('#admin-service-cancel-edit').classList.add('hidden');
}

// Создать или изменить услугу выбранного мастера
async function submitAdminService(e) {
    e.preventDefault();
    if (!state.selectedAdminMasterId) return;

    try {
        const f = new FormData(e.target);
        const body = JSON.stringify({
            name: f.get('name'),
            description: f.get('description'),
            price: Number(f.get('price')),
            duration_minutes: Number(f.get('duration_minutes')),
        });
        const base = `/api/v1/admin/masters/${state.selectedAdminMasterId}/services`;

        if (state.editingServiceId) {
            await api(`${base}/${state.editingServiceId}`, { method: 'PUT', body });
        } else {
            await api(base, { method: 'POST', body });
        }

        resetAdminServiceForm();
        await loadAdminServices();
        flash('Услуга сохранена');
    } catch (err) {
        flash(err.message, 'error');
    }
}

async function deleteAdminService(id) {
    try {
        await api(`/api/v1/admin/masters/${state.selectedAdminMasterId}/services/${id}`, { method: 'DELETE' });
        await loadAdminServices();
        flash('Услуга удалена');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// График выбранного мастера
async function loadAdminSchedule() {
    const items = await api(`/api/v1/masters/${state.selectedAdminMasterId}/schedule`);
    const days = ['', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

    $('#admin-schedule').innerHTML = items.length
        ? items.map(w => `
            <div class="item">
                <div class="item-head">
                    <strong>${days[w.day_of_week]}</strong>
                    <span class="badge">${w.start_time} – ${w.end_time}</span>
                </div>
                <div class="actions">
                    <button class="danger" data-delete-day="${w.day_of_week}">Удалить</button>
                </div>
            </div>
        `).join('')
        : '<div class="muted">График не задан.</div>';

    $$('[data-delete-day]').forEach(b => {
        b.onclick = () => deleteAdminScheduleDay(b.dataset.deleteDay);
    });
}

async function saveAdminSchedule(e) {
    e.preventDefault();
    if (!state.selectedAdminMasterId) return;

    try {
        const f = new FormData(e.target);
        await api(`/api/v1/admin/masters/${state.selectedAdminMasterId}/schedule`, {
            method: 'PUT',
            body: JSON.stringify({
                day_of_week: Number(f.get('day_of_week')),
                start_time: f.get('start_time'),
                end_time: f.get('end_time'),
            }),
        });

        await loadAdminSchedule();
        flash('График сохранён');
    } catch (err) {
        flash(err.message, 'error');
    }
}

async function deleteAdminScheduleDay(day) {
    try {
        await api(`/api/v1/admin/masters/${state.selectedAdminMasterId}/schedule/${day}`, { method: 'DELETE' });
        await loadAdminSchedule();
        flash('День удалён из графика');
    } catch (err) {
        flash(err.message, 'error');
    }
}

// Обновить всё сразу
async function refreshAll() {
    await loadMasters();
    await refreshAppointments();
    await refreshNotifications();
    await loadMasterArea();
    await loadAdminArea();
}

// Экранирование HTML
function escapeHtml(v) {
    return String(v ?? '').replace(/[&<>'"]/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        "'": '&#39;',
        '"': '&quot;',
    }[c]));
}

// Переключение табов вход/регистрация
$$('.tab').forEach(t => {
    t.onclick = () => {
        $$('.tab').forEach(x => x.classList.remove('active'));
        t.classList.add('active');

        $('#login-form').classList.toggle('hidden', t.dataset.tab !== 'login');
        $('#register-form').classList.toggle('hidden', t.dataset.tab !== 'register');
    };
});

// Переключение вкладок-фильтров списка записей
$$('#appointments-tabs .tab').forEach(t => {
    t.onclick = () => {
        $$('#appointments-tabs .tab').forEach(x => x.classList.remove('active'));
        t.classList.add('active');
        state.appointmentsFilter = t.dataset.statusTab;
        renderAppointments();
    };
});
$$('#master-appointments-tabs .tab').forEach(t => {
    t.onclick = () => {
        $$('#master-appointments-tabs .tab').forEach(x => x.classList.remove('active'));
        t.classList.add('active');
        state.masterAppointmentsFilter = t.dataset.statusTab;
        renderMasterAppointments();
    };
});

// Обработчики событий
$('#login-form').addEventListener('submit', login);
$('#register-form').addEventListener('submit', register);
$('#refresh-client').onclick = loadMasters;
$('#refresh-appointments').onclick = refreshAppointments;
$('#refresh-notifications').onclick = refreshNotifications;
$('#refresh-master').onclick = loadMasterArea;
$('#service-select').addEventListener('change', loadSlots);
$('#date-input').addEventListener('change', loadSlots);
$('#close-booking').onclick = () => $('#booking-card').classList.add('hidden');
$('#profile-form').addEventListener('submit', saveProfile);
$('#avatar-form').addEventListener('submit', uploadAvatar);
$('#service-form').addEventListener('submit', addService);
$('#schedule-form').addEventListener('submit', saveSchedule);
$('#admin-create-master-form').addEventListener('submit', adminCreateMaster);
$('#admin-master-select').addEventListener('change', onAdminMasterChange);
$('#admin-profile-form').addEventListener('submit', saveAdminProfile);
$('#admin-service-form').addEventListener('submit', submitAdminService);
$('#admin-service-cancel-edit').onclick = resetAdminServiceForm;
$('#admin-schedule-form').addEventListener('submit', saveAdminSchedule);

// Старт
loadMe();