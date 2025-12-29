import { createI18n } from './i18n.js';

const CONFIG = window.__REQTAP__ || {};
const API_BASE = CONFIG.apiBase || '/api';
const WS_PATH = CONFIG.wsEndpoint || `${API_BASE}/ws`;
const AUTH_ENABLED = CONFIG.authEnabled !== false;
const MAX_REQUESTS = CONFIG.maxRequests || 500;
const EXPORT_ENABLED = CONFIG.exportEnabled !== false;
const WEB_BASE = CONFIG.webBase || '/web';
const ROLE_ADMIN = CONFIG.roleAdmin || 'admin';
const ROLE_VIEWER = CONFIG.roleViewer || 'viewer';
const THEME_STORAGE_KEY = 'reqtap-theme';
const DEFAULT_THEME = 'dark';
const i18n = createI18n({
  defaultLocale: CONFIG.defaultLocale || 'en',
  supportedLocales: CONFIG.supportedLocales || ['en'],
  webBase: WEB_BASE,
});

const state = {
  requests: [],
  filters: {
    search: '',
    method: '',
  },
  userRole: '',
  locale: CONFIG.defaultLocale || 'en',
  activeRequest: null,
  activeRequestBody: '',
  theme: DEFAULT_THEME,
  detailBodyRaw: '',
  detailBodyPretty: '',
  detailBodyMode: 'raw',
  wsStatus: 'connecting',
};

let ws;
let reconnectTimer;
let actionStatusTimer;

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
  requestDownload: document.getElementById('request-download-btn'),
  requestCopy: document.getElementById('request-copy-btn'),
  curlCopy: document.getElementById('curl-copy-btn'),
  replayBtn: document.getElementById('replay-btn'),
  replayModal: document.getElementById('replay-modal'),
  replayClose: document.getElementById('replay-close'),
  replayCancel: document.getElementById('replay-cancel'),
  replaySubmit: document.getElementById('replay-submit'),
  replayTargetUrl: document.getElementById('replay-target-url'),
  replayMethod: document.getElementById('replay-method'),
  replayHeaders: document.getElementById('replay-headers'),
  replayBody: document.getElementById('replay-body'),
  replayQuery: document.getElementById('replay-query'),
  replayStatus: document.getElementById('replay-status'),
  actionStatus: document.getElementById('detail-action-status'),
  adminAreas: document.querySelectorAll('[data-admin-only="true"]'),
  adminButtons: document.querySelectorAll('[data-admin-action="true"]'),
  themeToggle: document.getElementById('theme-toggle'),
  themeToggleLabel: document.getElementById('theme-toggle-label'),
  themeToggleIcon: document.getElementById('theme-toggle-icon'),
  localeSelect: document.getElementById('locale-select'),
  headersCopyBtn: document.getElementById('headers-copy-btn'),
  headersWrapBtn: document.getElementById('headers-wrap-btn'),
  bodyCopyBtn: document.getElementById('body-copy-btn'),
  bodyWrapBtn: document.getElementById('body-wrap-btn'),
  bodyFormatToggle: document.getElementById('body-format-toggle'),
};

function getStoredTheme() {
  try {
    return localStorage.getItem(THEME_STORAGE_KEY);
  } catch (error) {
    console.warn('Unable to read theme preference', error);
    return null;
  }
}

function persistTheme(theme) {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch (error) {
    console.warn('Unable to persist theme preference', error);
  }
}

function updateThemeToggleUI(theme) {
  if (!els.themeToggle) return;
  const isLight = theme === 'light';
  const nextThemeLabel = isLight ? i18n.t('header.theme.dark') : i18n.t('header.theme.light');
  els.themeToggle.setAttribute('aria-pressed', String(isLight));
  els.themeToggle.setAttribute('title', i18n.t('header.theme.switch_to', { mode: nextThemeLabel }));
  if (els.themeToggleLabel) {
    els.themeToggleLabel.textContent = nextThemeLabel;
  }
  if (els.themeToggleIcon) {
    const icon = isLight ? 'fa-moon' : 'fa-sun';
    els.themeToggleIcon.className = `fa-solid ${icon}`;
  }
}

function applyTheme(theme) {
  const resolvedTheme = theme === 'light' ? 'light' : 'dark';
  state.theme = resolvedTheme;
  document.documentElement.setAttribute('data-theme', resolvedTheme);
  updateThemeToggleUI(resolvedTheme);
}

