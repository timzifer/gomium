package cefdl

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	fixturePath    = "testdata/cef_fixture.tar.bz2"
	fixtureVersion = "1.2.3+gabc+chromium-1.2.3.4"
	fixtureName    = "cef_binary_1.2.3+gabc+chromium-1.2.3.4_windows64_minimal.tar.bz2"
)

func fixtureSHA1(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum(content)
	return hex.EncodeToString(sum[:])
}

// testCDN serves an index plus the fixture archive, standing in for
// cef-builds.spotifycdn.com.
func testCDN(t *testing.T, platform string) (*httptest.Server, *http.Client) {
	t.Helper()
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		platform: map[string]any{
			"versions": []any{
				map[string]any{
					"cef_version":      fixtureVersion,
					"chromium_version": "1.2.3.4",
					"channel":          "stable",
					"files": []any{
						map[string]any{"type": "standard", "name": "irrelevant.tar.bz2", "sha1": "x", "size": 1},
						map[string]any{"type": ArchiveType, "name": fixtureName, "sha1": fixtureSHA1(t), "size": int64(len(content))},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "index.json") {
			json.NewEncoder(writer).Encode(body)
			return
		}
		writer.Write(content)
	}))
	t.Cleanup(server.Close)

	// Redirect both index and downloads at the test server.
	client := server.Client()
	client.Transport = rewriteTransport{base: server.URL, inner: client.Transport}
	return server, client
}

// rewriteTransport sends every request to the test server, whatever host the
// package asks for.
type rewriteTransport struct {
	base  string
	inner http.RoundTripper
}

func (r rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rewritten := request.Clone(request.Context())
	target := r.base + request.URL.EscapedPath()
	parsed, err := http.NewRequest(request.Method, target, nil)
	if err != nil {
		return nil, err
	}
	rewritten.URL = parsed.URL
	rewritten.Host = parsed.Host
	return r.inner.RoundTrip(rewritten)
}

