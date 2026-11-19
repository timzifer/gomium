#include "app_bridge.h"
#include "_cgo_export.h"
#include <string.h>
#include <stdio.h>

static struct _cef_render_process_handler_t* CEF_CALLBACK get_render_process_handler_trampoline(struct _cef_app_t* self) {
    return goGetRenderProcessHandler(self);
}

static struct _cef_browser_process_handler_t* CEF_CALLBACK get_browser_process_handler_trampoline(struct _cef_app_t* self) {
    return goGetBrowserProcessHandler(self);
}

// Custom schemes must be announced identically in every process - browser,
// renderer, gpu, utility. CEF calls this once per process before anything
// else runs, which is why the registration cannot be an ordinary API call
// after startup: the renderer would not know the scheme and refuse to parse
// its URLs.
static void CEF_CALLBACK on_register_custom_schemes_trampoline(struct _cef_app_t* self,
                                                               struct _cef_scheme_registrar_t* registrar) {
    goOnRegisterCustomSchemes(self, registrar);
}

static int g_ignore_certificate_errors = 0;

void cAppSetIgnoreCertificateErrors(int ignore) {
    g_ignore_certificate_errors = ignore;
}

const char* cAppExpectedApiHash() {
    return CEF_API_HASH_PLATFORM;
}

const char* cAppConfigureApiVersion() {
    // configures the client API version on first call and returns the hash
    // the library computed for it. Must not be called before
    // cAppRuntimeVersionMatches confirmed the library version: older
    // libcef versions export cef_api_hash with a different signature
    // (no version parameter) and return NULL for the arguments we pass.
    return cef_api_hash(CEF_API_VERSION, 0);
}

const char* cAppCompiledCefVersion() {
    return CEF_VERSION;
}

// cef_version_info is the only libcef call that is safe against a
// mismatched library: it has kept the signature int(int) since CEF 3 and
// out-of-range entries return 0. Ancient versions assign the entries
// differently (0=CEF major, 1=commit, 2..5=chromium), so the individual
// numbers may be shuffled for very old DLLs - a full 7-entry comparison
// still detects every mismatch either way.
int cAppRuntimeVersionMatches() {
    return cef_version_info(0) == CEF_VERSION_MAJOR &&
           cef_version_info(1) == CEF_VERSION_MINOR &&
           cef_version_info(2) == CEF_VERSION_PATCH &&
           cef_version_info(4) == CHROME_VERSION_MAJOR &&
           cef_version_info(5) == CHROME_VERSION_MINOR &&
           cef_version_info(6) == CHROME_VERSION_BUILD &&
           cef_version_info(7) == CHROME_VERSION_PATCH;
}

void cAppRuntimeCefVersion(char* buf, size_t len) {
    snprintf(buf, len, "%d.%d.%d (chromium %d.%d.%d.%d)",
             cef_version_info(0), cef_version_info(1), cef_version_info(2),
             cef_version_info(4), cef_version_info(5), cef_version_info(6),
             cef_version_info(7));
}

static void append_switch(struct _cef_command_line_t* command_line, const char* name) {
    cef_string_t switch_name = {0};
    cef_string_utf8_to_utf16(name, strlen(name), &switch_name);
    command_line->append_switch(command_line, &switch_name);
    cef_string_clear(&switch_name);
}

// merge_disable_feature adds a feature to the --disable-features switch,
// preserving any features that are already listed there. A plain
// append_switch_with_value would create a duplicate entry and only one of
// them wins, silently dropping the other list.
static void merge_disable_feature(struct _cef_command_line_t* command_line, const char* feature) {
    const char* switch_name_utf8 = "disable-features";
    cef_string_t switch_name = {0};
    cef_string_utf8_to_utf16(switch_name_utf8, strlen(switch_name_utf8), &switch_name);

    cef_string_utf8_t existing_utf8 = {0};
    if (command_line->has_switch(command_line, &switch_name)) {
        cef_string_userfree_t existing = command_line->get_switch_value(command_line, &switch_name);
        if (existing) {
            cef_string_to_utf8(existing->str, existing->length, &existing_utf8);
            cef_string_userfree_free(existing);
        }
    }

    size_t merged_len = (existing_utf8.str ? existing_utf8.length : 0) + strlen(feature) + 2;
    char* merged = (char*)calloc(1, merged_len);
    if (existing_utf8.str && existing_utf8.length > 0) {
        snprintf(merged, merged_len, "%s,%s", existing_utf8.str, feature);
    } else {
        snprintf(merged, merged_len, "%s", feature);
    }

    cef_string_t merged_value = {0};
    cef_string_utf8_to_utf16(merged, strlen(merged), &merged_value);
    command_line->append_switch_with_value(command_line, &switch_name, &merged_value);

    cef_string_clear(&merged_value);
    cef_string_utf8_clear(&existing_utf8);
    free(merged);
    cef_string_clear(&switch_name);
}

