package cef

import (
	"runtime"
	"strings"
	"testing"
)

// TestWithCustomSchemeRejectsUnusableNames pins the validation down: an
// Option cannot report an error itself, so a bad scheme name would otherwise
// only surface much later as a page failing with ERR_UNKNOWN_URL_SCHEME.
func TestWithCustomSchemeRejectsUnusableNames(t *testing.T) {
	runtime.LockOSThread()

	cases := []struct {
		name    string
		options []Option
		wants   string
	}{
		{
			name:    "upper case",
			options: []Option{WithCustomScheme("Help", SchemeStandard)},
			wants:   "lower-case",
		},
		{
			name:    "empty",
			options: []Option{WithCustomScheme("", SchemeStandard)},
			wants:   "lower-case",
		},
		{
			name:    "with separator",
			options: []Option{WithCustomScheme("help://", SchemeStandard)},
			wants:   "lower-case",
		},
		{
			name:    "built-in",
			options: []Option{WithCustomScheme("https", SchemeStandard)},
			wants:   "built-in scheme",
		},
		{
			name: "registered twice",
			options: []Option{
				WithCustomScheme("help", SchemeStandard),
				WithCustomScheme("help", SchemeSecure),
			},
			wants: "already registered",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			app, err := New(testCase.options...)
			if err == nil {
				_ = app.Close()
				t.Fatalf("New() accepted %s", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.wants) {
				t.Errorf("New() error = %q, want it to mention %q", err, testCase.wants)
			}
		})
	}
}

func TestWithCustomSchemeKeepsNameAndOptions(t *testing.T) {
	runtime.LockOSThread()

	app, err := New(WithCustomScheme("help", SchemeStandard|SchemeSecure|SchemeFetchEnabled))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		_ = app.Close()
	}()

	schemes := app.(*cefApp).options.customSchemes
	if len(schemes) != 1 {
		t.Fatalf("customSchemes = %v, want exactly one", schemes)
	}
	if schemes[0].Name != "help" {
		t.Errorf("scheme name = %q, want %q", schemes[0].Name, "help")
	}
	if want := SchemeStandard | SchemeSecure | SchemeFetchEnabled; schemes[0].Options != want {
		t.Errorf("scheme options = %d, want %d", schemes[0].Options, want)
	}
}
