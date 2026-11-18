package cef

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
)

// roundTrip sends values the way the renderer and the browser process
// exchange them: gob over a process message.
func roundTrip(t *testing.T, in []*Exported) []*Exported {
	t.Helper()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(in); err != nil {
		t.Fatal(err)
	}
	var out []*Exported
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

// gob drops zero values even behind a pointer. "", 0 and false must still
// arrive as themselves, not as null.
func TestZeroScalarsSurviveTheProcessMessage(t *testing.T) {
	values := []any{"", 0, uint(0), 0.0, false, "x", 7, uint(7), 1.5, true}
	in := make([]*Exported, len(values))
	for i, value := range values {
		exported, err := exportedFromNative(value, 10)
		if err != nil {
			t.Fatal(err)
		}
		in[i] = exported
	}

	out := roundTrip(t, in)
	for i, want := range values {
		if got := out[i].Native(); !reflect.DeepEqual(got, want) {
			t.Errorf("%#v arrived as %#v", want, got)
		}
	}
}

func TestZeroScalarsSurviveInsideArraysAndObjects(t *testing.T) {
	in, err := exportedFromNative(map[string]any{"pieces": 0, "names": []string{"", "a"}}, 10)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, []*Exported{in})[0].Native()
	want := map[string]any{"pieces": 0, "names": []any{"", "a"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestNullArgumentBecomesTheZeroValueOfTheParameter(t *testing.T) {
	fixed := reflect.TypeOf(func(string, bool) {})
	variadic := reflect.TypeOf(func(int, ...string) {})
	null := &Exported{Nil: true}

	cases := []struct {
		name string
		fn   reflect.Type
		i    int
		want any
	}{
		{"string", fixed, 0, ""},
		{"bool", fixed, 1, false},
		{"fixed before variadic", variadic, 0, 0},
		{"variadic element", variadic, 1, ""},
		{"later variadic element", variadic, 3, ""},
	}
	for _, c := range cases {
		got := callArgument(c.fn, c.i, null)
		if !got.IsValid() || !reflect.DeepEqual(got.Interface(), c.want) {
			t.Errorf("%s: got %v, want %#v", c.name, got, c.want)
		}
	}
	if got := callArgument(fixed, 0, &Exported{Kind: exportedString, String: ptr("x")}); got.Interface() != "x" {
		t.Errorf("a real argument was replaced: %v", got)
	}
}

// The whole path for mde.openDrawing(" A ", "", "A", "B"): every argument
// reaches the variadic function, the empty one included.
func TestEmptyStringReachesAVariadicFunction(t *testing.T) {
	var got []string
	fn := reflect.ValueOf(func(numbers ...string) { got = numbers })

	in := []*Exported{}
	for _, s := range []string{" A ", "", "A", "B"} {
		exported, _ := exportedFromNative(s, 10)
		in = append(in, exported)
	}
	var args []reflect.Value
	for i, arg := range roundTrip(t, in) {
		args = append(args, callArgument(fn.Type(), i, arg))
	}
	fn.Call(args)

	if want := []string{" A ", "", "A", "B"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
