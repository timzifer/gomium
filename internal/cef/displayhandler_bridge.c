#include "displayhandler_bridge.h"
#include "_cgo_export.h"

static void CEF_CALLBACK on_title_change_trampoline(struct _cef_display_handler_t* self,
                                                    struct _cef_browser_t* browser,
                                                    const cef_string_t* title) {
    goDisplayHandlerOnTitleChange(self, browser, (cef_string_t*)title);
}

static void CEF_CALLBACK on_address_change_trampoline(struct _cef_display_handler_t* self,
                                                      struct _cef_browser_t* browser,
                                                      struct _cef_frame_t* frame,
                                                      const cef_string_t* url) {
    goDisplayHandlerOnAddressChange(self, browser, frame, (cef_string_t*)url);
}

static int CEF_CALLBACK on_console_message_trampoline(struct _cef_display_handler_t* self,
                                                      struct _cef_browser_t* browser,
                                                      cef_log_severity_t level,
                                                      const cef_string_t* message,
                                                      const cef_string_t* source,
                                                      int line) {
    return goDisplayHandlerOnConsoleMessage(self, browser, (cef_string_t*)message, (cef_string_t*)source, line);
}

static int CEF_CALLBACK on_cursor_change_trampoline(struct _cef_display_handler_t* self,
                                                    struct _cef_browser_t* browser,
                                                    cef_cursor_handle_t cursor,
                                                    cef_cursor_type_t type,
                                                    const cef_cursor_info_t* custom_cursor_info) {
    return goDisplayHandlerOnCursorChange(self, browser, type);
}

static void CEF_CALLBACK on_favicon_urlchange_trampoline(struct _cef_display_handler_t* self,
                                                         struct _cef_browser_t* browser,
                                                         cef_string_list_t icon_urls) {
    goDisplayHandlerOnFaviconUrlChange(self, browser, icon_urls);
}

struct _cef_display_handler_t* cDisplayHandlerCreate() { DEBUG_CALL;
    cef_display_handler_t* handler = (cef_display_handler_t*)calloc(1, sizeof(cef_display_handler_t));
    handler -> base.size = sizeof(cef_display_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler->on_console_message = on_console_message_trampoline;
    handler->on_address_change = on_address_change_trampoline;
    handler->on_title_change = on_title_change_trampoline;
    handler->on_favicon_urlchange = on_favicon_urlchange_trampoline;
    // cursor changes moved from cef_render_handler_t to the display handler
    handler->on_cursor_change = on_cursor_change_trampoline;

    return handler;
}
void cDisplayHandlerFree(struct _cef_display_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}
