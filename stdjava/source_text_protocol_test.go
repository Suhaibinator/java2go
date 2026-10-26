package stdjava

import "testing"

type sourceTextOldHook struct{ calls int }

func (*sourceTextOldHook) JavaDynamicTypeID() TypeID                  { return "text.SourceOldHook" }
func (v *sourceTextOldHook) StringJava2goExecution(*Execution) string { v.calls++; return "ordinary" }
func (*sourceTextOldHook) HashCode() int32                            { return 42 }

type sourceTextRealHook struct {
	expected *Execution
	calls    int
	text     string
	abrupt   any
}

func (*sourceTextRealHook) JavaDynamicTypeID() TypeID { return "text.SourceRealHook" }
func (v *sourceTextRealHook) Java2goToStringExecution(e *Execution) string {
	if e != v.expected {
		panic("lost caller execution")
	}
	v.calls++
	if v.abrupt != nil {
		panic(v.abrupt)
	}
	return v.text
}
func (*sourceTextRealHook) StringJava2goExecution(*Execution) string { panic("ordinary old selector") }

func TestSourceTextProtocolRejectsOrdinaryAndRetainsJavaResult(t *testing.T) {
	RegisterJavaType("text.SourceOldHook", ObjectTypeID)
	RegisterJavaSourceType("text.SourceOldHook")
	RegisterJavaType("text.SourceRealHook", ObjectTypeID)
	RegisterJavaSourceType("text.SourceRealHook")
	RegisterJavaSourceToString("text.SourceRealHook", "Java2goToStringExecution")
	e := NewExecution()
	ordinary := &sourceTextOldHook{}
	if got := StringValueOfExecution(e, ordinary); got != "text.SourceOldHook@2a" || ordinary.calls != 0 {
		t.Fatalf("text=%q ordinary calls=%d", got, ordinary.calls)
	}
	value := &sourceTextRealHook{expected: e, text: NullString()}
	if got := StringValueOfExecution(e, value); !StringIsNull(got) || value.calls != 1 {
		t.Fatal("null source result or call count changed")
	}
	marker := NewIllegalStateException("marker")
	value.abrupt = marker
	func() {
		defer func() {
			if !JavaReferenceEqual(recover(), marker) {
				t.Error("source exception changed")
			}
		}()
		StringValueOfExecution(e, value)
	}()
	if value.calls != 2 {
		t.Fatal("source callback count changed")
	}
}
