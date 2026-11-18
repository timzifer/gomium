#ifndef DOWNLOAD_HANDLER_BRIDGE_H
#define DOWNLOAD_HANDLER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_download_handler_capi.h"
#include "include/capi/cef_download_item_capi.h"

struct _cef_download_handler_t* cDownloadHandlerCreate();
void cDownloadHandlerFree(struct _cef_download_handler_t*);

// cef_download_item_t accessors - the function pointers cannot be called from Go
int cDownloadItemIsInProgress(struct _cef_download_item_t* item);
int cDownloadItemIsComplete(struct _cef_download_item_t* item);
int cDownloadItemIsCanceled(struct _cef_download_item_t* item);
int cDownloadItemIsInterrupted(struct _cef_download_item_t* item);
int cDownloadItemGetInterruptReason(struct _cef_download_item_t* item);
int64_t cDownloadItemGetCurrentSpeed(struct _cef_download_item_t* item);
int cDownloadItemGetPercentComplete(struct _cef_download_item_t* item);
int64_t cDownloadItemGetTotalBytes(struct _cef_download_item_t* item);
int64_t cDownloadItemGetReceivedBytes(struct _cef_download_item_t* item);
cef_string_userfree_t cDownloadItemGetFullPath(struct _cef_download_item_t* item);
uint32_t cDownloadItemGetId(struct _cef_download_item_t* item);
cef_string_userfree_t cDownloadItemGetUrl(struct _cef_download_item_t* item);
cef_string_userfree_t cDownloadItemGetOriginalUrl(struct _cef_download_item_t* item);
cef_string_userfree_t cDownloadItemGetSuggestedFileName(struct _cef_download_item_t* item);
cef_string_userfree_t cDownloadItemGetContentDisposition(struct _cef_download_item_t* item);
cef_string_userfree_t cDownloadItemGetMimeType(struct _cef_download_item_t* item);

void cDownloadBeforeCallbackContinue(struct _cef_before_download_callback_t* callback,
                                     cef_string_t* download_path,
                                     int show_dialog);
void cDownloadItemCallbackCancel(struct _cef_download_item_callback_t* callback);
void cDownloadItemCallbackPause(struct _cef_download_item_callback_t* callback);
void cDownloadItemCallbackResume(struct _cef_download_item_callback_t* callback);

#endif
