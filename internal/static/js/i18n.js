const DEFAULT_STORAGE_KEY = 'reqtap-locale';

function normalizeLocale(value) {
  if (!value) return '';
  return value
    .trim()
    .replace('_', '-')
    .split('-')
    .map((part, index) => (index === 0 ? part.toLowerCase() : part.toUpperCase()))
    .join('-');
}

function resolveKey(dict, path) {
  if (!dict || !path) {
    return undefined;
  }
  return path.split('.').reduce((acc, segment) => {
    if (acc && Object.prototype.hasOwnProperty.call(acc, segment)) {
      return acc[segment];
    }
    return undefined;
  }, dict);
}

function formatTemplate(template, params = {}) {
  if (typeof template !== 'string') {
    return template;
  }
  return template.replace(/\{(\w+)\}/g, (match, token) => {
    if (Object.prototype.hasOwnProperty.call(params, token)) {
      return params[token];
    }
    return match;
  });
}

export function createI18n(config = {}) {
  const storageKey = config.storageKey || DEFAULT_STORAGE_KEY;
  const supportedLocales = (config.supportedLocales && config.supportedLocales.length
    ? config.supportedLocales
    : ['en']
  ).map((loc) => {
    const normalized = normalizeLocale(loc);
    return {
      original: loc,
      normalized,
      lower: normalized.toLowerCase(),
      base: normalized.split('-')[0],
    };
  });
  const defaultLocale = normalizeLocale(config.defaultLocale) || 'en';
  const webBase = config.webBase === '/' ? '' : config.webBase || '/web';
  const listeners = new Set();
  const state = {
    locale: defaultLocale,
    dict: {},
    fallbackDict: {},
  };

  function getSupportedLocales() {
    return supportedLocales.map((item) => item.original);
  }

  function findSupported(locale) {
    if (!locale) {
      return '';
    }

    const exactMatch = supportedLocales.find((entry) => entry.original === locale);
    if (exactMatch) {
      return exactMatch.original;
    }

    const lower = locale.toLowerCase();
    const lowerMatch = supportedLocales.find((entry) => entry.lower === lower);
    if (lowerMatch) {
      return lowerMatch.original;
    }

    const base = lower.split('-')[0];
    const baseMatch = supportedLocales.find((entry) => entry.base === base);
    return baseMatch ? baseMatch.original : '';
  }

  function resolveLocale(locale) {
    const normalized = normalizeLocale(locale);
    if (!normalized) {
      return '';
    }
    return findSupported(normalized) || '';
  }

  async function loadDictionary(locale) {
    const resolved = resolveLocale(locale) || findSupported(defaultLocale) || 'en';
    const path = `${webBase === '/' ? '' : webBase}/locales/${resolved}.json`;
    const resp = await fetch(path, { cache: 'no-cache' });
    if (!resp.ok) {
      throw new Error(`Failed to load locale ${resolved}`);
    }
    return resp.json();
  }

  function detectBrowserLocale() {
    const sources = [];
    if (Array.isArray(navigator.languages)) {
      sources.push(...navigator.languages);
    }
    if (navigator.language) {
      sources.push(navigator.language);
    }
    for (const candidate of sources) {
      const resolved = resolveLocale(candidate);
      if (resolved) {
        return resolved;
      }
    }
    return '';
  }

  function getStoredLocale() {
    try {
      return localStorage.getItem(storageKey) || '';
    } catch (error) {
      return '';
    }
  }

  function persistLocale(locale) {
    try {
      localStorage.setItem(storageKey, locale);
    } catch (error) {
      // ignore storage errors
    }
  }

  function t(key, params) {
    const value = resolveKey(state.dict, key);
    const fallback = resolveKey(state.fallbackDict, key);
    const template = typeof value === 'string' ? value : typeof fallback === 'string' ? fallback : key;
    return formatTemplate(template, params);
  }

  function applyTranslations(root = document) {
    if (!root || typeof root.querySelectorAll !== 'function') {
      return;
    }
    root.querySelectorAll('[data-i18n]').forEach((el) => {
      const key = el.getAttribute('data-i18n');
      if (!key) return;
      const text = t(key);
      if (typeof text === 'string') {
        el.textContent = text;
      }
    });

    const attrMappings = [
      { selector: 'data-i18n-placeholder', attr: 'placeholder' },
      { selector: 'data-i18n-title', attr: 'title' },
      { selector: 'data-i18n-aria-label', attr: 'aria-label' },
    ];

    attrMappings.forEach(({ selector, attr }) => {
      root.querySelectorAll(`[${selector}]`).forEach((el) => {
        const key = el.getAttribute(selector);
        if (!key) return;
        const text = t(key);
        if (typeof text === 'string') {
          el.setAttribute(attr, text);
        }
      });
    });
  }

  async function init() {
    const fallbackLocale = resolveLocale(defaultLocale) || 'en';
    try {
      state.fallbackDict = await loadDictionary(fallbackLocale);
    } catch (error) {
      state.fallbackDict = {};
    }

    const stored = resolveLocale(getStoredLocale());
    const browser = detectBrowserLocale();
    const initial = stored || browser || fallbackLocale;
    state.locale = resolveLocale(initial) || fallbackLocale;
    try {
      if (state.locale === fallbackLocale) {
        state.dict = state.fallbackDict;
      } else {
        state.dict = await loadDictionary(state.locale);
      }
    } catch (error) {
      state.dict = state.fallbackDict;
      state.locale = fallbackLocale;
    }
    document.documentElement.setAttribute('lang', state.locale.toLowerCase());
    applyTranslations();
    return state.locale;
  }

  async function setLocale(locale) {
    const resolved = resolveLocale(locale);
    if (!resolved || resolved === state.locale) {
      return state.locale;
    }
    let nextDict = state.fallbackDict;
    try {
      if (resolved === resolveLocale(defaultLocale)) {
        nextDict = state.fallbackDict;
      } else {
        nextDict = await loadDictionary(resolved);
      }
    } catch (error) {
      nextDict = state.fallbackDict;
    }
    state.locale = resolved;
    state.dict = resolved === resolveLocale(defaultLocale) ? state.fallbackDict : nextDict;
    persistLocale(resolved);
    document.documentElement.setAttribute('lang', state.locale.toLowerCase());
    applyTranslations();
    listeners.forEach((cb) => cb(state.locale));
    return state.locale;
  }

  function getLocale() {
    return state.locale;
  }

  function onChange(listener) {
    if (typeof listener === 'function') {
      listeners.add(listener);
    }
    return () => listeners.delete(listener);
  }

  return {
    init,
    t,
    setLocale,
    getLocale,
    getSupportedLocales,
    applyTranslations,
    onChange,
  };
}
