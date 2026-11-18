// Command cefbundle assembles a macOS application bundle around a Go binary
// that embeds gomium.
//
// CEF on macOS cannot run helper processes out of the main executable: it
// looks for them in bundles of their own, next to the CEF framework, inside
// the top-level .app. That layout is not something `go build` produces, and
// getting it wrong shows up as an unspecific cef_initialize failure. This
// tool builds it:
//
//	<App>.app/Contents/
//	  Info.plist
//	  MacOS/<App>                                     the -bin executable
//	  Resources/CEF_LICENSE.txt, CREDITS.html         copied from -cef, if present
//	  Frameworks/
//	    Chromium Embedded Framework.framework/        copied from -cef
//	    <App> Helper.app/Contents/{Info.plist,MacOS/<App> Helper}
//	    <App> Helper (GPU).app/…
//	    <App> Helper (Plugin).app/…
//	    <App> Helper (Renderer).app/…
//	    <App> Helper (Alerts).app/…
//
// All five helper executables are copies of the same binary. gomium's
// IsSubprocess routes them into ExecuteSubprocess, so one binary serves every
// process type; the five bundles exist because Chromium picks the helper by
// name per child-process type when browser_subprocess_path is left empty.
//
// Usage:
//
//	go build -o build/myapp ./cmd/myapp
//	go run ./tools/cefbundle -bin build/myapp -cef Release -out build/MyApp.app
//
// Re-running rebuilds the bundle from scratch, so it is safe to chain behind
// a build.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
)

// frameworkName is the base name of the CEF framework bundle, in the
// distribution and inside the application bundle alike.
const frameworkName = "Chromium Embedded Framework"

// helperSuffixes are the child-process helper variants Chromium looks for on
// macOS. The empty one is the default helper and the only one that is
// mandatory; the others exist so that a host can give a single process type
// different entitlements later without changing this layout again.
var helperSuffixes = []struct {
	name string // appended to "<App> Helper"
	id   string // appended to the bundle identifier
}{
	{"", ".helper"},
	{" (Alerts)", ".helper.alerts"},
	{" (GPU)", ".helper.gpu"},
	{" (Plugin)", ".helper.plugin"},
	{" (Renderer)", ".helper.renderer"},
}

func main() {
	var (
		binFlag     = flag.String("bin", "", "path to the built executable (required)")
		cefFlag     = flag.String("cef", "Release", "CEF distribution directory holding the framework bundle")
		outFlag     = flag.String("out", "", "output bundle path (default \"<name>.app\" next to -bin)")
		nameFlag    = flag.String("name", "", "application name (default: base name of -bin)")
		idFlag      = flag.String("id", "", "bundle identifier of the application (default \"com.example.<name>\")")
		versionFlag = flag.String("version", "0.0.0", "CFBundleShortVersionString / CFBundleVersion")
		minOSFlag   = flag.String("min-os", "11.0", "LSMinimumSystemVersion")
		signFlag    = flag.String("sign", "-", "codesign identity, \"-\" for ad-hoc, empty to skip signing")
	)
	flag.Parse()

	if err := run(*binFlag, *cefFlag, *outFlag, *nameFlag, *idFlag, *versionFlag, *minOSFlag, *signFlag); err != nil {
		fmt.Fprintln(os.Stderr, "cefbundle:", err)
		os.Exit(1)
	}
}

