package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "contextmenuhandler_bridge.h"
*/
import "C"
import (
	"sync"
	"unsafe"
)

type (
	ContextMenuTypeFlags int
	ContextMenuMediaType int
	ContextMenuEditFlags int
	MenuItemType         int

	// ContextMenuParams is a snapshot of the right-click context. It stays
	// valid after the callback returned - it carries no CEF references.
	ContextMenuParams struct {
		// X, Y are view coordinates of the click.
		X, Y              int
		LinkUrl           string
		UnfilteredLinkUrl string
		// SourceUrl is the media source (image src etc.) if clicked on media.
		SourceUrl     string
		PageUrl       string
		FrameUrl      string
		SelectionText string
		IsEditable    bool
		TypeFlags     ContextMenuTypeFlags
		MediaType     ContextMenuMediaType
		EditFlags     ContextMenuEditFlags
	}

	// ContextMenuItem mirrors one entry of the default Chromium menu.
	ContextMenuItem struct {
		CommandId int
		Label     string
		Type      MenuItemType
		Enabled   bool
		Checked   bool
		SubMenu   []ContextMenuItem
	}

	// ContextMenuCallback finishes a context menu the handler took over.
	// Exactly one of Continue or Cancel must be called; safe from any
	// goroutine. Continue executes the given command as if the user picked
	// it from Chromium's own menu (e.g. MenuIdCopy).
	ContextMenuCallback interface {
		Continue(commandId int)
		Cancel()
	}

	// ContextMenuHandler takes over context menu display. RunContextMenu
	// runs synchronously on the browser process UI thread - capture the
	// callback, return true, show an own (e.g. Fyne) menu and finish via
	// the callback. Return false for CEF's default handling.
	ContextMenuHandler interface {
		RunContextMenu(instance Browser, params ContextMenuParams, model []ContextMenuItem, callback ContextMenuCallback) bool
	}

	contextMenuCallback struct {
		ptr  *C.cef_run_context_menu_callback_t
		once *sync.Once
	}
)

const (
	// TypeFlags - what the click hit (bitmask).
	CmTypeNone      = ContextMenuTypeFlags(C.CM_TYPEFLAG_NONE)
	CmTypePage      = ContextMenuTypeFlags(C.CM_TYPEFLAG_PAGE)
	CmTypeFrame     = ContextMenuTypeFlags(C.CM_TYPEFLAG_FRAME)
	CmTypeLink      = ContextMenuTypeFlags(C.CM_TYPEFLAG_LINK)
	CmTypeMedia     = ContextMenuTypeFlags(C.CM_TYPEFLAG_MEDIA)
	CmTypeSelection = ContextMenuTypeFlags(C.CM_TYPEFLAG_SELECTION)
	CmTypeEditable  = ContextMenuTypeFlags(C.CM_TYPEFLAG_EDITABLE)

	// EditFlags - which edit operations apply (bitmask).
	CmEditCanUndo      = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_UNDO)
	CmEditCanRedo      = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_REDO)
	CmEditCanCut       = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_CUT)
	CmEditCanCopy      = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_COPY)
	CmEditCanPaste     = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_PASTE)
	CmEditCanDelete    = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_DELETE)
	CmEditCanSelectAll = ContextMenuEditFlags(C.CM_EDITFLAG_CAN_SELECT_ALL)

	MenuItemTypeCommand   = MenuItemType(C.MENUITEMTYPE_COMMAND)
	MenuItemTypeCheck     = MenuItemType(C.MENUITEMTYPE_CHECK)
	MenuItemTypeRadio     = MenuItemType(C.MENUITEMTYPE_RADIO)
	MenuItemTypeSeparator = MenuItemType(C.MENUITEMTYPE_SEPARATOR)
	MenuItemTypeSubMenu   = MenuItemType(C.MENUITEMTYPE_SUBMENU)

	// Command ids of Chromium's built-in menu entries, usable with
	// ContextMenuCallback.Continue.
	MenuIdBack       = int(C.MENU_ID_BACK)
	MenuIdForward    = int(C.MENU_ID_FORWARD)
	MenuIdReload     = int(C.MENU_ID_RELOAD)
	MenuIdUndo       = int(C.MENU_ID_UNDO)
	MenuIdRedo       = int(C.MENU_ID_REDO)
	MenuIdCut        = int(C.MENU_ID_CUT)
	MenuIdCopy       = int(C.MENU_ID_COPY)
	MenuIdPaste      = int(C.MENU_ID_PASTE)
	MenuIdDelete     = int(C.MENU_ID_DELETE)
	MenuIdSelectAll  = int(C.MENU_ID_SELECT_ALL)
	MenuIdFind       = int(C.MENU_ID_FIND)
	MenuIdPrint      = int(C.MENU_ID_PRINT)
	MenuIdViewSource = int(C.MENU_ID_VIEW_SOURCE)
)

