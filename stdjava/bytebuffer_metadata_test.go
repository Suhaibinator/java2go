package stdjava

import (
	"encoding/json"
	"sync"
	"testing"
)

// Observed with pinned JDK21; raw calibration stdout SHA256 a71efca82931af6f244ca29c41bf25280839a9baeba6282c00368fea8a0d92c4.
const byteBufferJDKNominalExpectedJSON = `{"messages":{"native-to-source":"class java.nio.HeapByteBuffer cannot be cast to class ByteBuffer (java.nio.HeapByteBuffer is in module java.base of loader 'bootstrap'; ByteBuffer is in unnamed module of loader 'app')","source-to-native":"class ByteBuffer cannot be cast to class java.nio.ByteBuffer (ByteBuffer is in unnamed module of loader 'app'; java.nio.ByteBuffer is in module java.base of loader 'bootstrap')"},"rows":{"allocate":{"capacity":8,"direct":false,"interfaces":"","limit":8,"loader":"bootstrap","modifiers":0,"module":"java.base","position":0,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"allocate-zero":{"capacity":0,"direct":false,"interfaces":"","limit":0,"loader":"bootstrap","modifiers":0,"module":"java.base","position":0,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"duplicate":{"capacity":8,"direct":false,"interfaces":"","limit":7,"loader":"bootstrap","modifiers":0,"module":"java.base","position":1,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"fluent":{"capacity":8,"direct":false,"interfaces":"","limit":7,"loader":"bootstrap","modifiers":0,"module":"java.base","position":1,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"readonly":{"capacity":8,"direct":false,"interfaces":"","limit":7,"loader":"bootstrap","modifiers":16,"module":"java.base","position":1,"readonly":true,"super":"java.nio.HeapByteBuffer","type":"java.nio.HeapByteBufferR"},"readonly-again":{"capacity":8,"direct":false,"interfaces":"","limit":7,"loader":"bootstrap","modifiers":16,"module":"java.base","position":1,"readonly":true,"super":"java.nio.HeapByteBuffer","type":"java.nio.HeapByteBufferR"},"readonly-duplicate":{"capacity":8,"direct":false,"interfaces":"","limit":7,"loader":"bootstrap","modifiers":16,"module":"java.base","position":1,"readonly":true,"super":"java.nio.HeapByteBuffer","type":"java.nio.HeapByteBufferR"},"readonly-slice":{"capacity":6,"direct":false,"interfaces":"","limit":6,"loader":"bootstrap","modifiers":16,"module":"java.base","position":0,"readonly":true,"super":"java.nio.HeapByteBuffer","type":"java.nio.HeapByteBufferR"},"slice":{"capacity":6,"direct":false,"interfaces":"","limit":6,"loader":"bootstrap","modifiers":0,"module":"java.base","position":0,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"slice-range":{"capacity":3,"direct":false,"interfaces":"","limit":3,"loader":"bootstrap","modifiers":0,"module":"java.base","position":0,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"wrap":{"capacity":8,"direct":false,"interfaces":"","limit":8,"loader":"bootstrap","modifiers":0,"module":"java.base","position":0,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"},"wrap-range":{"capacity":8,"direct":false,"interfaces":"","limit":7,"loader":"bootstrap","modifiers":0,"module":"java.base","position":1,"readonly":false,"super":"java.nio.ByteBuffer","type":"java.nio.HeapByteBuffer"}}}`

type byteBufferJDKNominalRow struct {
	Type      string `json:"type"`
	Super     string `json:"super"`
	Modifiers int32  `json:"modifiers"`
	Module    string `json:"module"`
	Loader    string `json:"loader"`
	ReadOnly  bool   `json:"readonly"`
	Capacity  int32  `json:"capacity"`
	Position  int32  `json:"position"`
	Limit     int32  `json:"limit"`
}

