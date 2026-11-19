package cefdl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Build is one entry of the CDN index.
type Build struct {
	CEFVersion      string `json:"cef_version"`
	ChromiumVersion string `json:"chromium_version"`
	Channel         string `json:"channel"`
	Files           []File `json:"files"`
}

// File is one downloadable archive of a build.
type File struct {
	Type string `json:"type"`
	Name string `json:"name"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
}

type index map[string]struct {
	Versions []Build `json:"versions"`
}

// List returns the builds the CDN offers for a platform, newest first. An
// empty platform means the platform this binary runs on; limit <= 0 returns
// all of them.
func List(ctx context.Context, platform string, limit int, client *http.Client) ([]Build, error) {
	if platform == "" {
		platform = DefaultPlatform()
	}
	if platform == "" {
		return nil, fmt.Errorf("no CEF platform key for this system")
	}
	idx, err := fetchIndex(ctx, client)
	if err != nil {
		return nil, err
	}
	entry, ok := idx[platform]
	if !ok {
		return nil, unknownPlatform(platform, idx)
	}
	if limit > 0 && limit < len(entry.Versions) {
		return entry.Versions[:limit], nil
	}
	return entry.Versions, nil
}

// resolve picks the build and archive matching the request.
func resolve(ctx context.Context, o Options) (Build, File, error) {
	idx, err := fetchIndex(ctx, o.client())
	if err != nil {
		return Build{}, File{}, err
	}
	entry, ok := idx[o.Platform]
	if !ok {
		return Build{}, File{}, unknownPlatform(o.Platform, idx)
	}
	for _, build := range entry.Versions {
		if !matches(build, o.Version) {
			continue
		}
		for _, file := range build.Files {
			if file.Type == o.archiveType() {
				return build, file, nil
			}
		}
		return Build{}, File{}, fmt.Errorf("build %s has no %q archive", build.CEFVersion, o.archiveType())
	}
	return Build{}, File{}, fmt.Errorf("no build matching %q for platform %s", o.Version, o.Platform)
}

func fetchIndex(ctx context.Context, client *http.Client) (index, error) {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, IndexURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch index: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch index: unexpected status %s", response.Status)
	}
	var idx index
	if err := json.NewDecoder(response.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}
	return idx, nil
}

func unknownPlatform(platform string, idx index) error {
	known := make([]string, 0, len(idx))
	for key := range idx {
		known = append(known, key)
	}
	sort.Strings(known)
	return fmt.Errorf("unknown platform %q (known: %s)", platform, strings.Join(known, ", "))
}

// matches resolves the requested version against one index entry. Besides an
// exact version string it accepts a branch prefix ("151") and the channel
// names "stable" and "beta", which select the newest entry the index lists for
// that channel.
func matches(build Build, want string) bool {
	switch want {
	case "stable", "beta":
		return build.Channel == want
	}
	if build.CEFVersion == want {
		return true
	}
	// Branch prefix, e.g. "151" or "151.3": only match on a version-component
	// boundary so that "15" does not match "151.x".
	return strings.HasPrefix(build.CEFVersion, want+".")
}

// DefaultPlatform is the CEF platform key for the system this binary runs on,
// or "" when CEF has no build for it.
func DefaultPlatform() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		return "windows64"
	case "windows/386":
		return "windows32"
	case "windows/arm64":
		return "windowsarm64"
	case "linux/amd64":
		return "linux64"
	case "linux/386":
		return "linux32"
	case "linux/arm":
		return "linuxarm"
	case "linux/arm64":
		return "linuxarm64"
	case "darwin/amd64":
		return "macosx64"
	case "darwin/arm64":
		return "macosarm64"
	default:
		return ""
	}
}
