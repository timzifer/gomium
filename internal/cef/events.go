package cef

type (
	Event interface {
		isEvent()
	}

	resizedEvent struct{}

	focusEvent struct {
		gained bool
	}

	mouseEvent struct {
		x, y int
	}

	mouseMovedEvent struct {
		mouseEvent
		mouseLeftBoundary bool
	}

	mouseClickEvent struct {
		mouseEvent
		button    MouseButton
		direction MouseButtonDirection
	}

	scrollEvent struct {
		mouseEvent
		deltaX, deltaY int
	}

	keyEvent struct {
		eventType      KeyEventType
		nativeKeyCode  int
		windowsKeyCode int
		character      rune
		isSystemKey    bool
		modifiers      KeyEventModifier
	}

	MouseButton          int
	MouseButtonDirection int

	KeyEventType      int
	KeyEventModifier  uint32
	KeyEventModifiers []KeyEventModifier
)

// Values mirror cef_event_flags_t (include/internal/cef_types.h).
const (
	KeyEventModifierNone              KeyEventModifier = 0
	KeyEventModifierCapsLock          KeyEventModifier = 1 << 0
	KeyEventModifierShift             KeyEventModifier = 1 << 1
	KeyEventModifierControl           KeyEventModifier = 1 << 2
	KeyEventModifierAlt               KeyEventModifier = 1 << 3
	KeyEventModifierLeftMouseButton   KeyEventModifier = 1 << 4
	KeyEventModifierMiddleMouseButton KeyEventModifier = 1 << 5
	KeyEventModifierRightMouseButton  KeyEventModifier = 1 << 6
	KeyEventModifierCommand           KeyEventModifier = 1 << 7
	KeyEventModifierNumLockOn         KeyEventModifier = 1 << 8
	KeyEventModifierIsKeyPad          KeyEventModifier = 1 << 9
	KeyEventModifierIsLeft            KeyEventModifier = 1 << 10
	KeyEventModifierIsRight           KeyEventModifier = 1 << 11
)

const (
	KeyEventTypeRawKeyDown = KeyEventType(iota)
	KeyEventTypeKeyDown
	KeyEventTypeKeyUp
	KeyEventTypeChar
)
const (
	MouseButtonLeft = MouseButton(iota)
	MouseButtonMiddle
	MouseButtonRight
)
const (
	MouseButtonDown = MouseButtonDirection(iota)
	MouseButtonUp
)

func (f focusEvent) isEvent()      {}
func (s scrollEvent) isEvent()     {}
func (m mouseMovedEvent) isEvent() {}
func (k keyEvent) isEvent()        {}
func (m mouseClickEvent) isEvent() {}
func (r resizedEvent) isEvent()    {}

func NewResizedEvent() Event {
	return resizedEvent{}
}

func NewFocusEvent(gained bool) Event {
	return focusEvent{gained: gained}
}

func (m KeyEventModifiers) Sum() (result KeyEventModifier) {
	for _, modifier := range m {
		result |= modifier
	}
	return
}

func NewKeyTypedEvent(r rune, modifiers ...KeyEventModifier) Event {
	return keyEvent{
		eventType:      KeyEventTypeChar,
		nativeKeyCode:  int(r),
		windowsKeyCode: int(r),
		character:      r,
		isSystemKey:    false,
		modifiers:      KeyEventModifiers(modifiers).Sum(),
	}
}

func NewKeyDownEvent(nativeKeyCode, windowsKeyCode int, isSystemKey bool, modifiers ...KeyEventModifier) Event {
	return keyEvent{
		eventType:      KeyEventTypeKeyDown,
		nativeKeyCode:  nativeKeyCode,
		windowsKeyCode: windowsKeyCode,
		isSystemKey:    isSystemKey,
		modifiers:      KeyEventModifiers(modifiers).Sum(),
	}
}

func NewKeyUpEvent(nativeKeyCode, windowsKeyCode int, isSystemKey bool, modifiers ...KeyEventModifier) Event {
	return keyEvent{
		eventType:      KeyEventTypeKeyUp,
		nativeKeyCode:  nativeKeyCode,
		windowsKeyCode: windowsKeyCode,
		isSystemKey:    isSystemKey,
		modifiers:      KeyEventModifiers(modifiers).Sum(),
	}
}

func firstBoolOrFalse(w []bool) bool {
	if len(w) == 0 {
		return false
	}
	return w[0]
}

func NewMouseClickEvent(x, y int, button MouseButton, direction MouseButtonDirection) Event {
	return mouseClickEvent{
		mouseEvent{
			x: x,
			y: y,
		},
		button,
		direction,
	}
}

func NewMouseMovedEvent(x, y int, boundaryLeft ...bool) Event {
	return mouseMovedEvent{
		mouseEvent: mouseEvent{
			x: x,
			y: y,
		},
		mouseLeftBoundary: firstBoolOrFalse(boundaryLeft),
	}
}

func NewScrollEvent(x, y int, deltaX, deltaY int) Event {
	return scrollEvent{
		mouseEvent: mouseEvent{
			x: x,
			y: y,
		},
		deltaX: deltaX,
		deltaY: deltaY,
	}
}