func byteBufferNominalFactories() map[string]func() *ByteBuffer {
	array := func() *PrimitiveArray[int8] { return NewPrimitiveArray[int8](8, PrimitiveByteTypeID) }
	heap := func() *ByteBuffer { return ByteBufferAllocate(8).SetPosition(1).SetLimit(7) }
	return map[string]func() *ByteBuffer{
		"allocate": func() *ByteBuffer { return ByteBufferAllocate(8) }, "allocate-zero": func() *ByteBuffer { return ByteBufferAllocate(0) },
		"wrap": func() *ByteBuffer { return ByteBufferWrap(array()) }, "wrap-range": func() *ByteBuffer { return ByteBufferWrap(array(), 1, 6) }, "fluent": heap,
		"slice": func() *ByteBuffer { return heap().Slice() }, "slice-range": func() *ByteBuffer { return heap().Slice(2, 3) }, "duplicate": func() *ByteBuffer { return heap().Duplicate() },
		"readonly": func() *ByteBuffer { return heap().AsReadOnlyBuffer() }, "readonly-again": func() *ByteBuffer { return heap().AsReadOnlyBuffer().AsReadOnlyBuffer() },
		"readonly-slice": func() *ByteBuffer { return heap().AsReadOnlyBuffer().Slice() }, "readonly-duplicate": func() *ByteBuffer { return heap().AsReadOnlyBuffer().Duplicate() },
	}
}
func byteBufferNominalRun(t *testing.T, label string, action func()) {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("%s unexpected panic: %v", label, p)
		}
	}()
	action()
}
func TestByteBufferNativeNominalAllocationJDK(t *testing.T) {
	raw := []byte(byteBufferJDKNominalExpectedJSON)
	var observed struct {
		Rows map[string]byteBufferJDKNominalRow `json:"rows"`
	}
	if err := json.Unmarshal(raw, &observed); err != nil {
		t.Fatal(err)
	}
	for label, factory := range byteBufferNominalFactories() {
		t.Run(label, func(t *testing.T) {
			b := factory()
			want, exists := observed.Rows[label]
			if !exists {
				t.Fatal("actual JDK row absent")
			}
			actual, known := ObjectDynamicType(b)
			t.Logf("dynamic=%q known=%v expectedJDK=%q", actual, known, want.Type)
			if !known || actual != TypeID(want.Type) {
				t.Errorf("native allocation class %q != actualJDK %q", actual, want.Type)
			}
			if b.Capacity() != want.Capacity || b.Position() != want.Position || b.Limit() != want.Limit || b.IsReadOnly() != want.ReadOnly {
				t.Error("fixture/state differs from actualJDK")
			}
			c := ClassLiteral(TypeID(want.Type))
			parent := c.GetSuperclass()
			if parent == nil || parent.TypeID() != TypeID(want.Super) {
				t.Errorf("actualJDK superclass %q missing/wrong", want.Super)
			}
			if c.GetModifiers() != want.Modifiers {
				t.Errorf("modifiers=%d actualJDK=%d", c.GetModifiers(), want.Modifiers)
			}
			loc := ClassRuntimeLocationOf(TypeID(want.Type))
			if !loc.Known || loc.ModuleName != want.Module || loc.LoaderIdentity != BootstrapClassLoaderIdentity {
				t.Errorf("actualJDK runtime location absent/wrong: %+v", loc)
			}
			if !JavaTypeAssignable(actual, JavaNIOByteBufferType) || !JavaTypeAssignable(actual, JavaNIOBufferType) || !JavaTypeAssignable(actual, ComparableTypeID) {
				t.Error("canonical Java ancestry not assignable")
			}
			byteBufferNominalRun(t, "checkedByteBuffer", func() {
				if ReferenceCheckCast[*ByteBuffer](b, JavaNIOByteBufferType) != b {
					t.Error("cast changed receiver identity")
				}
			})
			byteBufferNominalRun(t, "BufferView", func() {
				if ObjectView[*ByteBuffer](b, JavaNIOBufferType) != b {
					t.Error("Buffer view changed identity")
				}
			})
			byteBufferNominalRun(t, "arrayStore", func() {
				a := ReferenceArrayLiteralOf[*ByteBuffer](JavaNIOByteBufferType, b)
				if ReferenceArrayGet[*ByteBuffer](a, 0, JavaNIOByteBufferType) != b {
					t.Error("array read changed receiver")
				}
			})
		})
	}
}

type byteBufferHomonymSource struct{ info *ObjectInfo }

