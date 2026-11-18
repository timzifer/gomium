package cef

/*
#include <string.h>
#include "browser_client_bridge.h"
#include "keyboardhandler_bridge.h"
*/
import "C"
import "unsafe"

type (
	// KeyboardEvent is the Go view of cef_key_event_t as CEF delivers it to
	// the keyboard handler.
	KeyboardEvent struct {
		Type                 KeyEventType
		Modifiers            KeyEventModifier
		WindowsKeyCode       int
		NativeKeyCode        int
		Character            rune
		UnmodifiedCharacter  rune
		IsSystemKey          bool
		FocusOnEditableField bool
	}

	// KeyboardHandler sees every key before (OnPreKeyEvent) and after
	// (OnKeyEvent) the page does. Callbacks run on the browser process UI
	// thread.
	//
	// Use it for host shortcuts (return handled=true to swallow the key) and
	// to decide which shortcuts the page may see: OnPreKeyEvent can report
	// isShortcut=true to let CEF treat the key as a browser shortcut.
	KeyboardHandler interface {
		OnPreKeyEvent(instance Browser, event KeyboardEvent) (handled, isShortcut bool)
		OnKeyEvent(instance Browser, event KeyboardEvent) (handled bool)
	}
)

// newKeyboardEvent builds the Go event from plain values. Kept free of cgo so
// the conversion is unit-testable.
func newKeyboardEvent(typ KeyEventType, modifiers KeyEventModifier, windowsKeyCode, nativeKeyCode int,
	character, unmodifiedCharacter rune, isSystemKey, focusOnEditableField bool) KeyboardEvent {
	return KeyboardEvent{
		Type:                 typ,
		Modifiers:            modifiers,
		WindowsKeyCode:       windowsKeyCode,
		NativeKeyCode:        nativeKeyCode,
		Character:            character,
		UnmodifiedCharacter:  unmodifiedCharacter,
		IsSystemKey:          isSystemKey,
		FocusOnEditableField: focusOnEditableField,
	}
}

func keyboardEventFromC(event *C.cef_key_event_t) KeyboardEvent {
	return newKeyboardEvent(
		KeyEventType(event._type),
		KeyEventModifier(event.modifiers),
		int(event.windows_key_code),
		int(event.native_key_code),
		rune(event.character),
		rune(event.unmodified_character),
		event.is_system_key != 0,
		event.focus_on_editable_field != 0,
	)
}

func keyboardHandlerFor(self *C.cef_keyboard_handler_t, cefBrowser *C.cef_browser_t) (KeyboardHandler, Browser) {
	c := clientWhere(func(c *client) bool { return c.keyboardHandlerPtr == self })
	if c == nil || c.keyboardHandler == nil {
		return nil, nil
	}
	return c.keyboardHandler, c.browserByIdentifier(int(C.cBridgeGetBrowserIdentifier(cefBrowser)))
}

//export goGetKeyboardHandler
func goGetKeyboardHandler(ptr *C.cef_client_t) *C.cef_keyboard_handler_t {
	c := getClient(ptr)
	if c == nil {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.keyboardHandlerPtr == nil || c.keyboardHandler == nil {
		return nil
	}
	addRefBase(unsafe.Pointer(c.keyboardHandlerPtr))
	return c.keyboardHandlerPtr
}

//export goKeyboardHandlerOnPreKeyEvent
func goKeyboardHandlerOnPreKeyEvent(self *C.cef_keyboard_handler_t, cefBrowser *C.cef_browser_t, event *C.cef_key_event_t, isShortcut *C.int) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	h, b := keyboardHandlerFor(self, cefBrowser)
	if h == nil {
		return 0
	}
	handled, shortcut := h.OnPreKeyEvent(b, keyboardEventFromC(event))
	if isShortcut != nil && shortcut {
		*isShortcut = 1
	}
	if handled {
		return 1
	}
	return 0
}

//export goKeyboardHandlerOnKeyEvent
func goKeyboardHandlerOnKeyEvent(self *C.cef_keyboard_handler_t, cefBrowser *C.cef_browser_t, event *C.cef_key_event_t) C.int {
	defer releaseBase(unsafe.Pointer(cefBrowser))

	h, b := keyboardHandlerFor(self, cefBrowser)
	if h == nil {
		return 0
	}
	if h.OnKeyEvent(b, keyboardEventFromC(event)) {
		return 1
	}
	return 0
}