function initTheme() {
  const stored = getStoredTheme();
  applyTheme(stored || DEFAULT_THEME);
  if (els.themeToggle) {
    els.themeToggle.addEventListener('click', () => {
      const nextTheme = state.theme === 'light' ? 'dark' : 'light';
      applyTheme(nextTheme);
      persistTheme(nextTheme);
    });
  }
}

function escapeHtml(value) {
  if (value === null || value === undefined) {
    return '';
  }
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function buildDetailMeta(item, fullPath, bodySize) {
  const entries = [
    { label: i18n.t('detail.meta.request_id'), value: item.id || '-' },
    { label: i18n.t('detail.meta.timestamp'), value: formatTime(item.timestamp) },
    { label: i18n.t('detail.meta.method'), value: (item.method || '-').toUpperCase(), pill: 'method' },
    { label: i18n.t('detail.meta.body_size'), value: bodySize, pill: 'metric' },
    { label: i18n.t('detail.meta.content_type'), value: item.content_type || '-' },
    { label: i18n.t('detail.meta.client'), value: item.remote_addr || '-', mono: true },
    { label: i18n.t('detail.meta.full_path'), value: fullPath, full: true, code: true },
    { label: i18n.t('detail.meta.user_agent'), value: item.user_agent || '-', full: true, mono: true },
  ];

  const markup = entries
    .map((entry) => {
      const classes = ['detail-meta__item'];
      if (entry.full) {
        classes.push('detail-meta__item--full');
      }
      const label = escapeHtml(entry.label);
      const safeValue = escapeHtml(entry.value || '-');
      let valueMarkup = safeValue;
      if (entry.code) {
        valueMarkup = `<code class="detail-meta__code">${safeValue}</code>`;
      } else if (entry.mono) {
        valueMarkup = `<span class="detail-meta__mono">${safeValue}</span>`;
      }
      if (entry.pill) {
        valueMarkup = `<span class="detail-pill detail-pill--${entry.pill}">${valueMarkup}</span>`;
      }
      return `
        <div class="${classes.join(' ')}">
          <p class="detail-meta__label">${label}</p>
          <p class="detail-meta__value">${valueMarkup}</p>
        </div>`;
    })
    .join('');

  return `<div class="detail-meta__grid">${markup}</div>`;
}

function tryFormatJson(text) {
  if (!text) {
    return null;
  }
  const trimmed = text.trim();
  if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) {
    return null;
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch (error) {
    return null;
  }
}

function isBodyPlaceholder(text) {
  if (!text) {
    return true;
  }
  const emptyBody = i18n.t('detail.placeholders.empty_body');
  const binaryLabel = i18n.t('detail.placeholders.binary_body');
  return text === emptyBody || text.startsWith(binaryLabel);
}

function setWrapState(block, button, shouldWrap) {
  if (!block) {
    return;
  }
  block.classList.toggle('code-block--wrap', shouldWrap);
  updateWrapButton(button, shouldWrap);
}

function updateWrapButton(button, shouldWrap) {
  if (!button) {
    return;
  }
  button.setAttribute('aria-pressed', String(shouldWrap));
  const label = button.querySelector('.detail-tool-btn__label');
  if (label) {
    label.textContent = shouldWrap
      ? i18n.t('detail.tools.wrap')
      : i18n.t('detail.tools.scroll');
  }
}

function toggleWrapState(block, button) {
  if (!block) {
    return;
  }
  const nextWrap = !block.classList.contains('code-block--wrap');
  setWrapState(block, button, nextWrap);
}

function updateBodyFormatToggle() {
  if (!els.bodyFormatToggle) {
    return;
  }
  const label = els.bodyFormatToggle.querySelector('.detail-tool-btn__label');
  const hasPretty = Boolean(state.detailBodyPretty);
  els.bodyFormatToggle.disabled = !hasPretty;
  if (!hasPretty) {
    els.bodyFormatToggle.setAttribute('aria-pressed', 'false');
    if (label) {
      label.textContent = i18n.t('detail.tools.pretty');
    }
    return;
  }

  const isPretty = state.detailBodyMode === 'pretty';
  els.bodyFormatToggle.setAttribute('aria-pressed', String(isPretty));
  if (label) {
    label.textContent = isPretty ? i18n.t('detail.tools.raw') : i18n.t('detail.tools.pretty');
  }
}

function renderDetailBody() {
  if (!els.detailBody) {
    return;
  }
  const showPretty = state.detailBodyMode === 'pretty' && state.detailBodyPretty;
  const content = showPretty ? state.detailBodyPretty : state.detailBodyRaw;
  els.detailBody.textContent = content || '';
  updateBodyFormatToggle();
}

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
    throw new Error(message || i18n.t('alerts.request_failed'));
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
  const forbiddenText = i18n.t('alerts.export_forbidden');
  const adminText = i18n.t('alerts.admin_required');

  if (els.exportSection) {
    if (!canExport) {
      els.exportSection.style.opacity = '0.5';
      els.exportSection.style.pointerEvents = 'none';
      els.exportSection.title = forbiddenText;
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

  const adminAreas = els.adminAreas || [];
  adminAreas.forEach((area) => {
    if (!area) return;
    area.classList.toggle('admin-locked', !canExport);
    if (!canExport) {
      area.setAttribute('title', adminText);
    } else {
      area.removeAttribute('title');
    }
  });

  const adminButtons = els.adminButtons || [];
  adminButtons.forEach((btn) => {
    if (!btn) return;
    btn.disabled = !canExport;
    btn.setAttribute('aria-disabled', String(!canExport));
    const labelKey = btn.getAttribute('data-i18n-title');
    const label = labelKey ? i18n.t(labelKey) : btn.getAttribute('aria-label') || btn.dataset.label || '';
    if (!canExport) {
      btn.title = adminText;
    } else if (label) {
      btn.title = label;
    } else {
      btn.removeAttribute('title');
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
  const fullPath = composeRequestPath(item);
  const bodySize = formatSize(item.size || item.content_length || 0);
  els.detailMeta.innerHTML = buildDetailMeta(item, fullPath, bodySize);

  const headersText = formatHeaders(item.headers || {});
  if (els.detailHeaders) {
    els.detailHeaders.textContent = headersText || i18n.t('detail.placeholders.no_headers');
    setWrapState(els.detailHeaders, els.headersWrapBtn, true);
  }
  const decodedBody = decodeBody(item);
  state.activeRequest = item;
  state.activeRequestBody = decodedBody;
  state.detailBodyRaw = decodedBody;
  state.detailBodyPretty = isBodyPlaceholder(decodedBody) ? null : tryFormatJson(decodedBody);
  state.detailBodyMode = state.detailBodyPretty ? 'pretty' : 'raw';
  renderDetailBody();
  setWrapState(els.detailBody, els.bodyWrapBtn, true);
  clearActionStatus();
  els.modal.classList.remove('hidden');
  els.modal.classList.add('flex');
}

function closeDetail() {
  els.modal.classList.add('hidden');
  els.modal.classList.remove('flex');
  state.activeRequest = null;
  state.activeRequestBody = '';
  clearActionStatus();
}

function formatHeaders(headers) {
  return Object.entries(headers)
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(', ') : value}`)
    .join('\n');
}

function decodeBody(item) {
  if (!item.body || item.body.length === 0) {
    return i18n.t('detail.placeholders.empty_body');
  }

  if (item.is_binary) {
    const binaryLabel = i18n.t('detail.placeholders.binary_body');
    return `${binaryLabel} ${formatSize(item.size || item.content_length || item.body.length)} `;
  }

  try {
    const binary = window.atob(item.body);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return i18n.t('detail.placeholders.undecodable');
  }
}

function canUseAdminActions() {
  return !AUTH_ENABLED || state.userRole === ROLE_ADMIN;
}

function ensureAdminAction() {
  if (!canUseAdminActions()) {
    const message = i18n.t('detail.status.admin_required');
    setActionStatus(message, 'error');
    alert(i18n.t('alerts.admin_required'));
    return false;
  }
  return true;
}

function ensureActiveRequest() {
  if (!state.activeRequest) {
    setActionStatus(i18n.t('detail.status.select_request'), 'error');
    return null;
  }
  return state.activeRequest;
}

function updateWsStatus(status) {
  state.wsStatus = status;
  if (!els.wsStatus) return;
  const indicator = els.wsStatus.querySelector('.indicator');
  const label = els.wsStatus.querySelector('.status-label');
  if (!indicator || !label) return;

  switch (status) {
    case 'connected':
      indicator.className = 'indicator h-2 w-2 rounded-full bg-emerald-400';
      label.textContent = i18n.t('header.ws.connected');
      break;
    case 'connecting':
      indicator.className = 'indicator h-2 w-2 rounded-full bg-yellow-400';
      label.textContent = i18n.t('header.ws.connecting');
      break;
    case 'error':
      indicator.className = 'indicator h-2 w-2 rounded-full bg-rose-400';
      label.textContent = i18n.t('header.ws.error');
      break;
    default:
      indicator.className = 'indicator h-2 w-2 rounded-full bg-red-400';
      label.textContent = i18n.t('header.ws.offline');
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
    alert(i18n.t('alerts.export_disabled'));
    return;
  }
  
  if (AUTH_ENABLED && state.userRole !== ROLE_ADMIN) {
    alert(i18n.t('alerts.export_admin_required'));
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
      alert(text || i18n.t('alerts.export_forbidden'));
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
      alert(i18n.t('alerts.export_forbidden'));
    } else {
      console.error('Export failed', error);
      const fallback = error.message || i18n.t('alerts.unknown_error');
      alert(i18n.t('alerts.export_failed', { error: fallback }));
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

  if (els.requestDownload) {
    els.requestDownload.addEventListener('click', handleRequestDownload);
  }
  if (els.requestCopy) {
    els.requestCopy.addEventListener('click', () => {
      handleRequestCopy();
    });
  }
    if (els.curlCopy) {
    els.curlCopy.addEventListener('click', () => {
      handleCurlCopy();
    });
  }

  if (els.replayBtn) {
    els.replayBtn.addEventListener('click', () => {
      openReplayModal();
    });
  }

  if (els.replayClose) {
    els.replayClose.addEventListener('click', () => {
      closeReplayModal();
    });
  }

  if (els.replayCancel) {
    els.replayCancel.addEventListener('click', () => {
      closeReplayModal();
    });
  }

  if (els.replaySubmit) {
    els.replaySubmit.addEventListener('click', () => {
      handleReplaySubmit();
    });
  }

  if (els.headersCopyBtn) {
    els.headersCopyBtn.addEventListener('click', () => handleHeadersCopy());
  }
  if (els.bodyCopyBtn) {
    els.bodyCopyBtn.addEventListener('click', () => handleBodyCopy());
  }
  if (els.bodyFormatToggle) {
    els.bodyFormatToggle.addEventListener('click', () => handleBodyFormatToggle());
  }
  if (els.headersWrapBtn && els.detailHeaders) {
    els.headersWrapBtn.addEventListener('click', () => toggleWrapState(els.detailHeaders, els.headersWrapBtn));
  }
  if (els.bodyWrapBtn && els.detailBody) {
    els.bodyWrapBtn.addEventListener('click', () => toggleWrapState(els.detailBody, els.bodyWrapBtn));
  }
}

function composeRequestPath(item) {
  if (!item) {
    return '/';
  }
  const basePath = item.path || '/';
  if (item.query) {
    return `${basePath}?${item.query}`;
  }
  return basePath;
}

function formatHeadersText(headers = {}) {
  const keys = Object.keys(headers);
  keys.sort((a, b) => a.localeCompare(b));
  const lines = [];
  keys.forEach((key) => {
    const value = headers[key];
    if (Array.isArray(value)) {
      value.forEach((val) => {
        if (val !== undefined && val !== null) {
          lines.push(`${key}: ${val}`);
        }
      });
    } else if (value !== undefined && value !== null) {
      lines.push(`${key}: ${value}`);
    }
  });
  return lines.join('\n');
}

function buildRequestPayload(item, decodedBody) {
  if (!item) {
    return '';
  }
  const requestLine = `${(item.method || 'GET').toUpperCase()} ${composeRequestPath(item)} HTTP/1.1`;
  const headerSection = formatHeadersText(item.headers || {});
  const emptyBodyLabel = i18n.t('detail.placeholders.empty_body');
  const bodySection = item.body && item.body.length > 0 ? (decodedBody || '') : emptyBodyLabel;
  const parts = [requestLine];
  if (headerSection) {
    parts.push(headerSection);
  }
  parts.push('', bodySection || emptyBodyLabel);
  return parts.join('\n');
}


function flattenHeaders(headers = {}) {
  const entries = [];
  Object.keys(headers)
    .sort((a, b) => a.localeCompare(b))
    .forEach((key) => {
      const value = headers[key];
      if (Array.isArray(value)) {
        value.forEach((val) => {
          if (val !== undefined && val !== null && val !== '') {
            entries.push([key, val]);
          }
        });
      } else if (value !== undefined && value !== null && value !== '') {
        entries.push([key, value]);
      }
    });
  return entries;
}

function buildCurlCommand(item, decodedBody) {
  const origin = window.location.origin || `${window.location.protocol}//${window.location.host}`;
  const url = `${origin}${composeRequestPath(item)}`;

  const headers = flattenHeaders(item.headers || {});
  const hasHeaders = headers.length > 0;
  const undecodableLabel = i18n.t('detail.placeholders.undecodable');
  const hasBody = item.body && item.body.length > 0 && !item.is_binary && decodedBody && decodedBody !== undecodableLabel;

  // If it's a simple command (no headers and no body), return single line
  if (!hasHeaders && !hasBody) {
    return `curl -X ${(item.method || 'GET').toUpperCase()} '${escapeShellSingleQuotes(url)}'`;
  }

  // Build multiline command
  const parts = [];

  // First line with backslash
  parts.push(`curl -X ${(item.method || 'GET').toUpperCase()} '${escapeShellSingleQuotes(url)}' \\`);

  // Add headers
  if (hasHeaders) {
    headers.forEach(([key, value], index) => {
      const sanitizedValue = String(value).replace(/\r?\n/g, ' ');
      // Add backslash unless this is the last header and there's no body
      const isLastHeader = index === headers.length - 1;
      const addBackslash = hasBody || !isLastHeader;
      parts.push(`  -H '${escapeShellSingleQuotes(`${key}: ${sanitizedValue}`)}'${addBackslash ? ' \\' : ''}`);
    });
  }

  // Add body
  if (hasBody) {
    const bodyLines = decodedBody.split('\n');
    if (bodyLines.length === 1) {
      parts.push(`  --data '${escapeShellSingleQuotes(decodedBody)}'`);
    } else {
      // For multiline body, join with \n and keep in single quotes
      const sanitizedBody = decodedBody.replace(/'/g, `'"'"'`);
      parts.push(`  --data '${sanitizedBody}'`);
    }
  }

  return parts.join('\n');
}

function escapeShellSingleQuotes(value) {
  if (!value) {
    return '';
  }
  return value.replace(/'/g, `'"'"'`);
}

function downloadText(filename, content) {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename || 'reqtap.txt';
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

async function copyToClipboard(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }

  return new Promise((resolve, reject) => {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.left = '-9999px';
    document.body.appendChild(textarea);
    textarea.select();
    try {
      const success = document.execCommand('copy');
      if (!success) {
        reject(new Error('Copy command failed'));
      } else {
        resolve();
      }
    } catch (error) {
      reject(error);
    } finally {
      document.body.removeChild(textarea);
    }
  });
}

function setActionStatus(message, type = 'info') {
  if (!els.actionStatus) {
    return;
  }
  clearTimeout(actionStatusTimer);
  els.actionStatus.textContent = message || '';
  if (type === 'error') {
    els.actionStatus.classList.add('error');
  } else {
    els.actionStatus.classList.remove('error');
  }
  if (message) {
    els.actionStatus.classList.remove('hidden');
    actionStatusTimer = setTimeout(() => {
      els.actionStatus.textContent = '';
      els.actionStatus.classList.remove('error');
      els.actionStatus.classList.add('hidden');
    }, 4000);
  } else {
    els.actionStatus.classList.add('hidden');
  }
}

function clearActionStatus() {
  clearTimeout(actionStatusTimer);
  if (els.actionStatus) {
    els.actionStatus.textContent = '';
    els.actionStatus.classList.remove('error');
    els.actionStatus.classList.add('hidden');
  }
}

function handleRequestDownload() {
  if (!ensureAdminAction()) return;
  const item = ensureActiveRequest();
  if (!item) return;
  const payload = buildRequestPayload(item, state.activeRequestBody);
  downloadText(`reqtap-request-${item.id || 'payload'}.txt`, payload);
  setActionStatus(i18n.t('detail.actions.status.request_downloaded'));
}

async function handleRequestCopy() {
  if (!ensureAdminAction()) return;
  const item = ensureActiveRequest();
  if (!item) return;
  const payload = buildRequestPayload(item, state.activeRequestBody);
  try {
    await copyToClipboard(payload);
    setActionStatus(i18n.t('detail.actions.status.request_copied'));
  } catch (error) {
    console.error('Failed to copy request payload', error);
    setActionStatus(i18n.t('detail.actions.status.request_copy_failed'), 'error');
  }
}


async function handleCurlCopy() {
  if (!ensureAdminAction()) return;
  const item = ensureActiveRequest();
  if (!item) return;
  try {
    await copyToClipboard(buildCurlCommand(item, state.activeRequestBody));
    setActionStatus(i18n.t('detail.actions.status.curl_copied'));
  } catch (error) {
    console.error('Failed to copy curl command', error);
    setActionStatus(i18n.t('detail.actions.status.curl_copy_failed'), 'error');
  }
}

async function handleHeadersCopy() {
  if (!els.detailHeaders) return;
  try {
    await copyToClipboard(els.detailHeaders.textContent || '');
    setActionStatus(i18n.t('detail.actions.status.headers_copied'));
  } catch (error) {
    console.error('Failed to copy headers', error);
    setActionStatus(i18n.t('detail.actions.status.headers_copy_failed'), 'error');
  }
}

async function handleBodyCopy() {
  if (!els.detailBody) return;
  try {
    await copyToClipboard(els.detailBody.textContent || '');
    setActionStatus(i18n.t('detail.actions.status.body_copied'));
  } catch (error) {
    console.error('Failed to copy body', error);
    setActionStatus(i18n.t('detail.actions.status.body_copy_failed'), 'error');
  }
}

function handleBodyFormatToggle() {
  if (!state.detailBodyPretty) {
    return;
  }
  state.detailBodyMode = state.detailBodyMode === 'pretty' ? 'raw' : 'pretty';
  renderDetailBody();
}

function initLocaleSelector() {
  if (!els.localeSelect) {
    return;
  }
  const locales = i18n.getSupportedLocales();
  els.localeSelect.innerHTML = '';
  locales.forEach((loc) => {
    const option = document.createElement('option');
    option.value = loc;
    option.textContent = i18n.t(`header.locale_label.${loc}`) || loc;
    els.localeSelect.appendChild(option);
  });
  els.localeSelect.value = i18n.getLocale();
  els.localeSelect.addEventListener('change', async (event) => {
    const next = event.target.value;
    await i18n.setLocale(next);
  });
}

function refreshLocaleUI() {
  document.title = i18n.t('meta.app_title') || document.title;
  updateThemeToggleUI(state.theme);
  updateBodyFormatToggle();
  if (els.detailHeaders) {
    const shouldWrapHeaders = els.detailHeaders.classList.contains('code-block--wrap');
    setWrapState(els.detailHeaders, els.headersWrapBtn, shouldWrapHeaders);
  }
  if (els.detailBody) {
    const shouldWrapBody = els.detailBody.classList.contains('code-block--wrap');
    setWrapState(els.detailBody, els.bodyWrapBtn, shouldWrapBody);
  }
  updateWsStatus(state.wsStatus || 'connecting');
  if (els.localeSelect) {
    Array.from(els.localeSelect.options).forEach((option) => {
      option.textContent = i18n.t(`header.locale_label.${option.value}`) || option.value;
    });
  }
}

// Replay functions
function openReplayModal() {
  if (!ensureAdminAction()) return;
  const item = ensureActiveRequest();
  if (!item) return;

  // Pre-fill the form with current request data
  if (els.replayTargetUrl) els.replayTargetUrl.value = '';
  if (els.replayMethod) els.replayMethod.value = item.method || 'POST';

  // Format headers as JSON
  if (els.replayHeaders) {
    const headers = {};
    const rawHeaders = item.headers || {};
    Object.entries(rawHeaders).forEach(([key, value]) => {
      headers[key] = Array.isArray(value) ? value[0] : value;
    });
    els.replayHeaders.value = JSON.stringify(headers, null, 2);
  }

  // Set body
  if (els.replayBody) {
    els.replayBody.value = state.activeRequestBody || '';
  }

  // Set query
  if (els.replayQuery) {
    els.replayQuery.value = item.query || '';
  }

  // Clear status
  if (els.replayStatus) {
    els.replayStatus.classList.add('hidden');
    els.replayStatus.textContent = '';
  }

  // Show modal
  if (els.replayModal) {
    els.replayModal.classList.remove('hidden');
    els.replayModal.classList.add('flex');
  }

  // Focus on target URL input
  setTimeout(() => {
    if (els.replayTargetUrl) els.replayTargetUrl.focus();
  }, 100);
}

function closeReplayModal() {
  if (els.replayModal) {
    els.replayModal.classList.add('hidden');
    els.replayModal.classList.remove('flex');
  }
}

function setReplayStatus(message, type = 'success') {
  if (!els.replayStatus) return;
  els.replayStatus.textContent = message;
  els.replayStatus.classList.remove('hidden', 'bg-green-100', 'text-green-800', 'bg-red-100', 'text-red-800', 'bg-blue-100', 'text-blue-800');

  if (type === 'success') {
    els.replayStatus.classList.add('bg-green-100', 'text-green-800');
  } else if (type === 'error') {
    els.replayStatus.classList.add('bg-red-100', 'text-red-800');
  } else {
    els.replayStatus.classList.add('bg-blue-100', 'text-blue-800');
  }
}

async function handleReplaySubmit() {
  const item = ensureActiveRequest();
  if (!item) return;

  // Validate target URL
  const targetUrl = els.replayTargetUrl?.value.trim();
  if (!targetUrl) {
    setReplayStatus(i18n.t('replay.errors.target_url_required') || 'Target URL is required', 'error');
    return;
  }

  // Parse headers
  let headers = {};
  const headersText = els.replayHeaders?.value.trim();
  if (headersText) {
    try {
      headers = JSON.parse(headersText);
    } catch (error) {
      setReplayStatus(i18n.t('replay.errors.invalid_headers') || 'Invalid headers JSON format', 'error');
      return;
    }
  }

  // Prepare request payload
  const replayPayload = {
    request_id: item.id,
    target_url: targetUrl,
    method: els.replayMethod?.value || item.method,
    headers: headers,
    body: els.replayBody?.value || '',
    query: els.replayQuery?.value || '',
  };

  // Disable submit button
  if (els.replaySubmit) {
    els.replaySubmit.disabled = true;
    els.replaySubmit.textContent = i18n.t('replay.status.sending') || 'Sending...';
  }

  setReplayStatus(i18n.t('replay.status.sending') || 'Sending replay request...', 'info');

  try {
    const response = await fetch(`${API_BASE}/replay`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(replayPayload),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(errorText || `HTTP ${response.status}`);
    }

    const result = await response.json();

    // Show success message with response details
    const statusMessage = i18n.t('replay.status.success', {
      status_code: result.status_code,
      response_time: result.response_time_ms,
    }) || `Replay successful! Status: ${result.status_code}, Time: ${result.response_time_ms}ms`;

    setReplayStatus(statusMessage, 'success');

    // Close modal after 2 seconds
    setTimeout(() => {
      closeReplayModal();
    }, 2000);

  } catch (error) {
    console.error('Replay failed:', error);
    const errorMessage = i18n.t('replay.errors.failed', {
      error: error.message,
    }) || `Replay failed: ${error.message}`;
    setReplayStatus(errorMessage, 'error');
  } finally {
    // Re-enable submit button
    if (els.replaySubmit) {
      els.replaySubmit.disabled = false;
      els.replaySubmit.textContent = i18n.t('replay.actions.submit') || 'Replay';
    }
  }
}

async function bootstrap() {
  await i18n.init();
  state.locale = i18n.getLocale();
  document.title = i18n.t('meta.app_title') || document.title;
  initLocaleSelector();
  initTheme();
  i18n.applyTranslations();
  await loadUser();
  await loadRequests();
  bindEvents();
  initWebsocket();
}

i18n.onChange((locale) => {
  state.locale = locale;
  i18n.applyTranslations();
  refreshLocaleUI();
});

bootstrap();
