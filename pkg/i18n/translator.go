package i18n

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// localeFiles embeds default translation resources.
//
//go:embed locales/*.yaml
var localeFiles embed.FS

// Translator 提供多语言文本查找功能。
type Translator struct {
	locales       map[string]map[string]string
	defaultLocale string
	mu            sync.RWMutex
}

// NewTranslator 从嵌入资源加载所有翻译。
func NewTranslator(defaultLocale string) (*Translator, error) {
	entries, err := localeFiles.ReadDir("locales")
	if err != nil {
		return nil, fmt.Errorf("read locales: %w", err)
	}

	locales := make(map[string]map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		data, err := localeFiles.ReadFile("locales/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", name, err)
		}
		parsed := make(map[string]interface{})
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", name, err)
		}
		locales[name] = flattenMap(parsed, "")
	}

	if defaultLocale == "" {
		defaultLocale = "en"
	}
	if _, ok := locales[defaultLocale]; !ok {
		return nil, fmt.Errorf("default locale %s missing", defaultLocale)
	}

	return &Translator{locales: locales, defaultLocale: defaultLocale}, nil
}

// Supported 返回支持的语言列表。
func (t *Translator) Supported() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	keys := make([]string, 0, len(t.locales))
	for key := range t.locales {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Text 返回指定 key 的翻译，找不到时返回 key 本身。
func (t *Translator) Text(locale, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if key == "" {
		return ""
	}

	lookupChain := []string{}
	if locale != "" {
		lookupChain = append(lookupChain, locale)
		if base := baseLocale(locale); base != locale {
			lookupChain = append(lookupChain, base)
		}
	}
	if t.defaultLocale != "" {
		lookupChain = append(lookupChain, t.defaultLocale)
	}

	for _, candidate := range lookupChain {
		if values, ok := t.locales[candidate]; ok {
			if val, ok := values[key]; ok {
				return val
			}
		}
	}

	return key
}

// DefaultLocale 返回当前默认语言。
func (t *Translator) DefaultLocale() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.defaultLocale
}

func baseLocale(locale string) string {
	locale = strings.ReplaceAll(locale, "_", "-")
	parts := strings.Split(locale, "-")
	if len(parts) > 1 {
		return parts[0]
	}
	return locale
}

func flattenMap(data map[string]interface{}, prefix string) map[string]string {
	out := make(map[string]string)
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]interface{}:
			nested := flattenMap(v, fullKey)
			for nk, nv := range nested {
				out[nk] = nv
			}
		case string:
			out[fullKey] = v
		case fmt.Stringer:
			out[fullKey] = v.String()
		default:
			out[fullKey] = fmt.Sprintf("%v", v)
		}
	}
	return out
}
