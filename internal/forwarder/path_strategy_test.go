package forwarder

import "testing"

func TestPathStrategyStripPrefix(t *testing.T) {
	ps := newPathStrategy(PathStrategyOptions{
		Mode:        "strip_prefix",
		StripPrefix: "/api",
	}, nil)
	if ps == nil {
		t.Fatal("expected non-nil path strategy")
	}

	path, rule := ps.resolve("/api/v1/users")
	if path != "/v1/users" || rule == "" {
		t.Fatalf("expected stripped path, got %s rule %s", path, rule)
	}
}

func TestPathStrategyRewritePrefix(t *testing.T) {
	ps := newPathStrategy(PathStrategyOptions{
		Mode: "rewrite",
		Rules: []RewriteRuleOption{
			{Name: "svc", Match: "/service", Replace: "/backend"},
		},
	}, nil)
	if ps == nil {
		t.Fatal("expected non-nil path strategy")
	}

	path, rule := ps.resolve("/service/foo")
	if path != "/backend/foo" || rule != "svc" {
		t.Fatalf("unexpected rewrite result path=%s rule=%s", path, rule)
	}
}

func TestPathStrategyRegex(t *testing.T) {
	ps := newPathStrategy(PathStrategyOptions{
		Mode: "rewrite",
		Rules: []RewriteRuleOption{
			{Name: "regex", Match: `^/tenant/(.*)$`, Replace: "/$1", Regex: true},
		},
	}, nil)
	if ps == nil {
		t.Fatal("expected non-nil path strategy")
	}

	path, rule := ps.resolve("/tenant/acme/orders")
	if path != "/acme/orders" || rule != "regex" {
		t.Fatalf("unexpected regex rewrite path=%s rule=%s", path, rule)
	}
}

func TestPathStrategyAppendDefault(t *testing.T) {
	ps := newPathStrategy(PathStrategyOptions{}, nil)
	if ps != nil {
		t.Fatalf("expected nil strategy for default append mode")
	}
}