static void CEF_CALLBACK on_before_command_line_processing_trampoline(struct _cef_app_t* self,
                                                                     const cef_string_t* process_type,
                                                                     struct _cef_command_line_t* command_line) {
    // Chromium's stack sampling profilers unwind thread stacks with
    // RtlVirtualUnwind. Unwinding across Go frames crashes inside ntdll, so
    // the profiler must stay off in every process.
    append_switch(command_line, "disable-stack-profiler");
    merge_disable_feature(command_line, "ThreadGroupProfiler");
    merge_disable_feature(command_line, "SamplingProfilerReporting");

    // The GPU watchdog considers the GPU main thread hung because it never
    // drains a Windows message queue under the default message pump used in
    // this embedding, and kills the GPU process with RESULT_CODE_HUNG (2)
    // every ~15s. The browser gives up after a few kills ("GPU process isn't
    // usable"). Genuine GPU hangs go undetected with the watchdog off, which
    // is acceptable for an embedded webview.
    append_switch(command_line, "disable-gpu-watchdog");

    // Chrome's "Read Anything" feature registers a page-load-metrics observer
    // whose OnSoftNavigation handler calls tabs::TabInterface::GetFromContents.
    // Alloy-style (OSR) browsers have no tab model attached to their
    // WebContents, so the lookup dereferences null and the browser process
    // dies with an access violation on the first soft navigation (SPA-style
    // history.pushState after a user interaction). Feature is browser-UI only
    // and meaningless in an embedded webview - keep it off.
    merge_disable_feature(command_line, "ImmersiveReadAnything");

    // Without this switch Chrome's own login prompt answers HTTP auth
    // challenges of Alloy-style browsers, and the client's GetAuthCredentials
    // is never called (CEF 8078 libcef/browser/chrome/
    // chrome_content_browser_client_cef.cc:CreateLoginDelegate). With it, the
    // challenge goes to the client's request handler (login_delegate.cc).
    append_switch(command_line, "disable-chrome-login-prompt");

    if (g_ignore_certificate_errors) {
        append_switch(command_line, "ignore-certificate-errors");
    }

#if defined(__linux__)
    // The zygote is this Go executable, and the Go runtime always runs
    // several threads. Chromium's zygote refuses to fork a multi-threaded
    // process, so every child it should spawn (GPU, renderer) fails with
    // "GPU process launch failed: error_code=1002" and the browser dies with
    // "GPU process isn't usable". Without a zygote the browser process
    // launches children by fork+exec itself. Requires --no-sandbox, which
    // no_sandbox = 1 already implies.
    append_switch(command_line, "no-zygote");
#endif
    // Do NOT release command_line here: it wraps the process-global command
    // line. Releasing it corrupts the browser process and crashes the GPU
    // children (verified empirically against CEF 151).
}

struct _cef_app_t* cAppCreate() { DEBUG_CALL;
    cef_app_t* app = (cef_app_t*)calloc(1, sizeof(cef_app_t));
    app->base.size = sizeof(cef_app_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)app);

    app->get_render_process_handler = get_render_process_handler_trampoline;
    app->get_browser_process_handler = get_browser_process_handler_trampoline;
    app->on_before_command_line_processing = on_before_command_line_processing_trampoline;
    app->on_register_custom_schemes = on_register_custom_schemes_trampoline;

    return app;
}
void cAppFree(struct _cef_app_t* app) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)app)->release((cef_base_ref_counted_t*)app);
}

#ifdef _WIN32
void cAppSetMainArgsInstance(cef_main_args_t* args, uintptr_t instance) {
    args->instance = (HINSTANCE)instance;
}
#endif
