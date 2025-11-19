const CONFIG = window.__REQTAP__ || {};
const API_BASE = CONFIG.apiBase || '/api';
const WEB_BASE = CONFIG.webBase || '/web';
const AUTH_ENABLED = CONFIG.authEnabled !== false;
const THEME_STORAGE_KEY = 'reqtap-theme';
const DEFAULT_THEME = 'dark';

const form = document.getElementById('login-form');
const message = document.getElementById('login-message');
const themeToggle = document.getElementById('theme-toggle');
const themeToggleLabel = document.getElementById('theme-toggle-label');
const themeToggleIcon = document.getElementById('theme-toggle-icon');
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
  themeToggle.setAttribute('title', `Switch to ${isLight ? 'dark' : 'light'} mode`);
  if (themeToggleLabel) {
    themeToggleLabel.textContent = isLight ? 'Dark' : 'Light';
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
      throw new Error(text || 'Login failed');
    }

    window.location.href = getWebHomeURL();
  } catch (error) {
    message.textContent = error.message || 'Login failed';
  }
}

form.addEventListener('submit', handleLogin);
initThemeToggle();
checkSession();
