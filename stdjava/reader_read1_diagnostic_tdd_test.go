package stdjava

import "testing"

type readerDefaultDiagnosticSource struct {
	*ReaderBase
	body func(*Execution, *PrimitiveArray[rune], int32, int32) int32
}

func (source *readerDefaultDiagnosticSource) JavaReaderReadChars(e *Execution, cbuf *PrimitiveArray[rune], off, n int32) int32 {
	return source.body(e, cbuf, off, n)
}

type readerDefaultDiagnosticOverride struct {
	*readerDefaultDiagnosticSource
	read1 func(*Execution, *PrimitiveArray[rune]) int32
}

func (source *readerDefaultDiagnosticOverride) JavaReaderReadArray(e *Execution, cbuf *PrimitiveArray[rune]) int32 {
	return source.read1(e, cbuf)
}

func TestReaderRead1InheritedDiagnosticAndOverride(t *testing.T) {
	e := NewExecution()
	local := NewThreadLocal[string]()
	local.Set(e, "caller")
	defer local.Remove(e)
	owner := ThreadCurrentThread(e)
	calls := 0
	source := &readerDefaultDiagnosticSource{ReaderBase: NewReaderBase()}
	source.body = func(actual *Execution, cbuf *PrimitiveArray[rune], off, n int32) int32 {
		calls++
		if actual != e || local.Get(actual) != "caller" || ThreadCurrentThread(actual) != owner || !ThreadHoldsLockExecution(actual, source) {
			t.Error("virtual read3 lost caller execution/threadlocal/thread/monitor")
		}
		if cbuf == nil || off != 0 || n != int32(len(cbuf.Elements)) {
			t.Fatal("inherited read1 arguments")
		}
		if n == 0 {
			return 0
		}
		cbuf.Elements[0] = 'Q'
		return 1
	}
	literal := stringReaderLiteral(`Cannot read the array length because "cbuf" is null`)
	first := readerControlPanic(func() { ReaderReadArrayExecution(e, source, nil) })
	second := readerControlPanic(func() { ReaderReadArrayExecution(e, source, nil) })
	for _, value := range []any{first, second} {
		if !CaughtAs(value, "NullPointerException") || GetMessage(value) != `Cannot read the array length because "cbuf" is null` {
			t.Errorf("inherited default diagnostic got %T:%v", value, value)
		}
	}
	if calls != 0 {
		t.Fatal("null length must precede virtual read3")
	}
	firstMessage, secondMessage := JavaThrowableMessageExecution(e, first), JavaThrowableMessageExecution(e, second)
	if JavaReferenceEqual(firstMessage, secondMessage) || JavaReferenceEqual(firstMessage, literal) || !JavaReferenceEqual(InternJavaString(firstMessage), literal) {
		t.Error("enhanced diagnostic must be fresh noninterned canonical String")
	}
	guard := MonitorEnterExecution(e, source)
	defer MonitorExitExecution(guard)
	chars := NewPrimitiveArray[rune](2, PrimitiveTypeID("char"))
	if ReaderReadArrayExecution(e, source, chars) != 1 || chars.Elements[0] != 'Q' || ReaderReadArrayExecution(e, source, NewPrimitiveArray[rune](0, PrimitiveTypeID("char"))) != 0 || calls != 2 {
		t.Fatal("valid/zero array virtual read3")
	}
	overridden := &readerDefaultDiagnosticOverride{readerDefaultDiagnosticSource: &readerDefaultDiagnosticSource{ReaderBase: NewReaderBase(), body: func(*Execution, *PrimitiveArray[rune], int32, int32) int32 {
		t.Fatal("read1 override bypassed")
		return 0
	}}}
	overrideCalls := 0
	overridden.read1 = func(actual *Execution, cbuf *PrimitiveArray[rune]) int32 {
		overrideCalls++
		if actual != e || cbuf != nil || local.Get(actual) != "caller" || !ThreadHoldsLockExecution(actual, overridden) {
			t.Fatal("source read1 override lost null or Execution/lock")
		}
		return 73
	}
	overrideGuard := MonitorEnterExecution(e, overridden)
	defer MonitorExitExecution(overrideGuard)
	if ReaderReadArrayExecution(e, overridden, nil) != 73 || overrideCalls != 1 {
		t.Fatal("source read1 override must precede inherited null validation")
	}
	var absent *readerDefaultDiagnosticOverride
	if value := readerControlPanic(func() { ReaderReadArrayExecution(e, absent, nil) }); !CaughtAs(value, "NullPointerException") || overrideCalls != 1 {
		t.Fatal("receiver must be checked before source override")
	}
	native := NewStringReaderReference(NewJavaStringUTF16([]uint16{'Q'}))
	StringReaderCloseExecution(e, native)
	requireReaderFailure(t, "NullPointerException", `Cannot read the array length because "cbuf" is null`, func() { ReaderReadArrayExecution(e, native, nil) })
	requireReaderFailure(t, "IOException", "Stream closed", func() { ReaderReadCharsExecution(e, native, nil, 0, 0) })
}
