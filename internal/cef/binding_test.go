package cef

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
}

func TestNewBindingFuncAcceptsSupportedShapes(t *testing.T) {
	cases := map[string]any{
		"mde.a": func() {},
		"mde.b": func() error { return nil },
		"mde.c": func() string { return "" },
		"mde.d": func(float64) (int, error) { return 0, nil },
	}
	for path, fn := range cases {
		if _, err := newBindingFunc(path, fn); err != nil {
			t.Errorf("%s: unexpected error %v", path, err)
		}
	}
}

func TestNewBindingFuncRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		path string
		fn   any
	}{
		{"path without dot", "mde", func() {}},
		{"path with two dots", "a.b.c", func() {}},
		{"empty root", ".x", func() {}},
		{"not a function", "mde.x", 42},
		{"nil function", "mde.x", (func())(nil)},
		{"three results", "mde.x", func() (int, int, error) { return 0, 0, nil }},
		{"second result not error", "mde.x", func() (int, string) { return 0, "" }},
	}
	for _, c := range cases {
		if _, err := newBindingFunc(c.path, c.fn); err == nil {
			t.Errorf("%s: expected an error", c.name)
		}
	}
}

func TestWithBindingReportsDuplicatesAndBadSignatures(t *testing.T) {
	app := &cefApp{}
	WithBinding("mde.ping", func() string { return "pong" })(app)
	WithBinding("mde.ping", func() string { return "pong" })(app)
	WithBinding("mde.bad", func() (int, int) { return 0, 0 })(app)

	if len(app.options.bindings) != 1 {
		t.Errorf("bindings: %d, want 1", len(app.options.bindings))
	}
	if len(app.options.optionErrors) != 2 {
		t.Errorf("option errors: %v, want two", app.options.optionErrors)
	}
}

func TestInvokeConvertsArgumentsAndResults(t *testing.T) {
	add, err := newBindingFunc("mde.add", func(a, b float64) (float64, error) { return a + b, nil })
	if err != nil {
		t.Fatal(err)
	}
	result, err := add.invoke([]*Exported{
		{Kind: exportedFloat, Float: ptr(2.0)},
		{Kind: exportedFloat, Float: ptr(3.0)},
	})
	if err != nil || result != 5.0 {
		t.Errorf("add: %v, %v; want 5", result, err)
	}

	if _, err := add.invoke([]*Exported{{Kind: exportedFloat, Float: ptr(1.0)}}); err == nil {
		t.Error("wrong argument count must fail")
	}
	if _, err := add.invoke([]*Exported{
		{Kind: exportedString, String: ptr("x")},
		{Kind: exportedFloat, Float: ptr(1.0)},
	}); err == nil {
		t.Error("string for float64 must fail")
	}
}

func TestInvokeVariadicTakesSurplusArguments(t *testing.T) {
	join, err := newBindingFunc("mde.join", func(sep string, parts ...string) string {
		return sep + strings.Join(parts, sep)
	})
	if err != nil {
		t.Fatal(err)
	}
	str := func(s string) *Exported { return &Exported{Kind: exportedString, String: ptr(s)} }

	result, err := join.invoke([]*Exported{str("-"), str("a"), str("b")})
	if err != nil || result != "-a-b" {
		t.Errorf("two parts: %v, %v; want -a-b", result, err)
	}
	result, err = join.invoke([]*Exported{str("-")})
	if err != nil || result != "-" {
		t.Errorf("no parts: %v, %v; want -", result, err)
	}
	if _, err := join.invoke(nil); err == nil {
		t.Error("missing fixed argument must fail")
	}
	if _, err := join.invoke([]*Exported{str("-"), {Kind: exportedFloat, Float: ptr(1.0)}}); err == nil {
		t.Error("float for a string part must fail")
	}
}

func TestInvokeReturnsErrorsAndRecoversPanics(t *testing.T) {
	fail, _ := newBindingFunc("mde.fail", func() error { return errors.New("boom") })
	if _, err := fail.invoke(nil); err == nil || err.Error() != "boom" {
		t.Errorf("fail: %v, want boom", err)
	}

	boom, _ := newBindingFunc("mde.boom", func() string { panic("kaputt") })
	if _, err := boom.invoke(nil); err == nil {
		t.Error("panic must become an error")
	}
}

func TestSameOrigin(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"http://intranet-host/app", "http://intranet-host/", true},
		{"http://intranet-host:80/", "http://intranet-host/", true},
		{"https://intranet-host/", "http://intranet-host/", false},
		{"http://intranet-host:8080/", "http://intranet-host/", false},
		{"http://evil.example/", "http://intranet-host/", false},
		{"file:///C:/videos/a.mp4", "http://intranet-host/", false},
		{"help://de_DE/index.html", "http://intranet-host/", false},
	}
	for _, c := range cases {
		if got := SameOrigin(mustURL(t, c.a), mustURL(t, c.b)); got != c.want {
			t.Errorf("SameOrigin(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
	if SameOrigin(nil, mustURL(t, "http://x/")) {
		t.Error("nil address must not match")
	}
}

func TestBindingGuardFallsBackToDefault(t *testing.T) {
	allow := func(BindingCall) bool { return true }
	deny := func(BindingCall) bool { return false }

	app := &cefApp{}
	if bindingGuardFor(app, &client{}) != nil {
		t.Error("no guard anywhere must yield nil")
	}

	WithDefaultBindingGuard(allow)(app)
	if bindingGuardFor(app, &client{}) == nil {
		t.Error("client without guard must use the default")
	}

	own := &client{bindingGuard: deny}
	if bindingGuardFor(app, own) == nil || bindingGuardFor(app, own)(BindingCall{}) {
		t.Error("client guard must win over the default")
	}
}
