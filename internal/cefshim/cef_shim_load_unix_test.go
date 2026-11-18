//go:build darwin || linux

package cefshim

import (
	"path/filepath"
	"testing"
)

func TestResolveLibcefPath(t *testing.T) {
	const framework = "Chromium Embedded Framework"

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "framework bundle",
			in:   filepath.Join("Release", framework+".framework"),
			want: filepath.Join("Release", framework+".framework", framework),
		},
		{
			name: "framework bundle with trailing separator",
			in:   filepath.Join("Release", framework+".framework") + string(filepath.Separator),
			want: filepath.Join("Release", framework+".framework", framework),
		},
		{
			name: "library inside the bundle stays untouched",
			in:   filepath.Join("Release", framework+".framework", framework),
			want: filepath.Join("Release", framework+".framework", framework),
		},
		{
			name: "shared object stays untouched",
			in:   filepath.Join("Release", "libcef.so"),
			want: filepath.Join("Release", "libcef.so"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveLibcefPath(tc.in); got != tc.want {
				t.Errorf("resolveLibcefPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
