//go:build darwin

package cef

import (
	"path/filepath"
	"testing"
)

// TestEnclosingAppBundle pins the layout MacBundlePaths recognises. Only the
// exact "<Something>.app/Contents/MacOS/<exe>" shape counts - a looser match
// would hand back a bundle whose Frameworks directory has nothing to do with
// the running binary, and the resulting framework path would then point at
// somebody else's CEF.
func TestEnclosingAppBundle(t *testing.T) {
	for _, tc := range []struct {
		name string
		exe  string
		want string
	}{
		{
			name: "browser executable",
			exe:  "/Applications/MyApp.app/Contents/MacOS/MyApp",
			want: "/Applications/MyApp.app",
		},
		{
			name: "helper executable",
			exe:  "/Applications/MyApp.app/Contents/Frameworks/MyApp Helper (GPU).app/Contents/MacOS/MyApp Helper (GPU)",
			want: "/Applications/MyApp.app/Contents/Frameworks/MyApp Helper (GPU).app",
		},
		{
			name: "bare binary",
			exe:  "/Users/someone/build/myapp",
		},
		{
			name: "right depth, no .app suffix",
			exe:  "/Users/someone/MyApp/Contents/MacOS/MyApp",
		},
		{
			name: "inside a bundle but not in MacOS",
			exe:  "/Applications/MyApp.app/Contents/Resources/tool",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := enclosingAppBundle(tc.exe)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("enclosingAppBundle(%q) = %q, want an error", tc.exe, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("enclosingAppBundle(%q): %v", tc.exe, err)
			}
			if got != tc.want {
				t.Fatalf("enclosingAppBundle(%q) = %q, want %q", tc.exe, got, tc.want)
			}
		})
	}
}

// TestMacBundlePathsLayout checks the paths derived from a bundle, for the
// browser and the helper process alike: a helper has to climb out of its own
// bundle, because the framework it must dlopen lives in the outer one.
func TestMacBundlePathsLayout(t *testing.T) {
	const app = "/Applications/MyApp.app"
	frameworks := filepath.Join(app, "Contents", "Frameworks")

	wantFramework := filepath.Join(frameworks, MacFrameworkName+".framework")
	wantHelper := filepath.Join(frameworks, "MyApp Helper.app", "Contents", "MacOS", "MyApp Helper")

	for _, tc := range []struct {
		name string
		exe  string
	}{
		{"browser", filepath.Join(app, "Contents", "MacOS", "MyApp")},
		{"helper", wantHelper},
		{"gpu helper", filepath.Join(frameworks, "MyApp Helper (GPU).app", "Contents", "MacOS", "MyApp Helper (GPU)")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle, framework, helper, err := macBundlePathsFor(tc.exe)
			if err != nil {
				t.Fatalf("macBundlePathsFor(%q): %v", tc.exe, err)
			}
			if bundle != app {
				t.Errorf("bundle = %q, want %q", bundle, app)
			}
			if framework != wantFramework {
				t.Errorf("framework = %q, want %q", framework, wantFramework)
			}
			if helper != wantHelper {
				t.Errorf("helper = %q, want %q", helper, wantHelper)
			}
		})
	}
}
