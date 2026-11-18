package cef

/*
   #include <stdlib.h>
   #include <string.h>
   #include "include/internal/cef_types.h"
*/
import "C"
import (
	"runtime"
	"unsafe"
)

type (
	appSettings struct {
		CachePath        *string
		RootCachePath    *string
		LogSeverity      LogSeverity
		LogFile          *string
		ResourcesDirPath *string
		LocalesDirPath   *string

		// BrowserSubprocessPath names a separate executable for CEF's helper
		// processes. Empty means the main executable on Windows and Linux;
		// on macOS it means CEF derives the helper from the app bundle (see
		// tools/cefbundle). Must be absolute when set.
		BrowserSubprocessPath *string
		// FrameworkDirPath and MainBundlePath are macOS-only. Empty means
		// the framework and the bundle are found at their conventional place
		// inside the top-level .app - which is what tools/cefbundle builds.
		// Both must be absolute when set; CEF ignores them elsewhere.
		FrameworkDirPath *string
		MainBundlePath   *string

		JavaScriptFlags          *string
		RemoteDebuggingPort      *int
		IgnoreCertificateErrors  bool
		MultiThreadedMessageLoop bool

		///
		// Value that will be returned as the User-Agent HTTP header. If empty the
		// default User-Agent string will be used. Also configurable using the
		// "user-agent" command-line switch.
		///
		UserAgent *string
	}

	LogSeverity int
)

const (
	LOGSEVERITY_DEFAULT = LogSeverity(C.LOGSEVERITY_DEFAULT)
	LOGSEVERITY_VERBOSE = LogSeverity(C.LOGSEVERITY_VERBOSE)
	LOGSEVERITY_INFO    = LogSeverity(C.LOGSEVERITY_INFO)
	LOGSEVERITY_WARNING = LogSeverity(C.LOGSEVERITY_WARNING)
	LOGSEVERITY_ERROR   = LogSeverity(C.LOGSEVERITY_ERROR)
	LOGSEVERITY_DISABLE = LogSeverity(C.LOGSEVERITY_DISABLE)
)

func (s appSettings) apply(settings *C.cef_settings_t) {
	settings.size = C.sizeof_cef_settings_t

	settings.no_sandbox = C.int(1)
	settings.log_severity = (C.cef_log_severity_t)(C.int(s.LogSeverity))

	if s.RemoteDebuggingPort != nil {
		settings.remote_debugging_port = C.int(*s.RemoteDebuggingPort)
	}

	// ignore_certificate_errors is no longer a cef_settings_t field - it is
	// applied as a command-line switch in on_before_command_line_processing.
	if s.MultiThreadedMessageLoop && runtime.GOOS == "windows" {
		settings.multi_threaded_message_loop = C.int(1)
	}
	if s.CachePath != nil {
		cefStringInto(&settings.cache_path, *s.CachePath)
	}
	// every cache_path - including per-request-context ones - must live
	// inside root_cache_path. Without an explicit root the single cache
	// path doubles as the root.
	if s.RootCachePath != nil {
		cefStringInto(&settings.root_cache_path, *s.RootCachePath)
	} else if s.CachePath != nil {
		cefStringInto(&settings.root_cache_path, *s.CachePath)
	}
	if s.LogFile != nil {
		cefStringInto(&settings.log_file, *s.LogFile)
	}
	if s.ResourcesDirPath != nil {
		cefStringInto(&settings.resources_dir_path, *s.ResourcesDirPath)
	}
	if s.LocalesDirPath != nil {
		cefStringInto(&settings.locales_dir_path, *s.LocalesDirPath)
	}
	if s.BrowserSubprocessPath != nil {
		cefStringInto(&settings.browser_subprocess_path, *s.BrowserSubprocessPath)
	}
	if s.FrameworkDirPath != nil {
		cefStringInto(&settings.framework_dir_path, *s.FrameworkDirPath)
	}
	if s.MainBundlePath != nil {
		cefStringInto(&settings.main_bundle_path, *s.MainBundlePath)
	}
	if s.JavaScriptFlags != nil {
		cefStringInto(&settings.javascript_flags, *s.JavaScriptFlags)
	}
	if s.UserAgent != nil {
		cefStringInto(&settings.user_agent, *s.UserAgent)
	}

}

// clear releases the string buffers allocated by apply. CEF copies the
// settings during cef_initialize, so this is safe to call right after.
func (s appSettings) clear(settings *C.cef_settings_t) {
	C.cef_string_clear(&settings.cache_path)
	C.cef_string_clear(&settings.root_cache_path)
	C.cef_string_clear(&settings.log_file)
	C.cef_string_clear(&settings.resources_dir_path)
	C.cef_string_clear(&settings.locales_dir_path)
	C.cef_string_clear(&settings.browser_subprocess_path)
	C.cef_string_clear(&settings.framework_dir_path)
	C.cef_string_clear(&settings.main_bundle_path)
	C.cef_string_clear(&settings.javascript_flags)
	C.cef_string_clear(&settings.user_agent)
}

// cefString allocates a cef_string_t holding the given text. The result must
// be released with cefStringFree.
func cefString(original string) (final *C.cef_string_t) {
	final = (*C.cef_string_t)(C.calloc(1, C.sizeof_cef_string_t))
	cefStringInto(final, original)
	return final
}

// cefStringInto fills an existing (zeroed or previously cleared)
// cef_string_t in place.
func cefStringInto(dst *C.cef_string_t, s string) {
	charString := C.CString(s)
	defer C.free(unsafe.Pointer(charString))
	C.cef_string_from_utf8(charString, C.strlen(charString), dst)
}

// cefStringFree releases both the UTF-16 buffer and the container allocated
// by cefString.
func cefStringFree(s *C.cef_string_t) {
	if s == nil {
		return
	}
	C.cef_string_clear(s)
	C.free(unsafe.Pointer(s))
}

// cefUserfreeToGoString converts a cef_string_userfree_t (an owned string
// returned by CEF) to a Go string and frees it.
func cefUserfreeToGoString(source C.cef_string_userfree_t) string {
	if source == nil {
		return ""
	}
	defer C.cef_string_userfree_free(source)
	return cefToGoString(source)
}
