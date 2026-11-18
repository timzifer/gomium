#include "draghandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK on_drag_enter_trampoline(struct _cef_drag_handler_t* self,
                                                 struct _cef_browser_t* browser,
                                                 struct _cef_drag_data_t* dragData,
                                                 cef_drag_operations_mask_t mask) {
    return goDragHandlerOnDragEnter(self, browser, dragData, (int)mask);
}

struct _cef_drag_handler_t* cDragHandlerCreate() { DEBUG_CALL;
    cef_drag_handler_t* handler = (cef_drag_handler_t*)calloc(1, sizeof(cef_drag_handler_t));
    handler -> base.size = sizeof(cef_drag_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> on_drag_enter = on_drag_enter_trampoline;

    return handler;
}

void cDragHandlerFree(struct _cef_drag_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

int cBridgeDragDataIsFile(struct _cef_drag_data_t* data) {
    return data -> is_file(data);
}

int cBridgeDragDataFileNames(struct _cef_drag_data_t* data, cef_string_list_t names) {
    return data -> get_file_names(data, names);
}
