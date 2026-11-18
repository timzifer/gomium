#include "downloadhandler_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK can_download_trampoline(struct _cef_download_handler_t* self,
                                                struct _cef_browser_t* browser,
                                                const cef_string_t* url,
                                                const cef_string_t* request_method) {
    return goDownloadHandlerCanDownload(self, browser, (cef_string_t*)url, (cef_string_t*)request_method);
}

static int CEF_CALLBACK on_before_download_trampoline(struct _cef_download_handler_t* self,
                                                      struct _cef_browser_t* browser,
                                                      struct _cef_download_item_t* download_item,
                                                      const cef_string_t* suggested_name,
                                                      struct _cef_before_download_callback_t* callback) {
    return goDownloadHandlerOnBeforeDownload(self, browser, download_item, (cef_string_t*)suggested_name, callback);
}

static void CEF_CALLBACK on_download_updated_trampoline(struct _cef_download_handler_t* self,
                                                        struct _cef_browser_t* browser,
                                                        struct _cef_download_item_t* download_item,
                                                        struct _cef_download_item_callback_t* callback) {
    goDownloadHandlerOnDownloadUpdated(self, browser, download_item, callback);
}

struct _cef_download_handler_t* cDownloadHandlerCreate() { DEBUG_CALL;
    cef_download_handler_t* handler = (cef_download_handler_t*)calloc(1, sizeof(cef_download_handler_t));
    handler -> base.size = sizeof(cef_download_handler_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)handler);

    handler -> can_download = can_download_trampoline;
    handler -> on_before_download = on_before_download_trampoline;
    handler -> on_download_updated = on_download_updated_trampoline;

    return handler;
}

void cDownloadHandlerFree(struct _cef_download_handler_t* handler) { DEBUG_CALL;
    ((cef_base_ref_counted_t*)handler)->release((cef_base_ref_counted_t*)handler);
}

int cDownloadItemIsInProgress(struct _cef_download_item_t* item) { return item->is_in_progress(item); }
int cDownloadItemIsComplete(struct _cef_download_item_t* item) { return item->is_complete(item); }
int cDownloadItemIsCanceled(struct _cef_download_item_t* item) { return item->is_canceled(item); }
int cDownloadItemIsInterrupted(struct _cef_download_item_t* item) { return item->is_interrupted(item); }
int cDownloadItemGetInterruptReason(struct _cef_download_item_t* item) { return (int)item->get_interrupt_reason(item); }
int64_t cDownloadItemGetCurrentSpeed(struct _cef_download_item_t* item) { return item->get_current_speed(item); }
int cDownloadItemGetPercentComplete(struct _cef_download_item_t* item) { return item->get_percent_complete(item); }
int64_t cDownloadItemGetTotalBytes(struct _cef_download_item_t* item) { return item->get_total_bytes(item); }
int64_t cDownloadItemGetReceivedBytes(struct _cef_download_item_t* item) { return item->get_received_bytes(item); }
cef_string_userfree_t cDownloadItemGetFullPath(struct _cef_download_item_t* item) { return item->get_full_path(item); }
uint32_t cDownloadItemGetId(struct _cef_download_item_t* item) { return item->get_id(item); }
cef_string_userfree_t cDownloadItemGetUrl(struct _cef_download_item_t* item) { return item->get_url(item); }
cef_string_userfree_t cDownloadItemGetOriginalUrl(struct _cef_download_item_t* item) { return item->get_original_url(item); }
cef_string_userfree_t cDownloadItemGetSuggestedFileName(struct _cef_download_item_t* item) { return item->get_suggested_file_name(item); }
cef_string_userfree_t cDownloadItemGetContentDisposition(struct _cef_download_item_t* item) { return item->get_content_disposition(item); }
cef_string_userfree_t cDownloadItemGetMimeType(struct _cef_download_item_t* item) { return item->get_mime_type(item); }

void cDownloadBeforeCallbackContinue(struct _cef_before_download_callback_t* callback,
                                     cef_string_t* download_path,
                                     int show_dialog) {
    callback->cont(callback, download_path, show_dialog);
}

void cDownloadItemCallbackCancel(struct _cef_download_item_callback_t* callback) { callback->cancel(callback); }
void cDownloadItemCallbackPause(struct _cef_download_item_callback_t* callback) { callback->pause(callback); }
void cDownloadItemCallbackResume(struct _cef_download_item_callback_t* callback) { callback->resume(callback); }
