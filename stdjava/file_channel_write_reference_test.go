package stdjava

import "testing"

func writerSourceMethods() FileChannelSourceMethods {
	return FileChannelSourceMethods{Read: func(*Execution, *ByteBuffer) int32 { return 0 }, ReadAt: func(*Execution, *ByteBuffer, int64) int32 { return 0 }, ImplClose: func(*Execution) {}}
}
func TestFileChannelWriteSourceVirtualCallerAndValidation(t *testing.T) {
	constructor, caller := NewExecution(), NewExecution()
	_ = constructor
	local := NewThreadLocal[string]()
	local.Set(caller, "caller")
	defer local.Remove(caller)
	methods := writerSourceMethods()
	calls := 0
	boom := NewIOException("source-write")
	invoke := func(actual *Execution, src *ByteBuffer, p int64) int32 {
		if actual != caller || local.Get(actual) != "caller" || ThreadCurrentThread(actual) != ThreadCurrentThread(caller) {
			t.Fatal("lost invocation Execution")
		}
		calls++
		if p == 19 {
			panic(boom)
		}
		if src != nil {
			src.position++
		}
		return int32(30 + p)
	}
	methods.Write = func(e *Execution, b *ByteBuffer) int32 { return invoke(e, b, 1) }
	methods.WriteAt = invoke
	c, root := channelProbeSource(t, methods)
	if calls != 0 || !JavaReferenceEqual(c, root) {
		t.Fatal("construction invoked or changed source")
	}
	if FileChannelWriteVirtualExecution(caller, c, nil) != 31 || FileChannelWriteAtVirtualExecution(caller, c, nil, -9) != 21 {
		t.Fatal("native argument checks preceded override")
	}
	readonly := ByteBufferAllocate(2).AsReadOnlyBuffer()
	if FileChannelWriteAtVirtualExecution(caller, c, readonly, -2) != 28 || readonly.position != 1 {
		t.Fatal("readonly source inspected or callback effects lost")
	}
	ThreadInterruptDefaultExecution(NewExecution(), ThreadCurrentThread(caller))
	defer ThreadInterruptedExecution(caller)
	if FileChannelWriteVirtualExecution(caller, c, nil) != 31 || !ThreadIsInterruptedDefaultExecution(NewExecution(), ThreadCurrentThread(caller)) || !FileChannelIsOpenVirtual(c) {
		t.Fatal("source preinterrupt was validated natively")
	}
	FileChannelCloseVirtualExecution(caller, c)
	if FileChannelWriteAtVirtualExecution(caller, c, nil, -4) != 26 || FileChannelIsOpenVirtual(c) {
		t.Fatal("closed source override intercepted")
	}
	if channelProbePanic(func() { FileChannelWriteAtVirtualExecution(caller, c, nil, 19) }) != boom {
		t.Fatal("source exception replaced")
	}
	before := calls
	for _, action := range []func(){func() { FileChannelWriteVirtualExecution(nil, c, nil) }, func() { FileChannelWriteAtVirtualExecution(nil, c, nil, -1) }} {
		if channelProbePanic(action) == nil {
			t.Fatal("missing Execution admitted")
		}
	}
	if calls != before {
		t.Fatal("invalid Execution called source")
	}
}
func TestFileChannelWriteSourceOptionalPairInvariants(t *testing.T) {
	methods := writerSourceMethods()
	c, root := channelProbeSource(t, methods)
	e := NewExecution()
	for _, action := range []func(){func() { FileChannelWriteVirtualExecution(e, c, nil) }, func() { FileChannelWriteAtVirtualExecution(e, c, nil, -1) }} {
		if failure := channelProbePanic(action); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("legacy absent writes fell through native: %#v", failure)
		}
	}
	for _, first := range []bool{false, true} {
		partial := methods
		if first {
			partial.Write = func(*Execution, *ByteBuffer) int32 { return 1 }
		} else {
			partial.WriteAt = func(*Execution, *ByteBuffer, int64) int32 { return 2 }
		}
		if failure := channelProbePanic(func() { NewFileChannelSourceReference(root.info, partial) }); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("one-only pair admitted: %#v", failure)
		}
	}
	for _, invalid := range []*FileChannel{{}, {state: &randomAccessState{}, source: c.source}} {
		if failure := channelProbePanic(func() { FileChannelWriteVirtualExecution(e, invalid, nil) }); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatal("invalid backing admitted")
		}
	}
	for _, action := range []func(){func() { FileChannelWriteVirtualExecution(e, nil, nil) }, func() { FileChannelWriteAtVirtualExecution(e, nil, nil, -1) }} {
		if failure := channelProbePanic(action); !CaughtAs(failure, "NullPointerException") {
			t.Fatal("null receiver not canonical")
		}
	}
}