func run(bin, cef, out, name, id, version, minOS, sign string) error {
	if bin == "" {
		return fmt.Errorf("-bin is required")
	}
	info, err := os.Stat(bin)
	if err != nil {
		return fmt.Errorf("-bin: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("-bin: %s is a directory", bin)
	}

	framework := filepath.Join(cef, frameworkName+".framework")
	if _, err := os.Stat(filepath.Join(framework, frameworkName)); err != nil {
		return fmt.Errorf("no CEF framework in %s - run `go run ./tools/cefdl` first: %w", cef, err)
	}

	if name == "" {
		name = strings.TrimSuffix(filepath.Base(bin), filepath.Ext(bin))
	}
	if id == "" {
		id = "com.example." + sanitizeIdentifier(name)
	}
	if out == "" {
		out = filepath.Join(filepath.Dir(bin), name+".app")
	}

	// The bundle is assembled outside the destination and moved in at the
	// end. Not for atomicity: codesign rejects a bundle carrying
	// com.apple.FinderInfo, and a file provider - iCloud Drive over
	// ~/Documents, for one - puts that attribute back on every .app
	// directory within a second of it being stripped. Signing in place is
	// then a race that cannot be won. A scratch directory is not watched by
	// any provider, and an attribute stamped on the finished bundle
	// afterwards no longer invalidates the signature.
	scratch, err := os.MkdirTemp("", "cefbundle-")
	if err != nil {
		return fmt.Errorf("scratch directory: %w", err)
	}
	defer os.RemoveAll(scratch)
	staged := filepath.Join(scratch, filepath.Base(out))

	contents := filepath.Join(staged, "Contents")
	frameworks := filepath.Join(contents, "Frameworks")
	if err := os.MkdirAll(filepath.Join(contents, "MacOS"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(frameworks, 0o755); err != nil {
		return err
	}

	if err := writePlist(filepath.Join(contents, "Info.plist"), plistData{
		Executable: name,
		Identifier: id,
		Name:       name,
		Version:    version,
		MinOS:      minOS,
	}); err != nil {
		return err
	}
	if err := copyFile(bin, filepath.Join(contents, "MacOS", name), 0o755); err != nil {
		return fmt.Errorf("copying the executable: %w", err)
	}

	fmt.Printf("cefbundle: copying %s\n", framework)
	if err := copyTree(framework, filepath.Join(frameworks, frameworkName+".framework")); err != nil {
		return fmt.Errorf("copying the framework: %w", err)
	}

	// The license notices cefdl puts next to the framework are a
	// redistribution obligation, so they travel with the bundle. A -cef
	// directory from an older cefdl run lacks them - warn, don't fail.
	for _, notice := range []string{"CEF_LICENSE.txt", "CREDITS.html"} {
		source := filepath.Join(cef, notice)
		if _, statErr := os.Stat(source); statErr != nil {
			fmt.Printf("cefbundle: warning: %s not found in %s - re-run cefdl -force to fetch the license notices\n", notice, cef)
			continue
		}
		if err := copyFile(source, filepath.Join(contents, "Resources", notice), 0o644); err != nil {
			return fmt.Errorf("copying %s: %w", notice, err)
		}
	}

	var signTargets []string
	for _, helper := range helperSuffixes {
		helperName := name + " Helper" + helper.name
		helperApp := filepath.Join(frameworks, helperName+".app")
		if err := os.MkdirAll(filepath.Join(helperApp, "Contents", "MacOS"), 0o755); err != nil {
			return err
		}
		if err := writePlist(filepath.Join(helperApp, "Contents", "Info.plist"), plistData{
			Executable: helperName,
			Identifier: id + helper.id,
			Name:       helperName,
			Version:    version,
			MinOS:      minOS,
			// Helpers must not show up in the Dock or take focus - without
			// this every renderer process would appear as its own app.
			Background: true,
		}); err != nil {
			return err
		}
		if err := copyFile(bin, filepath.Join(helperApp, "Contents", "MacOS", helperName), 0o755); err != nil {
			return fmt.Errorf("copying the helper executable: %w", err)
		}
		signTargets = append(signTargets, helperApp)
	}

	// Inner-out, which is what --deep gets wrong: a nested bundle signed
	// after its container invalidates the container's signature.
	signTargets = append([]string{filepath.Join(frameworks, frameworkName+".framework")}, signTargets...)
	signTargets = append(signTargets, staged)
	if err := codesign(sign, staged, signTargets); err != nil {
		return err
	}

	// A bundle is replaced, never patched: a stale helper from a previous run
	// would keep working and quietly serve an old binary.
	if err := os.RemoveAll(out); err != nil {
		return fmt.Errorf("clearing %s: %w", out, err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := moveTree(staged, out); err != nil {
		return fmt.Errorf("moving the bundle to %s: %w", out, err)
	}

	fmt.Printf("cefbundle: %s ready (%d helper bundles)\n", out, len(helperSuffixes))
	return nil
}

// moveTree moves src to dst, falling back to a copy when the two are on
// different file systems - the scratch directory usually shares a volume with
// the destination, but TMPDIR can be pointed anywhere.
func moveTree(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyTree(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

type plistData struct {
	Executable string
	Identifier string
	Name       string
	Version    string
	MinOS      string
	Background bool
}

// infoPlist is deliberately minimal: everything here is either required for
// AppKit to treat the directory as an application, or required for CEF to
// find its helpers. NSPrincipalClass is left out on purpose - gomium installs
// its own CefAppProtocol-aware NSApplication before AppKit would read it (see
// InstallMacApplication).
var infoPlist = template.Must(template.New("plist").Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>{{.Executable}}</string>
	<key>CFBundleIdentifier</key>
	<string>{{.Identifier}}</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>{{.Name}}</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>{{.Version}}</string>
	<key>CFBundleVersion</key>
	<string>{{.Version}}</string>
	<key>LSMinimumSystemVersion</key>
	<string>{{.MinOS}}</string>
{{- if .Background}}
	<key>LSUIElement</key>
	<true/>
{{- end}}
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSSupportsAutomaticGraphicsSwitching</key>
	<true/>
</dict>
</plist>
`))

func writePlist(path string, data plistData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := infoPlist.Execute(f, data); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return f.Close()
}

// copyTree copies a directory recursively, preserving symlinks and the
// executable bit. Symlinks matter: a versioned framework bundle is held
// together by "Versions/Current", and resolving those links would both double
// the size and break the bundle.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case entry.IsDir():
			return os.MkdirAll(target, 0o755)
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode().Perm())
		default:
			// Sockets, devices and the like have no business in a framework;
			// skipping them beats copying something meaningless.
			return nil
		}
	})
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// OpenFile honours the umask, so the executable bit has to be set again.
	return os.Chmod(dst, perm)
}

// codesign signs the bundles with the given identity. An empty identity skips
// signing; so does a missing codesign binary, which keeps the tool usable on
// a machine that has no Xcode command line tools (the bundle it produces is
// still correct, it just cannot run on Apple silicon until it is signed).
func codesign(identity, root string, targets []string) error {
	if identity == "" {
		return nil
	}
	tool, err := exec.LookPath("codesign")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cefbundle: codesign not found, leaving the bundle unsigned")
		return nil
	}
	for _, target := range targets {
		// codesign refuses to sign a bundle carrying extended attributes -
		// "resource fork, Finder information, or similar detritus not
		// allowed" - and macOS keeps producing them: com.apple.provenance on
		// every freshly written file, com.apple.FinderInfo on every bundle
		// codesign itself has just signed, and com.apple.fileprovider.* on
		// anything below an iCloud-synced directory. So the whole tree is
		// stripped again before each signature, not once up front. None of
		// these are ours; the signature lives inside the Mach-O and
		// _CodeSignature, not in an attribute.
		//
		// The file provider re-stamps asynchronously, which can lose the
		// race against a single strip - hence the retry rather than one
		// attempt.
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if err := stripExtendedAttributes(root); err != nil {
				return err
			}
			cmd := exec.Command(tool, "--force", "--sign", identity, "--timestamp=none", target)
			out, err := cmd.CombinedOutput()
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = fmt.Errorf("codesign %s: %w\n%s", target, err, out)
		}
		if lastErr != nil {
			return lastErr
		}
	}
	fmt.Printf("cefbundle: signed %d bundles with identity %q\n", len(targets), identity)
	return nil
}

// stripExtendedAttributes clears the extended attributes below path. Missing
// xattr tooling is not an error: the attributes only matter because codesign
// rejects them, and codesign is optional too.
func stripExtendedAttributes(path string) error {
	tool, err := exec.LookPath("xattr")
	if err != nil {
		return nil
	}
	if out, err := exec.Command(tool, "-c", "-r", path).CombinedOutput(); err != nil {
		return fmt.Errorf("xattr -cr %s: %w\n%s", path, err, out)
	}
	return nil
}

// sanitizeIdentifier reduces a name to what a bundle identifier accepts:
// alphanumerics and hyphens. Anything else becomes a hyphen, so a name with
// spaces still yields a usable default.
func sanitizeIdentifier(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			return r
		default:
			return '-'
		}
	}, name)
}
