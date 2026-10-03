package stdjava

import (
	"reflect"
	"sync"
	"testing"
)

func TestJavaStringCaseReferenceEnglishUTF16AndIdentity(t *testing.T) {
	cases := []struct {
		in, upper, lower     []uint16
		upperSame, lowerSame bool
	}{
		{[]uint16{}, []uint16{}, []uint16{}, true, true},
		{[]uint16{'s', 't', 'r', 'a', 0xdf, 'e'}, []uint16{'S', 'T', 'R', 'A', 'S', 'S', 'E'}, []uint16{'s', 't', 'r', 'a', 0xdf, 'e'}, false, true},
		{[]uint16{0x39f, 0x3a3}, []uint16{0x39f, 0x3a3}, []uint16{0x3bf, 0x3c2}, true, false},
		{[]uint16{0x130}, []uint16{0x130}, []uint16{'i', 0x307}, false, false},
		{[]uint16{'a', 0xd800, 'B', 0xdc00, 'c'}, []uint16{'A', 0xd800, 'B', 0xdc00, 'C'}, []uint16{'a', 0xd800, 'b', 0xdc00, 'c'}, false, false},
		{[]uint16{0xd801, 0xdc00, 0xd801, 0xdc28}, []uint16{0xd801, 0xdc00, 0xd801, 0xdc00}, []uint16{0xd801, 0xdc28, 0xd801, 0xdc28}, false, false},
	}
	for _, c := range cases {
		value := NewJavaStringUTF16(c.in)
		upper, lower := JavaStringToUpperCase(value, LocaleENGLISH), JavaStringToLowerCase(value, LocaleENGLISH)
		if !reflect.DeepEqual(upper.UTF16Copy(), c.upper) || !reflect.DeepEqual(lower.UTF16Copy(), c.lower) {
			t.Fatalf("input%x upper%x want%x lower%x want%x", c.in, upper.UTF16Copy(), c.upper, lower.UTF16Copy(), c.lower)
		}
		if (upper == value) != c.upperSame || (lower == value) != c.lowerSame {
			t.Fatalf("case result identity differs from the captured JDK: %x", c.in)
		}
		if !reflect.DeepEqual(value.UTF16Copy(), c.in) {
			t.Fatal("casing mutated source payload")
		}
	}
}

func TestJavaStringCaseReferenceDefaultLocale(t *testing.T) {
	old := LocaleGetDefault()
	defer LocaleSetDefault(old)
	value := NewJavaStringUTF16([]uint16{'i', 'I', 0x131, 0x130})
	LocaleSetDefault(LocaleForLanguageTag("tr"))
	if got := JavaStringToUpperCase(value).UTF16Copy(); !reflect.DeepEqual(got, []uint16{0x130, 'I', 'I', 0x130}) {
		t.Fatalf("Turkish upper %x", got)
	}
	if got := JavaStringToLowerCase(value).UTF16Copy(); !reflect.DeepEqual(got, []uint16{'i', 0x131, 0x131, 'i'}) {
		t.Fatalf("Turkish lower %x", got)
	}
	LocaleSetDefault(LocaleENGLISH)
	if got := JavaStringToLowerCase(value).UTF16Copy(); !reflect.DeepEqual(got, []uint16{'i', 'i', 0x131, 'i', 0x307}) {
		t.Fatalf("updated default lower %x", got)
	}
}

func TestJavaStringCaseReferenceNullAndNativeBoundary(t *testing.T) {
	for name, action := range map[string]func(){"upper receiver": func() { _ = JavaStringToUpperCase(nil, LocaleENGLISH) }, "lower receiver": func() { _ = JavaStringToLowerCase(nil, LocaleENGLISH) }, "upper locale": func() { _ = JavaStringToUpperCase(NewJavaStringUTF16([]uint16{'x'}), nil) }, "lower locale": func() { _ = JavaStringToLowerCase(NewJavaStringUTF16([]uint16{'x'}), nil) }} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil || !CaughtAsType(r, "java.lang.NullPointerException") {
					t.Fatalf("expected NPE got %T", r)
				}
			}()
			action()
			t.Fatal("null casing returned")
		})
	}
	if got := StringToUpperCaseLocale("straße", LocaleENGLISH); got != "STRASSE" {
		t.Fatalf("native API changed: %q", got)
	}
}

func TestJavaStringCaseReferenceConcurrentMapping(t *testing.T) {
	value := NewJavaStringUTF16([]uint16{0x39f, 0x3a3})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 30; n++ {
				if got := JavaStringToLowerCase(value, LocaleENGLISH).UTF16Copy(); !reflect.DeepEqual(got, []uint16{0x3bf, 0x3c2}) {
					t.Errorf("context/state leaked: %x", got)
				}
			}
		}()
	}
	wg.Wait()
}
