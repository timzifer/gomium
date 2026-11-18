package cefdl

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// download fetches the archive into cacheDir, verifying its checksum. A cached
// file whose SHA1 still matches is reused.
func download(ctx context.Context, o Options, file File, cacheDir string) (string, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}
	destination := filepath.Join(cacheDir, file.Name)

	if sum, err := sha1File(destination); err == nil && sum == file.SHA1 {
		o.logf("using cached %s", file.Name)
		return destination, nil
	}

	o.logf("downloading %s (%d MiB)", file.Name, file.Size>>20)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, FileURL+url.PathEscape(file.Name), nil)
	if err != nil {
		return "", err
	}
	response, err := o.client().Do(request)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: unexpected status %s", response.Status)
	}

	temporary := destination + ".part"
	out, err := os.Create(temporary)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", temporary, err)
	}
	hash := sha1.New()
	_, err = io.Copy(io.MultiWriter(out, hash), response.Body)
	closeErr := out.Close()
	if err != nil {
		os.Remove(temporary)
		return "", fmt.Errorf("download: %w", err)
	}
	if closeErr != nil {
		os.Remove(temporary)
		return "", closeErr
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != file.SHA1 {
		os.Remove(temporary)
		return "", fmt.Errorf("checksum mismatch: got %s, want %s", got, file.SHA1)
	}
	if err := os.Rename(temporary, destination); err != nil {
		return "", fmt.Errorf("finalize download: %w", err)
	}
	return destination, nil
}

func sha1File(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// extract unpacks the CEF archive. Release/ and Resources/ are flattened into
// dest (that is the layout cgo and the running binary expect); include/ is
// only unpacked when headerDest is set, because a gomium checkout carries
// local header patches. The license notices from the archive root land in
// dest as well (see licenseFiles) - they are part of what has to ship.
//
// Regular files and symlinks are both unpacked: the macOS distribution ships
// its library as a framework bundle, and a bundle without its symlinks is not
// a bundle.
func extract(archive, dest, headerDest string) (int, error) {
	file, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := tar.NewReader(bzip2.NewReader(file))
	written := 0
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return written, fmt.Errorf("read archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeSymlink {
			continue
		}
		target, ok := mapPath(header.Name, dest, headerDest)
		if !ok {
			continue
		}
		if header.Typeflag == tar.TypeSymlink {
			// The macOS distribution ships a framework bundle, and a bundle
			// is held together by symlinks (Versions/Current, and the
			// Resources / Libraries entries pointing through it). Copying the
			// link targets instead would double the size and break dlopen on
			// the bundle path.
			if err := writeSymlink(target, header.Linkname); err != nil {
				return written, err
			}
			written++
			continue
		}
		if err := writeFile(target, reader, header.FileInfo().Mode()); err != nil {
			return written, err
		}
		written++
	}
	if written == 0 {
		return 0, fmt.Errorf("archive contained no Release/Resources files")
	}
	return written, nil
}

// licenseFiles maps the notices at the root of a CEF distribution to their
// name in dest. Both are redistribution obligations: LICENSE.txt is CEF's
// BSD license, CREDITS.html the licenses of Chromium and every third-party
// component compiled into libcef. LICENSE.txt is renamed so it cannot be
// mistaken for - or overwrite - the license of the application shipping it.
var licenseFiles = map[string]string{
	"LICENSE.txt":  "CEF_LICENSE.txt",
	"CREDITS.html": "CREDITS.html",
}

// mapPath turns an archive path such as
// "cef_binary_151.../Release/libcef.dll" into a destination path, or reports
// false when the entry is not wanted.
func mapPath(name, dest, headerDest string) (string, bool) {
	name = path.Clean(strings.ReplaceAll(name, `\`, "/"))
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		if renamed, ok := licenseFiles[parts[1]]; ok {
			return filepath.Join(dest, renamed), true
		}
		return "", false
	}
	if len(parts) < 3 {
		return "", false
	}
	rest := parts[2:] // strip "<cef_binary_...>/<top>/"
	for _, part := range rest {
		if part == ".." {
			return "", false
		}
	}
	switch parts[1] {
	case "Release", "Resources":
		return filepath.Join(append([]string{dest}, rest...)...), true
	case "include":
		if headerDest == "" {
			return "", false
		}
		return filepath.Join(append([]string{headerDest}, rest...)...), true
	default:
		return "", false
	}
}

func writeFile(target string, reader io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	// Write via a temp file so a half-written DLL can never be left behind and
	// so replacing a file that is currently mapped fails loudly instead of
	// producing a corrupt library.
	temporary := target + ".part"
	out, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return fmt.Errorf("create %s: %w", temporary, err)
	}
	_, err = io.Copy(out, reader)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(temporary)
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := os.Rename(temporary, target); err != nil {
		os.Remove(temporary)
		return fmt.Errorf("replace %s: %w (is the binary still running?)", target, err)
	}
	return nil
}

// writeSymlink recreates a symlink from the archive. Only relative links that
// stay inside the destination are accepted - an archive must not be able to
// plant a link into the rest of the file system.
func writeSymlink(target, linkname string) error {
	clean := path.Clean(strings.ReplaceAll(linkname, `\`, "/"))
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("refusing symlink %s -> %s: leaves the destination", target, linkname)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	// A symlink cannot be replaced in place, and the old one may point
	// somewhere else entirely after a version change.
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("replace %s: %w", target, err)
	}
	if err := os.Symlink(filepath.FromSlash(clean), target); err != nil {
		return fmt.Errorf("link %s -> %s: %w", target, clean, err)
	}
	return nil
}
