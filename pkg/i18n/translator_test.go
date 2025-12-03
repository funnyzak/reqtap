package i18n

import (
	"sync"
	"testing"
)

func TestTranslatorText(t *testing.T) {
	tr, err := NewTranslator("en")
	if err != nil {
		t.Fatalf("NewTranslator failed: %v", err)
	}

	// Test existing zh-CN translation
	if got := tr.Text("zh-CN", "cli.metadata.remote"); got != "来源" {
		t.Fatalf("expected zh-CN translation, got %s", got)
	}

	// Test new language translations
	if got := tr.Text("ja", "cli.metadata.remote"); got != "リモート" {
		t.Fatalf("expected ja translation, got %s", got)
	}
	if got := tr.Text("ko", "cli.metadata.remote"); got != "원격" {
		t.Fatalf("expected ko translation, got %s", got)
	}
	if got := tr.Text("fr", "cli.metadata.remote"); got != "Distant" {
		t.Fatalf("expected fr translation, got %s", got)
	}
	if got := tr.Text("ru", "cli.metadata.remote"); got != "Удаленный" {
		t.Fatalf("expected ru translation, got %s", got)
	}

	// Test fallback to default locale for unsupported language
	if got := tr.Text("de", "cli.metadata.remote"); got != "Remote" {
		t.Fatalf("expected fallback to default locale, got %s", got)
	}

	// Test other translations
	if got := tr.Text("ja", "cli.headers.redacted"); got != "[非表示]" {
		t.Fatalf("expected ja headers redacted translation, got %s", got)
	}

	// Test non-existent translation key
	if got := tr.Text("en", "non.existent.key"); got != "non.existent.key" {
		t.Fatalf("expected key returned for non-existent translation, got %s", got)
	}

	// Test empty key
	if got := tr.Text("en", ""); got != "" {
		t.Fatalf("expected empty string for empty key, got %s", got)
	}
}

func TestTranslatorSupported(t *testing.T) {
	tr, err := NewTranslator("en")
	if err != nil {
		t.Fatalf("NewTranslator failed: %v", err)
	}

	supported := tr.Supported()
	expected := []string{"en", "fr", "ja", "ko", "ru", "zh-CN"}

	if len(supported) != len(expected) {
		t.Fatalf("expected %d supported locales, got %d", len(expected), len(supported))
	}

	for i, loc := range supported {
		if loc != expected[i] {
			t.Fatalf("expected locale %s at position %d, got %s", expected[i], i, loc)
		}
	}
}

func TestTranslatorDefaultLocale(t *testing.T) {
	tr, err := NewTranslator("ja")
	if err != nil {
		t.Fatalf("NewTranslator failed: %v", err)
	}

	if got := tr.DefaultLocale(); got != "ja" {
		t.Fatalf("expected default locale ja, got %s", got)
	}
}

func TestTranslatorErrorHandling(t *testing.T) {
	// Test with non-existent default locale
	_, err := NewTranslator("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent default locale")
	}
}

func TestTranslatorConcurrency(t *testing.T) {
	tr, err := NewTranslator("en")
	if err != nil {
		t.Fatalf("NewTranslator failed: %v", err)
	}

	var wg sync.WaitGroup
	concurrency := 100

	// Test concurrent reads
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = tr.Text("ja", "cli.metadata.remote")
			_ = tr.Supported()
			_ = tr.DefaultLocale()
		}(i)
	}

	wg.Wait()
}
