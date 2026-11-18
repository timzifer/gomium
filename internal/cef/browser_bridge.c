#include "browser_bridge.h"
#include "_cgo_export.h"


void cBridgeBrowserLoadUrl(struct _cef_browser_t *browser, const cef_string_t* url) {
    struct _cef_frame_t* main_frame = browser -> get_main_frame(browser);
    if (main_frame == NULL) {
        return;
    }
    main_frame -> load_url(main_frame, url);
    ((cef_base_ref_counted_t*)main_frame)->release((cef_base_ref_counted_t*)main_frame);
}

void cBridgeBrowserReload(struct _cef_browser_t *browser) {
    browser -> reload(browser);
}

void cBridgeBrowserReloadIgnoreCache(struct _cef_browser_t *browser) {
    browser -> reload_ignore_cache(browser);
}

void cBridgeBrowserSendFocusEvent(struct _cef_browser_t *browser, int setFocus) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    // send_focus_event was removed - set_focus is the 151 equivalent
    browserHost -> set_focus(browserHost, setFocus);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserWasResized(struct _cef_browser_t *browser) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> was_resized(browserHost);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserCloseBrowser(struct _cef_browser_t *browser, int force) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> close_browser(browserHost, force);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserSendMouseClickEvent(struct _cef_browser_t *browser, int x, int y, int type, int mouseUp, int clickCount, uint32_t modifiers) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    cef_mouse_event_t event = {0};
    event.x = x;
    event.y = y;
    event.modifiers = modifiers;

    browserHost -> send_mouse_click_event(browserHost, &event, (cef_mouse_button_type_t) type, mouseUp, clickCount);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserSendMouseMoveEvent(struct _cef_browser_t *browser, int x, int y, int mouseLeave, uint32_t modifiers) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    cef_mouse_event_t event = {0};
    event.x = x;
    event.y = y;
    event.modifiers = modifiers;

    browserHost -> send_mouse_move_event(browserHost, &event, mouseLeave);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserSendMouseWheelEvent(struct _cef_browser_t *browser, int x, int y, int dx, int dy, uint32_t modifiers) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    cef_mouse_event_t event = {0};
    event.x = x;
    event.y = y;
    event.modifiers = modifiers;

    browserHost -> send_mouse_wheel_event(browserHost, &event, dx, dy);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserSendKeyEvent(struct _cef_browser_t *browser, cef_key_event_type_t eventType, int nativeKeyCode, int windowsKeyCode, char16_t character, int isSystemKey, uint32_t modifiers) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    cef_key_event_t event = {0};

    // cef_key_event_t hat seit CEF 151 ein fuehrendes size-Feld. Bleibt es 0,
    // verwirft CEF das Event stillschweigend - Tastatur tot, Maus weiter ok,
    // denn cef_mouse_event_t hat kein size-Feld.
    event.size = sizeof(cef_key_event_t);

    event.native_key_code = nativeKeyCode;
    event.windows_key_code = windowsKeyCode;
    event.type = eventType;
    event.character = character;
    event.unmodified_character = character;
    event.is_system_key = isSystemKey;
    event.modifiers = modifiers;

    browserHost -> send_key_event(browserHost, &event);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserExecuteJavascript(struct _cef_browser_t *browser, const cef_string_t* code, const cef_string_t* script_url, int start_line) {
    struct _cef_frame_t* main_frame = browser -> get_main_frame(browser);
    if (main_frame == NULL) {
        return;
    }
    main_frame -> execute_java_script(main_frame, code, script_url, start_line);
    ((cef_base_ref_counted_t*)main_frame)->release((cef_base_ref_counted_t*)main_frame);
}

int cBridgeBrowserCanGoBack(struct _cef_browser_t *browser) {
    return browser -> can_go_back(browser);
}

int cBridgeBrowserCanGoForward(struct _cef_browser_t *browser) {
    return browser -> can_go_forward(browser);
}

void cBridgeBrowserGoBack(struct _cef_browser_t *browser) {
    browser -> go_back(browser);
}