func (c contextMenuCallback) Continue(commandId int) {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))
		C.cRunContextMenuCallbackContinue(c.ptr, C.int(commandId))
	})
}

func (c contextMenuCallback) Cancel() {
	c.once.Do(func() {
		defer releaseBase(unsafe.Pointer(c.ptr))
		C.cRunContextMenuCallbackCancel(c.ptr)
	})
}

func snapshotContextMenuParams(params *C.struct__cef_context_menu_params_t) ContextMenuParams {
	return ContextMenuParams{
		X:                 int(C.cContextMenuParamsGetX(params)),
		Y:                 int(C.cContextMenuParamsGetY(params)),
		LinkUrl:           cefUserfreeToGoString(C.cContextMenuParamsGetLinkUrl(params)),
		UnfilteredLinkUrl: cefUserfreeToGoString(C.cContextMenuParamsGetUnfilteredLinkUrl(params)),
		SourceUrl:         cefUserfreeToGoString(C.cContextMenuParamsGetSourceUrl(params)),
		PageUrl:           cefUserfreeToGoString(C.cContextMenuParamsGetPageUrl(params)),
		FrameUrl:          cefUserfreeToGoString(C.cContextMenuParamsGetFrameUrl(params)),
		SelectionText:     cefUserfreeToGoString(C.cContextMenuParamsGetSelectionText(params)),
		IsEditable:        C.cContextMenuParamsIsEditable(params) != 0,
		TypeFlags:         ContextMenuTypeFlags(C.cContextMenuParamsGetTypeFlags(params)),
		MediaType:         ContextMenuMediaType(C.cContextMenuParamsGetMediaType(params)),
		EditFlags:         ContextMenuEditFlags(C.cContextMenuParamsGetEditStateFlags(params)),
	}
}

func snapshotMenuModel(model *C.struct__cef_menu_model_t) []ContextMenuItem {
	count := int(C.cMenuModelGetCount(model))
	items := make([]ContextMenuItem, 0, count)
	for i := 0; i < count; i++ {
		index := C.size_t(i)
		item := ContextMenuItem{
			CommandId: int(C.cMenuModelGetCommandIdAt(model, index)),
			Label:     cefUserfreeToGoString(C.cMenuModelGetLabelAt(model, index)),
			Type:      MenuItemType(C.cMenuModelGetTypeAt(model, index)),
			Enabled:   C.cMenuModelIsEnabledAt(model, index) != 0,
			Checked:   C.cMenuModelIsCheckedAt(model, index) != 0,
		}
		if item.Type == MenuItemTypeSubMenu {
			if sub := C.cMenuModelGetSubMenuAt(model, index); sub != nil {
				item.SubMenu = snapshotMenuModel(sub)
				releaseBase(unsafe.Pointer(sub))
			}
		}
		items = append(items, item)
	}
	return items
}

func contextMenuHandler(handler *C.cef_context_menu_handler_t, cefBrowser *C.cef_browser_t) (ContextMenuHandler, Browser) {
	app := getInstance()
	if app == nil {
		return nil, nil
	}

	app.clientMutex.Lock()
	var found *client
	for _, c := range app.clients {
		c.Lock()
		match := c.contextMenuHandlerPtr == handler
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
	return found.contextMenuHandler, found.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetContextMenuHandler
func goGetContextMenuHandler(ptr *C.cef_client_t) *C.cef_context_menu_handler_t {

	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.contextMenuHandlerPtr == nil || c.contextMenuHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.contextMenuHandlerPtr))
	return c.contextMenuHandlerPtr
}

//export goContextMenuHandlerRunContextMenu
func goContextMenuHandlerRunContextMenu(self *C.cef_context_menu_handler_t,
	cefBrowser *C.cef_browser_t,
	frame *C.cef_frame_t,
	params *C.struct__cef_context_menu_params_t,
	model *C.struct__cef_menu_model_t,
	callback *C.cef_run_context_menu_callback_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))
	defer releaseBase(unsafe.Pointer(frame))
	defer releaseBase(unsafe.Pointer(params))
	defer releaseBase(unsafe.Pointer(model))

	wrapped := contextMenuCallback{ptr: callback, once: &sync.Once{}}

	handler, browserInstance := contextMenuHandler(self, cefBrowser)
	if handler == nil {
		wrapped.once.Do(func() { releaseBase(unsafe.Pointer(callback)) })
		return 0
	}

	// params and model are only valid during this call - snapshot both
	paramsCopy := snapshotContextMenuParams(params)
	items := snapshotMenuModel(model)

	if handler.RunContextMenu(browserInstance, paramsCopy, items, wrapped) {
		return 1
	}
	wrapped.once.Do(func() { releaseBase(unsafe.Pointer(callback)) })
	return 0
}
