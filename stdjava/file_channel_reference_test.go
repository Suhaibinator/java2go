package stdjava

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type channelSourceProbe struct{ info *ObjectInfo }

func (p *channelSourceProbe) JavaObjectInfo() *ObjectInfo { return p.info }
func channelProbePanic(action func()) (failure any) {
	defer func() { failure = recover() }()
	action()
	return nil
}
func channelProbeRegistry(t *testing.T, id, parent TypeID, source bool) {
	t.Helper()
	javaTypeRegistry.Lock()
	previous, present := javaTypeRegistry.types[id]
	javaTypeRegistry.Unlock()
	t.Cleanup(func() {
		javaTypeRegistry.Lock()
		defer javaTypeRegistry.Unlock()
		if present {
			javaTypeRegistry.types[id] = previous
		} else {
			delete(javaTypeRegistry.types, id)
		}
	})
	RegisterJavaType(id, parent)
	if source {
		RegisterJavaSourceType(id)
	}
}
func channelProbeSource(t *testing.T, methods FileChannelSourceMethods) (*FileChannel, *channelSourceProbe) {
	t.Helper()
	id := TypeID("stageacontrol.SourceReadControl$SourceChannel")
	channelProbeRegistry(t, id, JavaNIOFileChannelType, true)
	root := &channelSourceProbe{}
	var channel *FileChannel
	root.info = NewObjectInfo(id, func(requested TypeID) any {
		if requested == id || requested == ObjectTypeID {
			return root
		}
		return channel
	})
	channel = NewFileChannelSourceReference(root.info, methods)
	return channel, root
}
func TestFileChannelSourceVirtualReadContracts(t *testing.T) {
	e := NewExecution()
	other := NewExecution()
	local := NewThreadLocal[string]()
	local.Set(e, "caller")
	boom := NewUnsupportedOperationException("original")
	var out, events strings.Builder
	calls := 0
	fail := false
	invoke := func(actual *Execution, label string) int32 {
		if actual != e || ThreadCurrentThread(actual) != ThreadCurrentThread(e) {
			t.Fatal("callback lost caller execution/thread")
		}
		calls++
		fmt.Fprintf(&events, "%s:%s;", label, local.Get(actual))
		if fail {
			panic(boom)
		}
		return 37
	}
	methods := FileChannelSourceMethods{Read: func(actual *Execution, _ *ByteBuffer) int32 { return invoke(actual, "read") }, ReadAt: func(actual *Execution, _ *ByteBuffer, p int64) int32 { return invoke(actual, fmt.Sprint("at=", p)) }, ImplClose: func(actual *Execution) {
		if actual != e {
			t.Fatal("close lost caller execution")
		}
		events.WriteString("close;")
	}}
	c, root := channelProbeSource(t, methods)
	another, _ := channelProbeSource(t, methods)
	var null *FileChannel
	readable := ObjectView[JavaReadableByteChannel](root, JavaNIOReadableByteChannelType)
	writable := ObjectView[JavaWritableByteChannel](root, JavaNIOWritableByteChannelType)
	fmt.Fprintf(&out, "views=%t:distinct=%t:class=%t:null=%t:initialCalls=%d\n", JavaReferenceEqual(c, readable) && JavaReferenceEqual(c, writable), !JavaReferenceEqual(c, another), ObjectGetClass(c) == ObjectGetClass(another), ObjectView[JavaReadableByteChannel](null, JavaNIOReadableByteChannelType) == nil, calls)
	guard := MonitorEnterExecution(e, c)
	fmt.Fprintf(&out, "monitorAlias=%t:otherMonitor=%t\n", ThreadHoldsLockExecution(e, root), ThreadHoldsLockExecution(e, another))
	MonitorExitExecution(guard)
	fmt.Fprintf(&out, "null=%d:negative=%d\n", FileChannelReadVirtualExecution(e, c, nil), FileChannelReadAtVirtualExecution(e, c, nil, -9))
	readonly := ByteBufferAllocate(2).AsReadOnlyBuffer()
	fmt.Fprintf(&out, "readonly=%d:position=%d\n", FileChannelReadAtVirtualExecution(e, c, readonly, -2), readonly.position)
	ThreadInterruptDefaultExecution(other, ThreadCurrentThread(e))
	fmt.Fprintf(&out, "interrupted=%d:flag=%t:open=%t\n", FileChannelReadAtVirtualExecution(e, c, nil, -3), ThreadIsInterruptedDefaultExecution(other, ThreadCurrentThread(e)), FileChannelIsOpenVirtual(c))
	ThreadInterruptedExecution(e)
	FileChannelCloseVirtualExecution(e, c)
	fmt.Fprintf(&out, "closed=%d:open=%t\n", FileChannelReadAtVirtualExecution(e, c, nil, -4), FileChannelIsOpenVirtual(c))
	fail = true
	actual := channelProbePanic(func() { FileChannelReadAtVirtualExecution(e, c, nil, -5) })
	fmt.Fprintf(&out, "sameThrow=%t\n", actual == boom)
	fmt.Fprintf(&out, "calls=%d:events=%s:identity=%t\n", calls, events.String(), JavaReferenceEqual(c, root))
	if out.String() != channelReadJDK {
		t.Fatalf("full JDK observations mismatch\ngot:\n%s\nwant:\n%s", out.String(), channelReadJDK)
	}
	// Each relative/positional override retains the same checks/effects ordering.
	fail = false
	before := calls
	for _, read := range []func(){func() { FileChannelReadVirtualExecution(e, c, nil) }, func() { FileChannelReadVirtualExecution(e, c, readonly) }, func() { FileChannelReadAtVirtualExecution(e, c, readonly, -99) }} {
		read()
	}
	if calls != before+3 || readonly.position != 0 {
		t.Fatal("closed/readonly override was validated or changed by facade")
	}
	for _, action := range []func(){func() { FileChannelReadVirtualExecution(nil, c, nil) }, func() { FileChannelReadAtVirtualExecution(nil, c, nil, -1) }, func() { FileChannelCloseVirtualExecution(nil, c) }} {
		if channelProbePanic(action) == nil {
			t.Fatal("missing execution accepted")
		}
	}
	if calls != before+3 {
		t.Fatal("invalid execution invoked callback")
	}
	local.Remove(e)
}
func TestFileChannelSourceCloseContracts(t *testing.T) {
	e := NewExecution()
	var out strings.Builder
	calls := 0
	boom := NewIOException("original-close")
	var c *FileChannel
	methods := FileChannelSourceMethods{Read: func(*Execution, *ByteBuffer) int32 { return 37 }, ReadAt: func(*Execution, *ByteBuffer, int64) int32 { return 37 }, ImplClose: func(actual *Execution) {
		calls++
		if actual != e || FileChannelIsOpenVirtual(c) {
			t.Fatal("close callback order/execution")
		}
		FileChannelCloseVirtualExecution(actual, c)
		panic(boom)
	}}
	c, _ = channelProbeSource(t, methods)
	actual := channelProbePanic(func() { FileChannelCloseVirtualExecution(e, c) })
	fmt.Fprintf(&out, "sameThrow=%t:open=%t\n", actual == boom, FileChannelIsOpenVirtual(c))
	FileChannelCloseVirtualExecution(e, c)
	fmt.Fprintf(&out, "reentrantOnce=%d\n", calls)
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan any, 1)
	secondDone := make(chan any, 1)
	secondStarting := make(chan struct{})
	concurrentCalls := 0
	first := NewExecution()
	second := NewExecution()
	main := NewExecution()
	var concurrent *FileChannel
	methods.ImplClose = func(actual *Execution) {
		if actual != first || FileChannelIsOpenVirtual(concurrent) {
			panic("wrong close callback")
		}
		concurrentCalls++
		close(entered)
		<-release
	}
	concurrent, _ = channelProbeSource(t, methods)
	guard := MonitorEnterExecution(main, concurrent)
	go func() { firstDone <- channelProbePanic(func() { FileChannelCloseVirtualExecution(first, concurrent) }) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("close incorrectly acquired source monitor")
	}
	fmt.Fprintf(&out, "sourceMonitorIndependent=%t\n", ThreadHoldsLockExecution(main, concurrent))
	MonitorExitExecution(guard)
	go func() {
		close(secondStarting)
		secondDone <- channelProbePanic(func() { FileChannelCloseVirtualExecution(second, concurrent) })
	}()
	<-secondStarting
	// This observation supplements the actual JDK BLOCKED-thread oracle. The
	// final completion and one-call assertions also run under the race detector.
	blocked := false
	select {
	case failure := <-secondDone:
		close(release)
		<-firstDone
		t.Fatalf("second close returned before original callback finished: %v", failure)
	case <-time.After(15 * time.Millisecond):
		blocked = true
	}
	fmt.Fprintf(&out, "blocked=%t:openDuring=%t\n", blocked, FileChannelIsOpenVirtual(concurrent))
	close(release)
	for _, done := range []chan any{firstDone, secondDone} {
		select {
		case failure := <-done:
			if failure != nil {
				t.Fatalf("unexpected close panic: %v", failure)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("close completion blocked")
		}
	}
	fmt.Fprintf(&out, "concurrentOnce=%d:finished=%t\n", concurrentCalls, true)
	if out.String() != channelCloseJDK {
		t.Fatalf("full JDK close mismatch\ngot:\n%s\nwant:\n%s", out.String(), channelCloseJDK)
	}
}
func TestFileChannelSourceReferenceIdentityAndInvariants(t *testing.T) {
	e := NewExecution()
	calls := 0
	methods := FileChannelSourceMethods{Read: func(*Execution, *ByteBuffer) int32 { calls++; return 37 }, ReadAt: func(*Execution, *ByteBuffer, int64) int32 { calls++; return 37 }, ImplClose: func(*Execution) { calls++ }}
	c, root := channelProbeSource(t, methods)
	other, _ := channelProbeSource(t, methods)
	if calls != 0 || c.JavaObjectInfo() != root.info || !JavaReferenceEqual(c, root) || JavaReferenceEqual(c, other) {
		t.Fatal("factory invoked callbacks or changed allocation identity")
	}
	if ObjectView[*FileChannel](root, JavaNIOFileChannelType) != c {
		t.Fatal("canonical view allocated or lost root")
	}
	array := NewReferenceArray(1, JavaNIOFileChannelType)
	ReferenceArraySet(array, 0, root)
	if ReferenceArrayGet[*FileChannel](array, 0, JavaNIOFileChannelType) != c {
		t.Fatal("source array view changed identity")
	}
	requireUnsupported := func(action func()) {
		t.Helper()
		if failure := channelProbePanic(action); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("invalid internal invariant was not refused: %v", failure)
		}
	}
	requireUnsupported(func() { NewFileChannelSourceReference(nil, methods) })
	for _, id := range []TypeID{JavaNIOFileChannelType, TypeID("sun.nio.ch.FileChannelImpl")} {
		for _, source := range []bool{false, true} {
			parent := JavaNIOFileChannelType
			if id == JavaNIOFileChannelType {
				parent = JavaNIOAbstractInterruptibleChannelType
			}
			channelProbeRegistry(t, id, parent, source)
			requireUnsupported(func() { NewFileChannelSourceReference(NewObjectInfo(id, nil), methods) })
		}
	}
	nonSource := TypeID("stageacontrol.NativeProviderInvariant")
	channelProbeRegistry(t, nonSource, JavaNIOFileChannelType, false)
	requireUnsupported(func() { NewFileChannelSourceReference(NewObjectInfo(nonSource, nil), methods) })
	wrong := TypeID("stageacontrol.NonChannelSourceInvariant")
	channelProbeRegistry(t, wrong, ObjectTypeID, true)
	requireUnsupported(func() { NewFileChannelSourceReference(NewObjectInfo(wrong, nil), methods) })
	for _, missing := range []FileChannelSourceMethods{{ReadAt: methods.ReadAt, ImplClose: methods.ImplClose}, {Read: methods.Read, ImplClose: methods.ImplClose}, {Read: methods.Read, ReadAt: methods.ReadAt}} {
		requireUnsupported(func() { NewFileChannelSourceReference(root.info, missing) })
	}
	for _, invalid := range []*FileChannel{{}, {state: &randomAccessState{}, source: c.source}} {
		requireUnsupported(func() { FileChannelReadVirtualExecution(e, invalid, nil) })
		requireUnsupported(func() { FileChannelCloseVirtualExecution(e, invalid) })
		requireUnsupported(func() { FileChannelIsOpenVirtual(invalid) })
	}
	for _, action := range []func(){func() { FileChannelReadVirtualExecution(e, nil, nil) }, func() { FileChannelReadAtVirtualExecution(e, nil, nil, -1) }, func() { FileChannelCloseVirtualExecution(e, nil) }, func() { FileChannelIsOpenVirtual(nil) }} {
		if failure := channelProbePanic(action); !CaughtAs(failure, "NullPointerException") {
			t.Fatalf("null receiver not canonical: %v", failure)
		}
	}
	for _, id := range []TypeID{"stageacontrol.FileChannel", "stageacontrol.FileChannelImpl"} {
		channelProbeRegistry(t, id, JavaNIOFileChannelType, true)
		if NewFileChannelSourceReference(NewObjectInfo(id, nil), methods) == nil {
			t.Fatal("source basename lookalike refused")
		}
	}
	if calls != 0 {
		t.Fatal("internal invariant validation invoked callback")
	}
}

// Full stdout of pinned JDK21 control, captured before any runtime producer.
const channelReadJDK = `views=true:distinct=true:class=true:null=true:initialCalls=0
monitorAlias=true:otherMonitor=false
null=37:negative=37
readonly=37:position=0
interrupted=37:flag=true:open=true
closed=37:open=false
sameThrow=true
calls=6:events=read:caller;at=-9:caller;at=-2:caller;at=-3:caller;close;at=-4:caller;at=-5:caller;:identity=true
`

// Full stdout of pinned JDK21 control, captured before any runtime producer.
const channelCloseJDK = `sameThrow=true:open=false
reentrantOnce=1
sourceMonitorIndependent=true
blocked=true:openDuring=false
concurrentOnce=1:finished=true
`
