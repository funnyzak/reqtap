const CONFIG = window.__REQTAP__ || {};
const API_BASE = CONFIG.apiBase || '/api';
const WS_PATH = CONFIG.wsEndpoint || `${API_BASE}/ws`;
const AUTH_ENABLED = CONFIG.authEnabled !== false;
const MAX_REQUESTS = CONFIG.maxRequests || 500;
const EXPORT_ENABLED = CONFIG.exportEnabled !== false;
const WEB_BASE = CONFIG.webBase || '/web';
const ROLE_ADMIN = CONFIG.roleAdmin || 'admin';
const ROLE_VIEWER = CONFIG.roleViewer || 'viewer';

const state = {
  requests: [],
  filters: {
    search: '',
    method: '',
  },
  userRole: '',
};

let ws;
let reconnectTimer;

const els = {
  body: document.getElementById('requests-body'),
  empty: document.getElementById('empty-state'),
  search: document.getElementById('search-input'),
  method: document.getElementById('method-filter'),
  refresh: document.getElementById('refresh-btn'),
  logout: document.getElementById('logout-btn'),
  total: document.getElementById('total-counter'),
  filtered: document.getElementById('filtered-counter'),
  exportBtns: document.querySelectorAll('.export-btn'),
  exportSection: document.getElementById('export-section'),
  wsStatus: document.getElementById('ws-status'),
  user: document.getElementById('current-user'),
  role: document.getElementById('current-role'),
  modal: document.getElementById('detail-modal'),
  modalClose: document.getElementById('detail-close'),
  detailMeta: document.getElementById('detail-meta'),
  detailHeaders: document.getElementById('detail-headers'),
  detailBody: document.getElementById('detail-body'),
};

async function apiFetch(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  const headers = new Headers(options.headers || {});
  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const resp = await fetch(url, {
    credentials: 'include',
    ...options,
    headers,
  });

  if (resp.status === 401 && AUTH_ENABLED) {
    window.location.href = `${WEB_BASE}/login`;
    throw new Error('Unauthorized');
  }

  if (!resp.ok) {
    const message = await resp.text();
    throw new Error(message || 'Request failed');
  }

  return resp;
}

async function loadUser() {
  try {
    const resp = await apiFetch('/auth/me');
    const data = await resp.json();
    const username = data.username || 'guest';
    const role = data.role || '';
    const authEnabled = data.auth !== false;
    els.user.textContent = username;
    els.role.textContent = role;
    state.userRole = role;
    updateUIForRole(role, authEnabled);
  } catch (error) {
    console.error('Failed to load user info', error);
    updateUIForRole('', false);
  }
}

function updateUIForRole(role, authEnabled) {
  // 如果认证未启用，允许所有操作
  const canExport = !authEnabled || role === ROLE_ADMIN;
  const exportButtons = els.exportBtns;
  
  if (els.exportSection) {
    if (!canExport) {
      els.exportSection.style.opacity = '0.5';
      els.exportSection.style.pointerEvents = 'none';
      els.exportSection.title = 'You are not authorized to export data';
    } else {
      els.exportSection.style.opacity = '1';
      els.exportSection.style.pointerEvents = 'auto';
      els.exportSection.title = '';
    }
  }
  
  exportButtons.forEach(btn => {
    if (!canExport) {
      btn.disabled = true;
      btn.style.cursor = 'not-allowed';
      btn.style.opacity = '0.6';
    } else {
      btn.disabled = false;
      btn.style.cursor = 'pointer';
      btn.style.opacity = '1';
    }
  });
}

async function loadRequests() {
  try {
    const params = new URLSearchParams({
      limit: '200',
    });
    const resp = await apiFetch(`/requests?${params.toString()}`);
    const payload = await resp.json();
    state.requests = payload.data || [];
    render();
  } catch (error) {
    console.error('Failed to load requests', error);
  }
}

function render() {
  const filtered = applyFilters();
  els.total.textContent = state.requests.length;
  els.filtered.textContent = filtered.length;
  els.empty.classList.toggle('hidden', filtered.length > 0);

  els.body.innerHTML = '';
  const template = document.getElementById('row-template');

  filtered.forEach((item) => {
    const clone = template.content.firstElementChild.cloneNode(true);
    const cells = clone.querySelectorAll('td');
    cells[0].textContent = formatTime(item.timestamp);
    cells[1].innerHTML = `<span class="method-badge">${item.method}</span>`;
    cells[2].textContent = `${item.path}${item.query ? `?${item.query}` : ''}`;
    cells[3].textContent = item.remote_addr;
    cells[4].textContent = item.user_agent || '-';
    cells[5].textContent = formatSize(item.size || item.content_length || 0);
    clone.addEventListener('click', () => openDetail(item));
    els.body.appendChild(clone);
  });
}

function applyFilters() {
  const search = state.filters.search.toLowerCase();
  const method = state.filters.method.toUpperCase();
  return state.requests.filter((req) => {
    if (method && req.method !== method) {
      return false;
    }
    if (search) {
      const target = [
        req.path,
        req.query,
        req.remote_addr,
        req.user_agent,
      ].join(' ').toLowerCase();
      return target.includes(search);
    }
    return true;
  });
}

function pushRequest(data) {
  state.requests.unshift(data);
  if (state.requests.length > MAX_REQUESTS) {
    state.requests.length = MAX_REQUESTS;
  }
  render();
}

function formatTime(value) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

function formatSize(bytes) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = bytes;
  let idx = 0;
  while (size >= 1024 && idx < units.length - 1) {
    size /= 1024;
    idx += 1;
  }
  return `${size.toFixed(1)} ${units[idx]}`;
}

