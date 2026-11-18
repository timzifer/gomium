package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "filedialoghandler_bridge.h"
*/
import "C"
import (
	"sync"
	"unsafe"
)

type (
	// FileDialogMode is the base mode of a file dialog (cef_file_dialog_mode_t).
	FileDialogMode int

	// FileDialogCallback finishes a dialog the handler took over. Exactly one
	// of Continue or Cancel must be called. Safe from any goroutine.
	FileDialogCallback interface {
		Continue(filePaths []string)
		Cancel()
	}

	// FileDialogHandler receives <input type=file>, showSaveDialog-style and
	// folder requests. Runs on the browser process UI thread - return fast.
	//
	// Escape hatch: return handled=true to take the dialog over (e.g. show a
	// Fyne file picker) and call callback when the user decided. Return
	// handled=false to let CEF show its native dialog.
	FileDialogHandler interface {
		OnFileDialog(instance Browser, mode FileDialogMode, title, defaultFilePath string, acceptFilters []string, callback FileDialogCallback) (handled bool)
	}

	fileDialogCallback struct {
		ptr  *C.cef_file_dialog_callback_t
		once *sync.Once
	}
)

const (
	FileDialogOpen         = FileDialogMode(C.FILE_DIALOG_OPEN)
	FileDialogOpenMultiple = FileDialogMode(C.FILE_DIALOG_OPEN_MULTIPLE)
	FileDialogOpenFolder   = FileDialogMode(C.FILE_DIALOG_OPEN_FOLDER)
	FileDialogSave         = FileDialogMode(C.FILE_DIALOG_SAVE)
)

func (c fileDialogCallback) Continue(filePaths []string) {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))

		list := C.cef_string_list_alloc()
		defer C.cef_string_list_free(list)
		for _, p := range filePaths {
			s := cefString(p)
			C.cef_string_list_append(list, s)
			cefStringFree(s)
		}
		C.cBridgeFileDialogContinue(c.ptr, list)
	})
}

func (c fileDialogCallback) Cancel() {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))
		C.cBridgeFileDialogCancel(c.ptr)
	})
}

// discard drops the reference without touching the callback - used when the
// handler did not take the dialog over.
func (c fileDialogCallback) discard() {
	c.once.Do(func() {
		releaseBase(unsafe.Pointer(c.ptr))
	})
}

func cefStringListToGo(list C.cef_string_list_t) []string {
	if list == nil {
		return nil
	}
	n := C.cef_string_list_size(list)
	out := make([]string, 0, int(n))
	for i := C.size_t(0); i < n; i++ {
		var str C.cef_string_t
		if C.cef_string_list_value(list, i, &str) == 0 {
			continue
		}
		out = append(out, cefToGoString(&str))
		C.cef_string_clear(&str)
	}
	return out
}

func fileDialogHandlerFor(self *C.cef_dialog_handler_t, cefBrowser *C.cef_browser_t) (FileDialogHandler, Browser) {
	c := clientWhere(func(c *client) bool { return c.fileDialogHandlerPtr == self })
	if c == nil || c.fileDialogHandler == nil {
		return nil, nil
	}
	return c.fileDialogHandler, c.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetFileDialogHandler
func goGetFileDialogHandler(ptr *C.cef_client_t) *C.cef_dialog_handler_t {
	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.fileDialogHandlerPtr == nil || c.fileDialogHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.fileDialogHandlerPtr))
	return c.fileDialogHandlerPtr
}

//export goFileDialogHandlerOnFileDialog
func goFileDialogHandlerOnFileDialog(self *C.cef_dialog_handler_t, cefBrowser *C.cef_browser_t,
	mode C.cef_file_dialog_mode_t, title *C.cef_string_t, defaultFilePath *C.cef_string_t,
	acceptFilters C.cef_string_list_t, callback *C.cef_file_dialog_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	h, b := fileDialogHandlerFor(self, cefBrowser)
	if h == nil {
		releaseBase(unsafe.Pointer(callback))
		return 0
	}

	cb := fileDialogCallback{ptr: callback, once: &sync.Once{}}
	handled := h.OnFileDialog(b, FileDialogMode(mode), cefToGoString(title), cefToGoString(defaultFilePath),
		cefStringListToGo(acceptFilters), cb)
	if !handled {
		cb.discard()
		return 0
	}
	return 1
}
