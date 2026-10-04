package stdjava

import "testing"

func TestMapPutAllCovariantValuesAndNull(t *testing.T) {
	source := NewMap[string, string]()
	source.Put("key", "value")
	source.Put("nil", NullString())
	target := NewMap[string, any]()
	target.PutAll(source)
	if target.Get("key") != "value" || !javaReferenceIsNull(target.Get("nil")) {
		t.Fatal("covariant value or null lost")
	}
	source.Put("key", "source")
	if target.Get("key") != "value" {
		t.Fatal("entry storage aliased")
	}
	for _, test := range []struct {
		name string
		call func()
	}{
		{"nil source", func() { var source *Map[string, string]; target.PutAll(source) }},
		{"nil target", func() { var target *Map[string, string]; target.PutAll(source) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if failure := recover(); !CaughtAs(failure, "NullPointerException") {
					t.Fatalf("failure=%v", failure)
				}
			}()
			test.call()
			t.Fatal("expected null failure")
		})
	}
}
