const CONFIG = window.__REQTAP__ || {};
const API_BASE = CONFIG.apiBase || '/api';
const WEB_BASE = CONFIG.webBase || '/web';
const AUTH_ENABLED = CONFIG.authEnabled !== false;

const form = document.getElementById('login-form');
const message = document.getElementById('login-message');

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
checkSession();
