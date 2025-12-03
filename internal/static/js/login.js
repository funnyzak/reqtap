import { createI18n } from './i18n.js';

const CONFIG = window.__REQTAP__ || {};
const API_BASE = CONFIG.apiBase || '/api';
const WEB_BASE = CONFIG.webBase || '/web';
const AUTH_ENABLED = CONFIG.authEnabled !== false;
const THEME_STORAGE_KEY = 'reqtap-theme';
const DEFAULT_THEME = 'dark';
const i18n = createI18n({
  defaultLocale: CONFIG.defaultLocale || 'en',
  supportedLocales: CONFIG.supportedLocales || ['en'],
  webBase: WEB_BASE,
});

const form = document.getElementById('login-form');
const message = document.getElementById('login-message');
const themeToggle = document.getElementById('theme-toggle');
const themeToggleLabel = document.getElementById('theme-toggle-label');
const themeToggleIcon = document.getElementById('theme-toggle-icon');
const localeSelect = document.getElementById('login-locale-select');
let currentTheme = DEFAULT_THEME;

function getStoredTheme() {
  try {
    return localStorage.getItem(THEME_STORAGE_KEY);
  } catch (error) {
    return null;
  }
}

function persistTheme(theme) {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch (error) {
    // ignore storage errors
  }
}

function updateThemeToggleUI(theme) {
  if (!themeToggle) {
    return;
  }
  const isLight = theme === 'light';
  themeToggle.setAttribute('aria-pressed', String(isLight));
  const nextThemeLabel = isLight ? i18n.t('header.theme.dark') : i18n.t('header.theme.light');
  themeToggle.setAttribute('title', i18n.t('header.theme.switch_to', { mode: nextThemeLabel }));
  if (themeToggleLabel) {
    themeToggleLabel.textContent = nextThemeLabel;
  }
  if (themeToggleIcon) {
    themeToggleIcon.className = `fa-solid ${isLight ? 'fa-moon' : 'fa-sun'}`;
  }
}

function applyTheme(theme) {
  const resolved = theme === 'light' ? 'light' : 'dark';
  currentTheme = resolved;
  document.documentElement.setAttribute('data-theme', resolved);
  updateThemeToggleUI(resolved);
}

function initThemeToggle() {
  const saved = getStoredTheme();
  applyTheme(saved || DEFAULT_THEME);
  if (themeToggle) {
    themeToggle.addEventListener('click', () => {
      const nextTheme = currentTheme === 'light' ? 'dark' : 'light';
      applyTheme(nextTheme);
      persistTheme(nextTheme);
    });
  }
}

function initLocaleSelector() {
  if (!localeSelect) {
    return;
  }
  const locales = i18n.getSupportedLocales();
  localeSelect.innerHTML = '';
  locales.forEach((loc) => {
    const option = document.createElement('option');
    option.value = loc;
    option.textContent = i18n.t(`header.locale_label.${loc}`) || loc;
    localeSelect.appendChild(option);
  });
  localeSelect.value = i18n.getLocale();
  localeSelect.addEventListener('change', async (event) => {
    await i18n.setLocale(event.target.value);
  });
}

function refreshLocaleUI() {
  document.title = i18n.t('meta.login_title') || document.title;
  if (localeSelect) {
    Array.from(localeSelect.options).forEach((option) => {
      option.textContent = i18n.t(`header.locale_label.${option.value}`) || option.value;
    });
  }
  updateThemeToggleUI(currentTheme);
}

function getWebHomeURL() {
  return WEB_BASE === '/' ? '/' : `${WEB_BASE}/`;
}

async function checkSession() {
  if (!AUTH_ENABLED) {
    window.location.href = getWebHomeURL();
    return;
  }

  try {
    const resp = await fetch(`${API_BASE}/auth/me`, { credentials: 'include' });
    if (resp.ok) {
      window.location.href = getWebHomeURL();
    }
  } catch {
    // ignore
  }
}

async function handleLogin(event) {
  event.preventDefault();
  message.textContent = '';

  const formData = new FormData(form);
  const payload = {
    username: formData.get('username'),
    password: formData.get('password'),
  };

  try {
    const resp = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(text || i18n.t('login.message_failed'));
    }

    window.location.href = getWebHomeURL();
  } catch (error) {
    message.textContent = error.message || i18n.t('login.message_failed');
  }
}

async function bootstrap() {
  await i18n.init();
  document.title = i18n.t('meta.login_title') || document.title;
  i18n.applyTranslations();
  initLocaleSelector();
  initThemeToggle();
  form.addEventListener('submit', handleLogin);
  checkSession();
}

i18n.onChange(() => {
  i18n.applyTranslations();
  refreshLocaleUI();
});

bootstrap();
