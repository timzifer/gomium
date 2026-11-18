package cef

import "testing"

// TestSplitMediaType pins down what reaches CEF: set_mime_type matches only
// the bare type, and a "text/html; charset=utf-8" would leave the page blank
// at status 200 - a failure with nothing to see in the log.
func TestSplitMediaType(t *testing.T) {
	cases := []struct {
		value       string
		wantType    string
		wantCharset string
	}{
		{"text/html", "text/html", ""},
		{"text/html; charset=utf-8", "text/html", "utf-8"},
		{"text/css;charset=UTF-8", "text/css", "UTF-8"},
		{`text/plain; charset="utf-8"`, "text/plain", "utf-8"},
		{"text/html; boundary=x; charset=iso-8859-1", "text/html", "iso-8859-1"},
		{"application/json; boundary=x", "application/json", ""},
		{"", "", ""},
	}

	for _, testCase := range cases {
		mimeType, charset := splitMediaType(testCase.value)
		if mimeType != testCase.wantType || charset != testCase.wantCharset {
			t.Errorf("splitMediaType(%q) = (%q, %q), want (%q, %q)",
				testCase.value, mimeType, charset, testCase.wantType, testCase.wantCharset)
		}
	}
}