func TestFetchUnpacksRuntime(t *testing.T) {
	platform := "windows64"
	_, client := testCDN(t, platform)

	root := t.TempDir()
	dest := filepath.Join(root, "Release")
	result, err := Fetch(context.Background(), Options{
		Version:  fixtureVersion,
		Platform: platform,
		Dest:     dest,
		CacheDir: filepath.Join(root, "cache"),
		Client:   client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != fixtureVersion || result.Skipped {
		t.Fatalf("unexpected result: %+v", result)
	}

	// Release/ and Resources/ are flattened into dest.
	for _, name := range []string{"libcef.dll", "chrome_elf.dll", "icudtl.dat", filepath.Join("locales", "de.pak")} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
	// Headers stay untouched unless asked for, and stray files are ignored.
	if _, err := os.Stat(filepath.Join(root, "include")); err == nil {
		t.Fatal("include/ must not be unpacked without HeaderDest")
	}
	if _, err := os.Stat(filepath.Join(dest, "README.txt")); err == nil {
		t.Fatal("only Release/ and Resources/ belong in the destination")
	}

	marker, err := os.ReadFile(filepath.Join(dest, MarkerFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(marker)) != fixtureVersion {
		t.Fatalf("marker holds %q", marker)
	}

	// A second call sees the marker and does nothing.
	again, err := Fetch(context.Background(), Options{
		Version:  fixtureVersion,
		Platform: platform,
		Dest:     dest,
		CacheDir: filepath.Join(root, "cache"),
		Client:   client,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Skipped || again.Files != 0 {
		t.Fatalf("second call should be a no-op: %+v", again)
	}
}

func TestFetchUnpacksHeadersOnRequest(t *testing.T) {
	platform := "windows64"
	_, client := testCDN(t, platform)

	root := t.TempDir()
	if _, err := Fetch(context.Background(), Options{
		Version:    "1.2",
		Platform:   platform,
		Dest:       filepath.Join(root, "Release"),
		CacheDir:   filepath.Join(root, "cache"),
		HeaderDest: filepath.Join(root, "include"),
		Client:     client,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "include", "cef_version.h")); err != nil {
		t.Fatalf("header missing: %v", err)
	}
}

func TestFetchRejectsChecksumMismatch(t *testing.T) {
	platform := "windows64"
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{%q:{"versions":[{"cef_version":%q,"channel":"stable","files":[{"type":%q,"name":%q,"sha1":"0000000000000000000000000000000000000000","size":%d}]}]}}`,
		platform, fixtureVersion, ArchiveType, fixtureName, len(content))

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "index.json") {
			writer.Write([]byte(body))
			return
		}
		writer.Write(content)
	}))
	defer server.Close()
	client := server.Client()
	client.Transport = rewriteTransport{base: server.URL, inner: client.Transport}

	root := t.TempDir()
	_, err = Fetch(context.Background(), Options{
		Version:  fixtureVersion,
		Platform: platform,
		Dest:     filepath.Join(root, "Release"),
		CacheDir: filepath.Join(root, "cache"),
		Client:   client,
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("a wrong checksum must fail loudly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "Release", "libcef.dll")); err == nil {
		t.Fatal("nothing may be unpacked from an archive that failed its checksum")
	}
}

func TestFetchValidatesOptions(t *testing.T) {
	if _, err := Fetch(context.Background(), Options{Dest: t.TempDir()}); err == nil {
		t.Fatal("a missing version must be an error")
	}
	if _, err := Fetch(context.Background(), Options{Version: "151"}); err == nil {
		t.Fatal("a missing destination must be an error")
	}
}

func TestList(t *testing.T) {
	platform := "windows64"
	_, client := testCDN(t, platform)

	builds, err := List(context.Background(), platform, 5, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 1 || builds[0].CEFVersion != fixtureVersion {
		t.Fatalf("unexpected builds: %+v", builds)
	}

	if _, err := List(context.Background(), "amiga68k", 5, client); err == nil {
		t.Fatal("an unknown platform must be an error")
	}
}

func TestMatches(t *testing.T) {
	build := Build{CEFVersion: "151.3.14+g5d67476+chromium-151.0.7922.72", Channel: "stable"}
	for _, want := range []string{"151.3.14+g5d67476+chromium-151.0.7922.72", "151", "151.3", "stable"} {
		if !matches(build, want) {
			t.Fatalf("%q should match", want)
		}
	}
	for _, want := range []string{"15", "152", "beta", "151.4"} {
		if matches(build, want) {
			t.Fatalf("%q should not match", want)
		}
	}
}

func TestMapPath(t *testing.T) {
	dest := filepath.Join("out", "Release")
	cases := []struct {
		name    string
		headers string
		want    string
		ok      bool
	}{
		{"cef_binary_1/Release/libcef.dll", "", filepath.Join(dest, "libcef.dll"), true},
		{"cef_binary_1/Resources/locales/de.pak", "", filepath.Join(dest, "locales", "de.pak"), true},
		{"cef_binary_1/include/cef_version.h", "", "", false},
		{"cef_binary_1/include/cef_version.h", "inc", filepath.Join("inc", "cef_version.h"), true},
		{"cef_binary_1/Debug/libcef.dll", "", "", false},
		{"cef_binary_1/Release", "", "", false},
		{"cef_binary_1/Release/../../escape.dll", "", "", false},
		{"cef_binary_1/LICENSE.txt", "", filepath.Join(dest, "CEF_LICENSE.txt"), true},
		{"cef_binary_1/CREDITS.html", "", filepath.Join(dest, "CREDITS.html"), true},
		{"cef_binary_1/README.txt", "", "", false},
	}
	for _, testCase := range cases {
		got, ok := mapPath(testCase.name, dest, testCase.headers)
		if ok != testCase.ok || (ok && got != testCase.want) {
			t.Fatalf("%s: got %q/%v, want %q/%v", testCase.name, got, ok, testCase.want, testCase.ok)
		}
	}
}

func TestWriteSymlinkRefusesEscape(t *testing.T) {
	target := filepath.Join(t.TempDir(), "link")
	if err := writeSymlink(target, "../../etc/passwd"); err == nil {
		t.Fatal("a link leaving the destination must be refused")
	}
	if err := writeSymlink(target, "/etc/passwd"); err == nil {
		t.Fatal("an absolute link must be refused")
	}
}

func TestDefaultPlatform(t *testing.T) {
	got := DefaultPlatform()
	switch runtime.GOOS {
	case "windows", "linux", "darwin":
		if got == "" {
			t.Fatalf("no platform key for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}
}

// The embedded pin is what tools/cefdl fetches by default - an empty or
// multi-line file would turn into a confusing index lookup failure.
func TestPinnedVersion(t *testing.T) {
	pinned := PinnedVersion()
	if pinned == "" {
		t.Fatal("cefdl/cef.version is empty")
	}
	if strings.ContainsAny(pinned, " \t\r\n") {
		t.Fatalf("pinned version %q contains whitespace", pinned)
	}
	if !strings.Contains(pinned, "+chromium-") {
		t.Fatalf("pinned version %q is not a full CEF version string", pinned)
	}
}

func TestReadVersionFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cef.version")
	if err := os.WriteFile(path, []byte(fixtureVersion+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	version, err := ReadVersionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if version != fixtureVersion {
		t.Fatalf("read %q", version)
	}

	empty := filepath.Join(t.TempDir(), "cef.version")
	if err := os.WriteFile(empty, []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVersionFile(empty); err == nil {
		t.Fatal("an empty version file must be an error")
	}
}

// The macOS distribution ships a framework bundle held together by symlinks,
// so extract recreates them. Creating a symlink needs privileges on Windows -
// skip there rather than fail on a machine without developer mode.
func TestWriteSymlinkRecreatesRelativeLink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "Current")
	if err := writeSymlink(target, "A"); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need privileges on Windows:", err)
		}
		t.Fatal(err)
	}
	link, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if link != "A" {
		t.Fatalf("link points at %q", link)
	}

	// A link change must replace the old one, not fail on it.
	if err := writeSymlink(target, "B"); err != nil {
		t.Fatal(err)
	}
	if link, _ := os.Readlink(target); link != "B" {
		t.Fatalf("link not replaced: %q", link)
	}
}