func (b *byteBufferHomonymSource) JavaObjectInfo() *ObjectInfo { return b.info }
func TestByteBufferNativeNominalNullUnknownSourceShield(t *testing.T) {
	var absent *ByteBuffer
	if id, ok := ObjectDynamicType(absent); id != "" || ok {
		t.Error("typednil acquired dynamic Java class")
	}
	if ReferenceCheckCast[*ByteBuffer](absent, JavaNIOByteBufferType) != nil {
		t.Error("null cast changed null")
	}
	if id, ok := ObjectDynamicType(&ByteBuffer{}); id != "" || ok {
		t.Errorf("host zero carrier gained Java allocation identity %q", id)
	}
	id := TypeID("ByteBuffer")
	RegisterJavaType(id, ObjectTypeID)
	RegisterJavaSourceType(id)
	RegisterClassDescriptor(ClassDescriptor{Type: id, HasModifiers: true, Modifiers: 0, RuntimeLocation: ClassRuntimeLocation{Known: true, LoaderIdentity: AppClassLoaderIdentity, LoaderName: "app"}})
	source := &byteBufferHomonymSource{}
	source.info = NewObjectInfo(id, func(want TypeID) any {
		if want == id || want == ObjectTypeID {
			return source
		}
		return nil
	})
	if actual, ok := ObjectDynamicType(source); !ok || actual != id {
		t.Fatal("source class identity lost")
	}
	if ObjectView[*byteBufferHomonymSource](source, id) != source {
		t.Fatal("source provider/view identity lost")
	}
	var observed struct {
		Messages map[string]string `json:"messages"`
	}
	if err := json.Unmarshal([]byte(byteBufferJDKNominalExpectedJSON), &observed); err != nil {
		t.Fatal(err)
	}
	reject := func(label string, action func()) {
		t.Helper()
		defer func() {
			p := recover()
			exception, ok := p.(ClassCastException)
			if !ok {
				t.Errorf("%s must reject with exact CCE kind; actual=%v", label, p)
				return
			}
			if exception.Message() != observed.Messages[label] {
				t.Errorf("%s CCE message=%q actualJDK=%q", label, exception.Message(), observed.Messages[label])
			}
		}()
		action()
	}
	reject("native-to-source", func() { ReferenceCheckCast[*ByteBuffer](ByteBufferAllocate(1), id) })
	reject("source-to-native", func() { ReferenceCheckCast[any](source, JavaNIOByteBufferType) })
}

func TestByteBufferNativeDeclaredAndAllocatedTypes(t *testing.T) {
	declared := ClassLiteral(JavaNIOByteBufferType)
	if declared.TypeID() != JavaNIOByteBufferType || declared.GetName() != "java.nio.ByteBuffer" || declared.GetSuperclass().TypeID() != JavaNIOBufferType || declared.GetModifiers() != 1025 {
		t.Error("declared ByteBuffer class token changed to allocation class")
	}
	byteBufferNominalRun(t, "declaredComponentArray", func() {
		b := ByteBufferAllocate(1)
		a := ReferenceArrayLiteralOf[*ByteBuffer](JavaNIOByteBufferType, b)
		if id, ok := ObjectDynamicType(a); !ok || id != ArrayTypeID(JavaNIOByteBufferType) {
			t.Error("array declared component changed to dynamic heap class")
		}
		if ReferenceArrayGet[*ByteBuffer](a, 0, JavaNIOByteBufferType) != b {
			t.Error("declared-component array read changed identity")
		}
	})
}
func TestByteBufferNativeMetadataConcurrentRead(t *testing.T) {
	heap := ByteBufferAllocate(8)
	readOnly := heap.AsReadOnlyBuffer()
	values := []*ByteBuffer{heap, heap.Slice(), heap.Duplicate(), readOnly, readOnly.Slice(), readOnly.Duplicate()}
	var wg sync.WaitGroup
	failures := make(chan string, 20)
	for worker := 0; worker < 20; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if recover() != nil {
					failures <- "metadata/cast panic"
				}
			}()
			for i := 0; i < 64; i++ {
				for _, b := range values {
					actual, ok := ObjectDynamicType(b)
					if !ok || !JavaTypeAssignable(actual, JavaNIOByteBufferType) || !ClassRuntimeLocationOf(actual).Known || ReferenceCheckCast[*ByteBuffer](b, JavaNIOByteBufferType) != b {
						failures <- "metadata/cast identity mismatch"
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}
