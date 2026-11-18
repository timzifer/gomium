package cef

import "testing"

func TestNewKeyboardEvent(t *testing.T) {
	ev := newKeyboardEvent(KeyEventTypeKeyDown, KeyEventModifierControl|KeyEventModifierShift,
		0x46, 33, 'f', 'F', false, true)

	if ev.Type != KeyEventTypeKeyDown {
		t.Errorf("Type = %v, want KeyDown", ev.Type)
	}
	if ev.Modifiers != KeyEventModifierControl|KeyEventModifierShift {
		t.Errorf("Modifiers = %b", ev.Modifiers)
	}
	if ev.WindowsKeyCode != 0x46 || ev.NativeKeyCode != 33 {
		t.Errorf("key codes = %d/%d", ev.WindowsKeyCode, ev.NativeKeyCode)
	}
	if ev.Character != 'f' || ev.UnmodifiedCharacter != 'F' {
		t.Errorf("characters = %q/%q", ev.Character, ev.UnmodifiedCharacter)
	}
	if ev.IsSystemKey || !ev.FocusOnEditableField {
		t.Errorf("flags = system:%v editable:%v", ev.IsSystemKey, ev.FocusOnEditableField)
	}
}

func TestClientWhereWithoutInstance(t *testing.T) {
	// no app instance registered in unit tests: lookup must return nil, not panic
	if c := clientWhere(func(*client) bool { return true }); c != nil {
		t.Fatalf("clientWhere returned %v without instance", c)
	}
}
