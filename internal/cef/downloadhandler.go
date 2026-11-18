package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "downloadhandler_bridge.h"
*/
import "C"
import "unsafe"

type (
	// DownloadItem is a snapshot of a download's state. It stays valid after
	// the callback returned - it carries no CEF references.
	DownloadItem struct {
		Id                 uint32
		Url                string
		OriginalUrl        string
		SuggestedFileName  string
		ContentDisposition string
		MimeType           string
		FullPath           string
		TotalBytes         int64
		ReceivedBytes      int64
		CurrentSpeed       int64
		PercentComplete    int
		InProgress         bool
		Complete           bool
		Canceled           bool
		Interrupted        bool
		InterruptReason    int
	}

	// DownloadControl cancels, pauses or resumes a running download. Only
	// valid during the OnDownloadUpdated call it was handed to.
	DownloadControl interface {
		Cancel()
		Pause()
		Resume()
	}

	// DownloadHandler receives download events. CanDownload runs synchronously
	// on the browser process UI thread and must return quickly; the other
	// callbacks are dispatched on their own goroutines.
	DownloadHandler interface {
		// CanDownload decides whether a user-initiated download may start.
		CanDownload(instance Browser, url string, requestMethod string) bool
		// OnBeforeDownload determines the target path for a download. Return
		// the full file path including the file name, or an empty string to
		// download to the suggested name in the default temp directory.
		OnBeforeDownload(instance Browser, item DownloadItem) (downloadPath string)
		// OnDownloadUpdated fires on every status or progress change, before
		// and after OnBeforeDownload.
		OnDownloadUpdated(instance Browser, item DownloadItem, control DownloadControl)
	}

	downloadControl struct {
		ptr *C.cef_download_item_callback_t
	}
)

func (d downloadControl) Cancel() { C.cDownloadItemCallbackCancel(d.ptr) }
func (d downloadControl) Pause()  { C.cDownloadItemCallbackPause(d.ptr) }
func (d downloadControl) Resume() { C.cDownloadItemCallbackResume(d.ptr) }

// snapshotDownloadItem copies all download-item state into Go memory. Must be
// called synchronously inside the CEF callback - the item is only valid there.
func snapshotDownloadItem(item *C.struct__cef_download_item_t) DownloadItem {
	return DownloadItem{
		Id:                 uint32(C.cDownloadItemGetId(item)),
		Url:                cefUserfreeToGoString(C.cDownloadItemGetUrl(item)),
		OriginalUrl:        cefUserfreeToGoString(C.cDownloadItemGetOriginalUrl(item)),
		SuggestedFileName:  cefUserfreeToGoString(C.cDownloadItemGetSuggestedFileName(item)),
		ContentDisposition: cefUserfreeToGoString(C.cDownloadItemGetContentDisposition(item)),
		MimeType:           cefUserfreeToGoString(C.cDownloadItemGetMimeType(item)),
		FullPath:           cefUserfreeToGoString(C.cDownloadItemGetFullPath(item)),
		TotalBytes:         int64(C.cDownloadItemGetTotalBytes(item)),
		ReceivedBytes:      int64(C.cDownloadItemGetReceivedBytes(item)),
		CurrentSpeed:       int64(C.cDownloadItemGetCurrentSpeed(item)),
		PercentComplete:    int(C.cDownloadItemGetPercentComplete(item)),
		InProgress:         C.cDownloadItemIsInProgress(item) != 0,
		Complete:           C.cDownloadItemIsComplete(item) != 0,
		Canceled:           C.cDownloadItemIsCanceled(item) != 0,
		Interrupted:        C.cDownloadItemIsInterrupted(item) != 0,
		InterruptReason:    int(C.cDownloadItemGetInterruptReason(item)),
	}
}

func downloadHandler(handler *C.cef_download_handler_t, cefBrowser *C.cef_browser_t) (DownloadHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.downloadHandlerPtr == handler
		c.Unlock()
		if match {
			found = c
			break
		}
	}
	app.clientMutex.Unlock()

	if found == nil {
		return nil, nil
	}
	// the browser map is guarded by the client mutex, not the clientMutex
	return found.downloadHandler, found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetDownloadHandler
func goGetDownloadHandler(ptr *C.cef_client_t) *C.cef_download_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.downloadHandlerPtr == nil || c.downloadHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.downloadHandlerPtr))
	return c.downloadHandlerPtr
}

//export goDownloadHandlerCanDownload
func goDownloadHandlerCanDownload(self *C.cef_download_handler_t,
	cefBrowser *C.cef_browser_t,
	url *C.cef_string_t,
	requestMethod *C.cef_string_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	handler, browserInstance := downloadHandler(self, cefBrowser)
	if handler == nil {
		return 1
	}
	if handler.CanDownload(browserInstance, cefToGoString(url), cefToGoString(requestMethod)) {
		return 1
	}
	return 0
}

//export goDownloadHandlerOnBeforeDownload
func goDownloadHandlerOnBeforeDownload(self *C.cef_download_handler_t,
	cefBrowser *C.cef_browser_t,
	downloadItem *C.struct__cef_download_item_t,
	suggestedName *C.cef_string_t,
	callback *C.cef_before_download_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(downloadItem))

	handler, browserInstance := downloadHandler(self, cefBrowser)
	if handler == nil {
		// no handler - default handling (cancel with the Alloy runtime style)
		releaseBase(unsafe.Pointer(callback))
		return 0
	}

	item := snapshotDownloadItem(downloadItem)
	if item.SuggestedFileName == "" {
		item.SuggestedFileName = cefToGoString(suggestedName)
	}

	// the callback reference is kept across the goroutine boundary and
	// released after continuation
	go func() {
		defer releaseBase(unsafe.Pointer(callback))

		path := handler.OnBeforeDownload(browserInstance, item)
		var cPath *C.cef_string_t
		if path != "" {
			cPath = cefString(path)
			defer cefStringFree(cPath)
		}
		C.cDownloadBeforeCallbackContinue(callback, cPath, 0)
	}()
	return 1
}

//export goDownloadHandlerOnDownloadUpdated
func goDownloadHandlerOnDownloadUpdated(self *C.cef_download_handler_t,
	cefBrowser *C.cef_browser_t,
	downloadItem *C.struct__cef_download_item_t,
	callback *C.cef_download_item_callback_t) {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(downloadItem))

	handler, browserInstance := downloadHandler(self, cefBrowser)
	if handler == nil {
		releaseBase(unsafe.Pointer(callback))
		return
	}

	item := snapshotDownloadItem(downloadItem)

	// the callback reference is kept for the duration of the dispatched call
	go func() {
		defer releaseBase(unsafe.Pointer(callback))
		handler.OnDownloadUpdated(browserInstance, item, downloadControl{callback})
	}()
}
