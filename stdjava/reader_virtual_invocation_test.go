package stdjava

import "testing"

const (
	readerVirtualBaseID = TypeID("reader.virtual.fixture.Base")
	readerVirtualDerivedID = TypeID("reader.virtual.fixture.Derived")
)

// This is the existing generated hierarchy shape: the source base has read3,
// the derived source has real read0/read1/read3 bridges, and every Java view
// shares ObjectInfo for the most-derived allocation.
type readerVirtualBase struct {
	*ReaderBase
	*ObjectInfo
	baseCalls int
}

func (base *readerVirtualBase) JavaReaderReadChars(_ *Execution, buffer *PrimitiveArray[rune], offset, length int32) int32 {
	base.baseCalls++
	if buffer != nil && offset >= 0 && length > 0 && int(offset) < len(buffer.Elements) {
		buffer.Elements[offset] = 'B'
	}
	return 1
}

func (*readerVirtualBase) Java2goReferenceDynamicType() TypeID { return readerVirtualBaseID }
func (base *readerVirtualBase) Java2goReferenceView(requested TypeID) any {
	if requested == readerVirtualBaseID {
		return base
	}
	return nil
}

type readerVirtualDerived struct {
	*readerVirtualBase
	calls [3]int
	executions [3]*Execution
	buffer *PrimitiveArray[rune]
	offset, length int32
	monitorHeld bool
}

func (*readerVirtualDerived) Java2goReferenceDynamicType() TypeID { return readerVirtualDerivedID }
func (derived *readerVirtualDerived) Java2goReferenceView(requested TypeID) any {
	switch requested {
	case readerVirtualDerivedID:
		return derived
	case readerVirtualBaseID:
		return derived.readerVirtualBase
	default:
		return nil
	}
}
func (derived *readerVirtualDerived) record(index int, execution *Execution) {
	derived.calls[index]++
	derived.executions[index] = execution
	derived.monitorHeld = ThreadHoldsLockExecution(execution, derived)
}
func (derived *readerVirtualDerived) JavaReaderReadChar(execution *Execution) int32 {
	derived.record(0, execution)
	return 93
}
func (derived *readerVirtualDerived) JavaReaderReadArray(execution *Execution, buffer *PrimitiveArray[rune]) int32 {
	derived.record(1, execution)
	derived.buffer = buffer
	if buffer == nil {
		return 31
	}
	return 32
}
func (derived *readerVirtualDerived) JavaReaderReadChars(execution *Execution, buffer *PrimitiveArray[rune], offset, length int32) int32 {
	derived.record(2, execution)
	derived.buffer, derived.offset, derived.length = buffer, offset, length
	if buffer != nil && offset >= 0 && length > 0 && int(offset) < len(buffer.Elements) {
		buffer.Elements[offset] = 'Z'
	}
	return 73
}

func readerVirtualFixture() *readerVirtualDerived {
	RegisterJavaType(readerVirtualBaseID, JavaIOReaderType)
	RegisterJavaSourceType(readerVirtualBaseID)
	RegisterJavaType(readerVirtualDerivedID, readerVirtualBaseID)
	RegisterJavaSourceType(readerVirtualDerivedID)
	derived := &readerVirtualDerived{readerVirtualBase: &readerVirtualBase{}}
	derived.ReaderBase = NewReaderBase(derived)
	derived.ObjectInfo = NewGeneratedObjectInfo(derived)
	return derived
}

func readerVirtualExistingPanic(body func()) (caught any) {
	defer func() { caught = recover() }()
	body()
	return nil
}

func TestReaderVirtualInvocationMostDerivedExistingAPI(t *testing.T) {
	cases := []struct {
		name string
		index int
		want int32
		invoke func(*Execution, Reader) int32
	}{
		{"read0", 0, 93, func(execution *Execution, reader Reader) int32 { return ReaderReadCharExecution(execution, reader) }},
		{"read1Null", 1, 31, func(execution *Execution, reader Reader) int32 { return ReaderReadArrayExecution(execution, reader, nil) }},
		{"read3InvalidArguments", 2, 73, func(execution *Execution, reader Reader) int32 { return ReaderReadCharsExecution(execution, reader, nil, -7, -9) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			derived := readerVirtualFixture()
			view := derived.readerVirtualBase
			execution := NewExecution()
			guard := MonitorEnterExecution(execution, view)
			defer MonitorExitExecution(guard)
			var result int32
			caught := readerVirtualExistingPanic(func() { result = test.invoke(execution, view) })
			if caught != nil || result != test.want {
				t.Errorf("base view must dispatch most-derived source bridge: result=%d want=%d panic=%T:%v", result, test.want, caught, caught)
			}
			if derived.calls[test.index] != 1 || derived.executions[test.index] != execution || !derived.monitorHeld || derived.baseCalls != 0 {
				t.Errorf("wrong source receiver/execution/monitor: calls=%v execution=%p want=%p monitor=%v baseCalls=%d", derived.calls, derived.executions[test.index], execution, derived.monitorHeld, derived.baseCalls)
			}
			if !JavaReferenceEqual(view, derived) || view.JavaObjectInfo() != derived.JavaObjectInfo() {
				t.Fatal("fixture must preserve the existing allocation identity")
			}
			if test.index == 2 && (derived.buffer != nil || derived.offset != -7 || derived.length != -9) {
				t.Error("source read3 must receive invalid arguments without native prevalidation")
			}
		})
	}
}