void cBridgeBrowserGoForward(struct _cef_browser_t *browser) {
    browser -> go_forward(browser);
}

void cBridgeBrowserSetZoomLevel(struct _cef_browser_t *browser, double level) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> set_zoom_level(browserHost, level);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

double cBridgeBrowserGetZoomLevel(struct _cef_browser_t *browser) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    double level = browserHost -> get_zoom_level(browserHost);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
    return level;
}

void cBridgeBrowserFind(struct _cef_browser_t *browser, const cef_string_t* text, int forward, int matchCase, int findNext) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> find(browserHost, text, forward, matchCase, findNext);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserStopFinding(struct _cef_browser_t *browser, int clearSelection) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> stop_finding(browserHost, clearSelection);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

void cBridgeBrowserSetAudioMuted(struct _cef_browser_t *browser, int muted) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> set_audio_muted(browserHost, muted);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

int cBridgeBrowserIsAudioMuted(struct _cef_browser_t *browser) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    int muted = browserHost -> is_audio_muted(browserHost);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
    return muted;
}

static void CEF_CALLBACK on_pdf_print_finished_trampoline(struct _cef_pdf_print_callback_t* self,
                                                          const cef_string_t* path,
                                                          int ok) {
    goPdfPrintFinished((void*)self, ok);
}

struct _cef_pdf_print_callback_t* cPdfPrintCallbackCreate() { DEBUG_CALL;
    cef_pdf_print_callback_t* callback = (cef_pdf_print_callback_t*)calloc(1, sizeof(cef_pdf_print_callback_t));
    callback -> base.size = sizeof(cef_pdf_print_callback_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)callback);
    callback -> on_pdf_print_finished = on_pdf_print_finished_trampoline;
    return callback;
}

void cBridgeBrowserPrintToPdf(struct _cef_browser_t *browser,
                              const cef_string_t* path,
                              int landscape,
                              int printBackground,
                              struct _cef_pdf_print_callback_t* callback) {
    cef_pdf_print_settings_t settings = {0};
    settings.size = sizeof(cef_pdf_print_settings_t);
    settings.landscape = landscape;
    settings.print_background = printBackground;

    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    browserHost -> print_to_pdf(browserHost, path, &settings, callback);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
}

struct _cef_dictionary_value_t* cBridgeDictionaryCreate() { DEBUG_CALL;
    return cef_dictionary_value_create();
}

void cBridgeDictionarySetString(struct _cef_dictionary_value_t *dictionary, const cef_string_t* key, const cef_string_t* value) {
    dictionary -> set_string(dictionary, key, value);
}

void cBridgeDictionarySetBool(struct _cef_dictionary_value_t *dictionary, const cef_string_t* key, int value) {
    dictionary -> set_bool(dictionary, key, value);
}

void cBridgeDictionarySetInt(struct _cef_dictionary_value_t *dictionary, const cef_string_t* key, int value) {
    dictionary -> set_int(dictionary, key, value);
}

void cBridgeDictionaryRelease(struct _cef_dictionary_value_t *dictionary) {
    ((cef_base_ref_counted_t*)dictionary)->release((cef_base_ref_counted_t*)dictionary);
}

// cBridgeBrowserExecuteDevToolsMethod must run on the browser process UI
// thread - execute_dev_tools_method answers with 0 anywhere else instead of
// doing the work.
//
// The params reference is NOT released here: the CEF wrapper takes it over
// (the same handover as print_to_pdf and its callback). Releasing it a second
// time would free a dictionary CEF still holds.
int cBridgeBrowserExecuteDevToolsMethod(struct _cef_browser_t *browser,
                                        const cef_string_t* method,
                                        struct _cef_dictionary_value_t *params) {
    struct _cef_browser_host_t *browserHost = browser -> get_host(browser);
    int messageId = browserHost -> execute_dev_tools_method(browserHost, 0, method, params);
    ((cef_base_ref_counted_t*)browserHost)->release((cef_base_ref_counted_t*)browserHost);
    return messageId;
}