function openDetail(item) {
  const fullPath = `${item.path}${item.query ? `?${item.query}` : ''}`;
  const bodySize = formatSize(item.size || item.content_length || 0);
  els.detailMeta.innerHTML = `
    <div class="grid gap-2 text-sm">
      <div><span class="detail-label">ID:</span>${item.id || '-'}</div>
      <div><span class="detail-label">Timestamp:</span>${formatTime(item.timestamp)}</div>
      <div><span class="detail-label">Method:</span>${item.method}</div>
      <div><span class="detail-label">Path:</span>${fullPath}</div>
      <div><span class="detail-label">Client:</span>${item.remote_addr || '-'}</div>
      <div><span class="detail-label">User-Agent:</span>${item.user_agent || '-'}</div>
      <div><span class="detail-label">Content-Type:</span>${item.content_type || '-'}</div>
      <div><span class="detail-label">Body Size:</span>${bodySize}</div>
    </div>
  `;

  els.detailHeaders.textContent = formatHeaders(item.headers || {});
  els.detailBody.textContent = decodeBody(item);
  els.modal.classList.remove('hidden');
  els.modal.classList.add('flex');
}

function closeDetail() {
  els.modal.classList.add('hidden');
  els.modal.classList.remove('flex');
}

function formatHeaders(headers) {
  return Object.entries(headers)
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(', ') : value}`)
    .join('\n');
}

function decodeBody(item) {
  if (!item.body || item.body.length === 0) {
    return '(empty body)';
  }

  if (item.is_binary) {
    return `[Binary payload] ${formatSize(item.size || item.content_length || item.body.length)} `;
  }

  try {
    const binary = window.atob(item.body);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return '(Unable to decode body)';
  }
}

function updateWsStatus(status) {
  if (!els.wsStatus) return;
  const indicator = els.wsStatus.querySelector('.indicator');
  const label = els.wsStatus.querySelector('.status-label');
  if (!indicator || !label) return;

  switch (status) {
    case 'connected':
      indicator.className = 'indicator h-2 w-2 rounded-full bg-emerald-400';
      label.textContent = 'Online';
      break;
    case 'connecting':
      indicator.className = 'indicator h-2 w-2 rounded-full bg-yellow-400';
      label.textContent = 'Connecting';
      break;
    case 'error':
      indicator.className = 'indicator h-2 w-2 rounded-full bg-rose-400';
      label.textContent = 'Error';
      break;
    default:
      indicator.className = 'indicator h-2 w-2 rounded-full bg-red-400';
      label.textContent = 'Offline';
  }
}

function initWebsocket() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
  }

  updateWsStatus('connecting');
  try {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${protocol}://${window.location.host}${WS_PATH}`;
    ws = new WebSocket(url);
  } catch (error) {
    console.error('Failed to initiate websocket', error);
    scheduleReconnect();
    return;
  }

  ws.onopen = () => updateWsStatus('connected');
  ws.onerror = () => updateWsStatus('error');
  ws.onclose = () => {
    updateWsStatus('disconnected');
    scheduleReconnect();
  };
  ws.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data);
      if (payload.type === 'request' && payload.data) {
        pushRequest(payload.data);
      }
    } catch (error) {
      console.error('Failed to parse websocket payload', error);
    }
  };
}

function scheduleReconnect() {
  updateWsStatus('connecting');
  reconnectTimer = setTimeout(() => {
    initWebsocket();
  }, 3000);
}

async function handleExport(format) {
  if (!EXPORT_ENABLED) {
    alert('Export feature is disabled');
    return;
  }
  
  if (AUTH_ENABLED && state.userRole !== ROLE_ADMIN) {
    alert('You need admin role to export data');
    return;
  }
  
  const params = new URLSearchParams({
    format,
    search: state.filters.search || '',
    method: state.filters.method || '',
  });

  try {
    const resp = await apiFetch(`/export?${params.toString()}`, {
      method: 'GET',
      headers: {},
    });
    
    if (resp.status === 403) {
      const text = await resp.text();
      alert(text || 'You are not authorized to export data');
      return;
    }
    
    const blob = await resp.blob();
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = resp.headers.get('Content-Disposition')?.split('filename=')[1]?.replace(/"/g, '') || `reqtap.${format}`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  } catch (error) {
    if (error.message && error.message.includes('403')) {
      alert('You are not authorized to export data');
    } else {
      console.error('Export failed', error);
      alert('Export failed: ' + (error.message || 'Unknown error'));
    }
  }
}

async function handleLogout() {
  try {
    await apiFetch('/auth/logout', { method: 'POST' });
  } catch (error) {
    console.error('Logout failed', error);
  } finally {
    window.location.href = `${WEB_BASE}/login`;
  }
}

function bindEvents() {
  els.search.addEventListener('input', (event) => {
    state.filters.search = event.target.value;
    render();
  });

  els.method.addEventListener('change', (event) => {
    state.filters.method = event.target.value;
    render();
  });

  els.refresh.addEventListener('click', () => loadRequests());
  els.logout.addEventListener('click', handleLogout);
  els.modalClose.addEventListener('click', closeDetail);
  els.modal.addEventListener('click', (event) => {
    if (event.target === els.modal) {
      closeDetail();
    }
  });

  els.exportBtns.forEach((btn) =>
    btn.addEventListener('click', () => handleExport(btn.dataset.format))
  );

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      closeDetail();
    }
  });
}

async function bootstrap() {
  await loadUser();
  await loadRequests();
  bindEvents();
  initWebsocket();
}

bootstrap();
