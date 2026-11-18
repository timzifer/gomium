#include "browser_client_bridge.h"
#include "_cgo_export.h"

static struct _cef_life_span_handler_t* CEF_CALLBACK get_life_span_handler_trampoline(struct _cef_client_t* self) {
    return goGetLifeSpanHandler(self);
}

static struct _cef_render_handler_t* CEF_CALLBACK get_render_handler_trampoline(struct _cef_client_t* self) {
    return goGetRenderHandler(self);
}

static struct _cef_request_handler_t* CEF_CALLBACK get_request_handler_trampoline(struct _cef_client_t* self) {
    return goGetRequestHandler(self);
}

static struct _cef_display_handler_t* CEF_CALLBACK get_display_handler_trampoline(struct _cef_client_t* self) {
    return goGetDisplayHandler(self);
}

static struct _cef_load_handler_t* CEF_CALLBACK get_load_handler_trampoline(struct _cef_client_t* self) {
    return goGetLoadHandler(self);
}

static struct _cef_download_handler_t* CEF_CALLBACK get_download_handler_trampoline(struct _cef_client_t* self) {
    return goGetDownloadHandler(self);
}

static struct _cef_jsdialog_handler_t* CEF_CALLBACK get_jsdialog_handler_trampoline(struct _cef_client_t* self) {
    return goGetJsDialogHandler(self);
}

static struct _cef_context_menu_handler_t* CEF_CALLBACK get_context_menu_handler_trampoline(struct _cef_client_t* self) {
    return goGetContextMenuHandler(self);
}

static struct _cef_find_handler_t* CEF_CALLBACK get_find_handler_trampoline(struct _cef_client_t* self) {
    return goGetFindHandler(self);
}

static struct _cef_drag_handler_t* CEF_CALLBACK get_drag_handler_trampoline(struct _cef_client_t* self) {
    return goGetDragHandler(self);
}

static struct _cef_focus_handler_t* CEF_CALLBACK get_focus_handler_trampoline(struct _cef_client_t* self) {
    return goGetFocusHandler(self);
}

static struct _cef_keyboard_handler_t* CEF_CALLBACK get_keyboard_handler_trampoline(struct _cef_client_t* self) {
    return goGetKeyboardHandler(self);
}

static struct _cef_dialog_handler_t* CEF_CALLBACK get_dialog_handler_trampoline(struct _cef_client_t* self) {
    return goGetFileDialogHandler(self);
}

static struct _cef_frame_handler_t* CEF_CALLBACK get_frame_handler_trampoline(struct _cef_client_t* self) {
    return goGetFrameHandler(self);
}

static int CEF_CALLBACK on_process_message_received_trampoline(struct _cef_client_t* self, struct _cef_browser_t* browser, struct _cef_frame_t* frame, cef_process_id_t source_process, struct _cef_process_message_t* message) {
    return goOnProcessMessageReceived(self, browser, frame, source_process, message);
}

struct _cef_client_t* cClientCreate() { DEBUG_CALL;
    cef_client_t* client = (cef_client_t*)calloc(1, sizeof(cef_client_t));
    client -> base.size = sizeof(cef_client_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)client);

	client -> get_life_span_handler = get_life_span_handler_trampoline;
	client -> get_render_handler = get_render_handler_trampoline;
	client -> get_request_handler = get_request_handler_trampoline;
	client -> get_display_handler = get_display_handler_trampoline;
	client -> get_load_handler = get_load_handler_trampoline;
	client -> get_download_handler = get_download_handler_trampoline;
	client -> get_jsdialog_handler = get_jsdialog_handler_trampoline;
	client -> get_context_menu_handler = get_context_menu_handler_trampoline;
	client -> get_find_handler = get_find_handler_trampoline;
	client -> get_drag_handler = get_drag_handler_trampoline;
	client -> get_focus_handler = get_focus_handler_trampoline;
	client -> get_keyboard_handler = get_keyboard_handler_trampoline;
	client -> get_dialog_handler = get_dialog_handler_trampoline;
	client -> get_frame_handler = get_frame_handler_trampoline;
	client -> on_process_message_received = on_process_message_received_trampoline;

    return client;
}

void cClientFree(struct _cef_client_t* client) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)client)->release((cef_base_ref_counted_t*)client);
}

int cBridgeGetBrowserIdentifier(struct _cef_browser_t *browser) {
    return browser -> get_identifier(browser);
}

void cBridgeBrowserStopLoad(struct _cef_browser_t *browser) {
    browser -> stop_load(browser);
}

cef_string_utf8_t * cBridgeCefSourceToString(cef_string_t * source) {
      cef_string_utf8_t * output = cef_string_userfree_utf8_alloc();
      if (source == 0) {
          return output;
      }
      cef_string_to_utf8(source->str, source->length, output);
      return output;
}