func TestReaderVirtualInvocationDirectAndDefaultExistingAPI(t *testing.T) {
	t.Run("directMostDerived", func(t *testing.T) {
		derived := readerVirtualFixture()
		execution := NewExecution()
		guard := MonitorEnterExecution(execution, derived)
		defer MonitorExitExecution(guard)
		if ReaderReadCharExecution(execution, derived) != 93 || ReaderReadArrayExecution(execution, derived, nil) != 31 || ReaderReadCharsExecution(execution, derived, nil, -7, -9) != 73 {
			t.Fatal("direct source bridges changed")
		}
		if derived.calls != [3]int{1, 1, 1} || derived.baseCalls != 0 || !derived.monitorHeld {
			t.Fatal("direct source receiver changed")
		}
	})
	t.Run("explicitReaderDefaultsStillInvokeVirtualRead3", func(t *testing.T) {
		derived := readerVirtualFixture()
		execution := NewExecution()
		buffer := NewPrimitiveArray[rune](1, PrimitiveTypeID("char"))
		if ReaderReadCharDefaultExecution(execution, derived.readerVirtualBase) != 'Z' || ReaderReadArrayDefaultExecution(execution, derived.readerVirtualBase, buffer) != 73 || buffer.Elements[0] != 'Z' {
			t.Error("Reader default bodies must retain virtual read3 dispatch")
		}
		if derived.calls != [3]int{0, 0, 2} || derived.baseCalls != 0 || derived.executions[2] != execution {
			t.Errorf("default call selected wrong source declaration: calls=%v baseCalls=%d", derived.calls, derived.baseCalls)
		}
	})
}

func TestReaderVirtualInvocationMetadataBoundaryExistingAPI(t *testing.T) {
	for _, name := range []string{"noInfo", "unknown", "nativeReader", "sourceNonReader"} {
		t.Run(name, func(t *testing.T) {
			base := &readerVirtualBase{ReaderBase: NewReaderBase()}
			providerCalls := 0
			id := TypeID("reader.virtual.boundary." + name)
			switch name {
			case "nativeReader":
				RegisterJavaType(id, JavaIOReaderType)
			case "sourceNonReader":
				RegisterJavaType(id, ObjectTypeID)
				RegisterJavaSourceType(id)
			}
			if name != "noInfo" {
				base.ObjectInfo = NewObjectInfo(id, func(TypeID) any { providerCalls++; return nil })
			}
			if ReaderReadCharsExecution(NewExecution(), base, nil, -7, -9) != 1 || base.baseCalls != 1 || providerCalls != 0 {
				t.Fatal("uncertified/native/opaque existing Reader protocol must not acquire a source identity view")
			}
		})
	}
	t.Run("nullReceiver", func(t *testing.T) {
		var absent *readerVirtualBase
		if caught := readerVirtualExistingPanic(func() { ReaderReadArrayExecution(NewExecution(), absent, nil) }); !CaughtAs(caught, "NullPointerException") {
			t.Fatalf("null receiver changed: %T:%v", caught, caught)
		}
	})
}

func TestReaderVirtualInvocationMalformedSourceProviderExistingAPI(t *testing.T) {
	for _, name := range []string{"nilProvider", "nilView", "typedNilView", "wrongProtocol", "foreignAllocation"} {
		t.Run(name, func(t *testing.T) {
			derived := readerVirtualFixture()
			foreign := readerVirtualFixture()
			var provider func(TypeID) any
			if name != "nilProvider" {
				provider = func(TypeID) any {
					switch name {
					case "typedNilView":
						var absent *readerVirtualDerived
						return absent
					case "wrongProtocol":
						return NewJavaStringUTF16([]uint16{'x'})
					case "foreignAllocation":
						return foreign
					default:
						return nil
					}
				}
			}
			derived.ObjectInfo = NewObjectInfo(readerVirtualDerivedID, provider)
			caught := readerVirtualExistingPanic(func() { ReaderReadCharsExecution(NewExecution(), derived.readerVirtualBase, nil, -7, -9) })
			if !CaughtAs(caught, "ClassCastException") || derived.baseCalls != 0 || derived.calls != [3]int{} || foreign.calls != [3]int{} {
				t.Errorf("invalid proven source view must reject before callback: panic=%T:%v base=%d actual=%v foreign=%v", caught, caught, derived.baseCalls, derived.calls, foreign.calls)
			}
		})
	}
}

func TestReaderVirtualInvocationCloneBindingExistingAPI(t *testing.T) {
	original := readerVirtualFixture()
	clone := &readerVirtualDerived{readerVirtualBase: &readerVirtualBase{ReaderBase: CloneReaderBaseReference(original.ReaderBase)}}
	clone.ObjectInfo = NewGeneratedObjectInfo(clone)
	if JavaReferenceEqual(original, clone) || !JavaReferenceEqual(clone.readerVirtualBase, clone) || !JavaReferenceEqual(clone.Lock, original) {
		t.Fatal("clone identity/default original lock contract invalid")
	}
	execution := NewExecution()
	if ReaderReadCharExecution(execution, clone.readerVirtualBase) != 93 || clone.calls[0] != 1 || original.calls != [3]int{} || clone.executions[0] != execution {
		t.Errorf("clone base view callback did not rebind to clone: original=%v clone=%v", original.calls, clone.calls)
	}
	clone.Lock = nil
	if original.Lock == nil {
		t.Fatal("later protected lock writes must remain independent")
	}
}
