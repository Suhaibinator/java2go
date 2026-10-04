package stdjava

import (
	"sync"
	"testing"
	"unicode/utf16"
)

func TestUUID101ValuesAndNominal(t *testing.T) {
	a := NewJavaUUID(0x123456789abcdef0, -1)
	b := UUIDFromStringJavaString(JavaStringFromHostUTF8("12345678-9ABC-DEF0-FFFF-FFFFFFFFFFFF"))
	if a.GetMostSignificantBits() != 0x123456789abcdef0 || a.GetLeastSignificantBits() != -1 || !a.Equals(b) || a == b || a.CompareTo(b) != 0 {
		t.Fatal("UUID bits/value/identity contract")
	}
	e := NewExecution()
	if !ObjectEqualsExecution(e, a, b) || ObjectHashCodeExecution(e, a) != a.HashCode() {
		t.Fatal("Object execution dispatch")
	}
	for _, id := range []TypeID{"java.util.UUID", ObjectTypeID, ComparableTypeID, SerializableTypeID} {
		if !ObjectInstanceOf(a, id) {
			t.Fatalf("missing nominal relation %s", id)
		}
	}
	if ObjectInstanceOf((*JavaUUID)(nil), "java.util.UUID") || a.Equals((*JavaUUID)(nil)) || a.Equals(JavaStringFromHostUTF8("foreign")) {
		t.Fatal("nil/foreign equality")
	}
	if NewJavaUUID(-1, 0).CompareTo(NewJavaUUID(1, 0)) != -1 || NewJavaUUID(0, -1).CompareTo(NewJavaUUID(0, 1)) != -1 {
		t.Fatal("compare must be signed")
	}
	if got := uuid101Text(a.StringJava2goExecution(e)); got != "12345678-9abc-def0-ffff-ffffffffffff" {
		t.Fatal(got)
	}
	short := UUIDFromStringJavaString(JavaStringFromHostUTF8("1-1-1-1-1"))
	if got := uuid101Text(short.StringJava2goExecution(e)); got != "00000001-0001-0001-0001-000000000001" {
		t.Fatal(got)
	}
	for _, text := range []string{"+1-+1-+1-+1-+1", "１-１-１-１-１"} {
		if !short.Equals(UUIDFromStringJavaString(JavaStringFromHostUTF8(text))) {
			t.Fatalf("JDK legacy parse domain %s", text)
		}
	}
}
func TestUUID101NameAndImmutable(t *testing.T) {
	bytes := PrimitiveArrayLiteral[int8](PrimitiveTypeID("byte"), 97, 98, 99)
	a := UUIDNameFromBytes(bytes)
	bytes.Elements[0] = 0
	if got := uuid101Text(a.StringJava2goExecution(NewExecution())); got != "90015098-3cd2-3fb0-9696-3f7d28e17f72" {
		t.Fatal(got)
	}
	if got := uuid101Text(UUIDNameFromBytes(PrimitiveArrayLiteral[int8](PrimitiveTypeID("byte"))).StringJava2goExecution(NewExecution())); got != "d41d8cd9-8f00-3204-a980-0998ecf8427e" {
		t.Fatal(got)
	}
}
func TestUUID101ExceptionsAndUTF16(t *testing.T) {
	for _, text := range []string{"", "x", "1-1-1-1-1-1", "g-1-1-1-1", "1--1-1-1", "8000000000000000-1-1-1-1"} {
		func() {
			defer func() {
				if p := recover(); p == nil || !ObjectInstanceOf(p, "java.lang.IllegalArgumentException") {
					t.Errorf("not IAE: %q %v", text, p)
				}
			}()
			UUIDFromStringJavaString(JavaStringFromHostUTF8(text))
		}()
	}
	func() {
		defer func() {
			if p := recover(); p == nil || !ObjectInstanceOf(p, "java.lang.NullPointerException") {
				t.Errorf("null not NPE: %v", p)
			}
		}()
		UUIDFromStringJavaString(nil)
	}()
	func() {
		defer func() {
			if p := recover(); p == nil || !ObjectInstanceOf(p, "java.lang.NullPointerException") {
				t.Errorf("null bytes not NPE: %v", p)
			}
		}()
		UUIDNameFromBytes(nil)
	}()
}
func TestUUID101ConcurrentValues(t *testing.T) {
	a := NewJavaUUID(-1, 17)
	var w sync.WaitGroup
	for i := 0; i < 8; i++ {
		w.Add(1)
		go func() {
			defer w.Done()
			for j := 0; j < 100; j++ {
				b := UUIDFromStringJavaString(a.StringJava2goExecution(nil))
				if !a.Equals(b) || a.HashCode() != b.HashCode() {
					t.Error("immutable race contract")
				}
			}
		}()
	}
	w.Wait()
}

func uuid101Text(s *JavaString) string { return string(utf16.Decode(s.units)) }
