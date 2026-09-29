package stdjava_test

import (
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

type nominalTextLookalike struct {
	*j.ObjectInfo
	execution                       *j.Execution
	messages, names, errors, hashes int
	held                            bool
}

func (p *nominalTextLookalike) Message() string           { p.messages++; return "unrelated-message" }
func (p *nominalTextLookalike) ThrowableTypeName() string { p.names++; return "Pretend" }
func (p *nominalTextLookalike) Error() string             { p.errors++; return "unrelated-error" }
func (p *nominalTextLookalike) HashCodeJava2goExecution(e *j.Execution) int32 {
	if e == nil || (p.execution != nil && e != p.execution) {
		panic("lost hashCode execution")
	}
	p.hashes++
	p.held = j.ThreadHoldsLockExecution(e, p)
	return 42
}

func TestStringConversionNominalLookalikeUsesObjectBody(t *testing.T) {
	j.RegisterJavaType("text.nominal.Lookalike", j.ObjectTypeID)
	p := &nominalTextLookalike{}
	p.ObjectInfo = j.NewObjectInfo("text.nominal.Lookalike", func(j.TypeID) any { return p })
	var structural j.Throwable = p
	if j.ObjectInstanceOf(structural, j.ThrowableTypeID) {
		t.Fatal("ordinary class is nominal Throwable")
	}
	e := j.NewExecution()
	p.execution = e
	guard := j.MonitorEnterExecution(e, p)
	defer j.MonitorExitExecution(guard)
	if got := j.StringValueOfExecution(e, structural); got != "text.nominal.Lookalike@2a" {
		t.Fatalf("default Java text = %q", got)
	}
	if !p.held || p.hashes != 1 {
		t.Fatal("default body lost virtual hashCode or caller monitor")
	}
	p.execution = nil
	if got := j.StringValueOf(p); got != "text.nominal.Lookalike@2a" {
		t.Fatalf("native entry Java text = %q", got)
	}
	if p.messages != 0 || p.names != 0 || p.errors != 0 || p.hashes != 2 {
		t.Fatalf("wrong callbacks: message%d name%d error%d hash%d", p.messages, p.names, p.errors, p.hashes)
	}
}

type nominalTextOverride struct {
	*j.ObjectInfo
	execution *j.Execution
	calls     int
	text      string
	abrupt    any
}

func (p *nominalTextOverride) StringJava2goExecution(e *j.Execution) string {
	if e != p.execution || !j.ThreadHoldsLockExecution(e, p) {
		panic("lost override execution or receiver")
	}
	p.calls++
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.text
}

type nominalTextRenamedOverride struct {
	*j.ObjectInfo
	execution *j.Execution
	calls     int
}

func (p *nominalTextRenamedOverride) StringJava2goExecution1(e *j.Execution) string {
	if e != p.execution {
		panic("lost renamed override execution")
	}
	p.calls++
	return "renamed"
}

func TestStringConversionNominalOverridesPreserveViewNullAndAbrupt(t *testing.T) {
	j.RegisterJavaType("text.nominal.Override", j.ObjectTypeID)
	e := j.NewExecution()
	p := &nominalTextOverride{execution: e, text: j.NullString()}
	p.ObjectInfo = j.NewObjectInfo("text.nominal.Override", func(j.TypeID) any { return p })
	// A base view carries the same canonical object info but no text method.
	base := struct{ *j.ObjectInfo }{p.ObjectInfo}
	guard := j.MonitorEnterExecution(e, p)
	defer j.MonitorExitExecution(guard)
	if got := j.StringValueOfExecution(e, base); !j.StringIsNull(got) || p.calls != 1 {
		t.Fatal("base view failed to preserve null source override")
	}
	marker := j.NewIllegalStateException("marker")
	p.abrupt = marker
	func() {
		defer func() {
			if !j.JavaReferenceEqual(recover(), marker) {
				t.Error("source abrupt completion changed")
			}
		}()
		j.StringValueOfExecution(e, base)
	}()
	if p.calls != 2 {
		t.Fatal("override evaluated more than once")
	}
	r := &nominalTextRenamedOverride{execution: e}
	r.ObjectInfo = j.NewObjectInfo("text.nominal.Renamed", func(j.TypeID) any { return r })
	if got := j.StringValueOfExecution(e, r); got != "renamed" || r.calls != 1 {
		t.Fatal("renamed companion bypassed")
	}
	var absent *nominalTextOverride
	if j.StringValueOfExecution(e, absent) != "null" || j.StringValueOf(absent) != "null" || j.StringValueOfExecution(e, j.NullString()) != "null" {
		t.Fatal("null conversion changed")
	}
}

type nativeTextStringer struct{}

func (nativeTextStringer) String() string { return "native-string" }

type nativeTextError struct{}

func (nativeTextError) Error() string { return "native-error" }

type nativeTextStructural struct{}

func (nativeTextStructural) ThrowableTypeName() string { return "Pretend" }
func (nativeTextStructural) Message() string           { panic("ordinary native Message called") }
func (nativeTextStructural) Error() string             { return "native-structural" }
func TestStringConversionNominalKeepsNativeFormatting(t *testing.T) {
	e := j.NewExecution()
	for _, test := range []struct {
		value any
		want  string
	}{{nativeTextStringer{}, "native-string"}, {nativeTextError{}, "native-error"}, {nativeTextStructural{}, "native-structural"}, {j.NewParseException("detail", 0), "java.text.ParseException: detail"}} {
		if got := j.StringValueOf(test.value); got != test.want {
			t.Fatalf("native entry %T = %q", test.value, got)
		}
		if got := j.StringValueOfExecution(e, test.value); got != test.want {
			t.Fatalf("execution entry %T = %q", test.value, got)
		}
	}
}

func TestStringConversionNominalNativeCauseEntry(t *testing.T) {
	e := j.NewExecution()
	cause := &nativeTextStructural{}
	constructors := []struct {
		name string
		make func() j.Throwable
	}{
		{"Exception native", func() j.Throwable { return j.NewException(cause) }},
		{"Exception execution", func() j.Throwable { return j.NewExceptionExecution(e, cause) }},
		{"Throwable native", func() j.Throwable { return j.NewThrowable(cause) }},
		{"Throwable execution", func() j.Throwable { return j.NewThrowableExecution(e, cause) }},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			value := constructor.make()
			if value.Message() != "native-structural" || j.GetCause(value) != cause {
				t.Fatal("native structural cause text or identity changed")
			}
		})
	}
}

type nominalTextLeaf struct {
	messages, names, errors, hashes int
	execution                       *j.Execution
}

func (*nominalTextLeaf) JavaDynamicTypeID() j.TypeID { return "text.nominal.Leaf" }
func (p *nominalTextLeaf) Message() string           { p.messages++; return "unrelated" }
func (p *nominalTextLeaf) ThrowableTypeName() string { p.names++; return "Pretend" }
func (p *nominalTextLeaf) Error() string             { p.errors++; return "unrelated" }
func (p *nominalTextLeaf) HashCodeJava2goExecution(e *j.Execution) int32 {
	if e != p.execution {
		panic("leaf lost caller execution")
	}
	p.hashes++
	return 42
}

type nominalNativeDescriptor struct{}

func (nominalNativeDescriptor) JavaDynamicTypeID() j.TypeID { return "native.Descriptor" }
func (nominalNativeDescriptor) String() string              { return "native-descriptor" }

func TestStringConversionNominalDescriptorOnlySource(t *testing.T) {
	// Source registration is independent of hierarchy registration ordering.
	j.RegisterJavaSourceType("text.nominal.Leaf")
	j.RegisterJavaType("text.nominal.Leaf", j.ObjectTypeID)
	p := &nominalTextLeaf{execution: j.NewExecution()}
	if got := j.StringValueOfExecution(p.execution, p); got != "text.nominal.Leaf@2a" {
		t.Fatalf("descriptor-only source = %q", got)
	}
	if p.messages != 0 || p.names != 0 || p.errors != 0 || p.hashes != 1 {
		t.Fatal("descriptor-only source invoked unrelated method")
	}
	j.RegisterJavaType("native.Descriptor", j.ObjectTypeID)
	for _, got := range []string{j.StringValueOf(nominalNativeDescriptor{}), j.StringValueOfExecution(p.execution, nominalNativeDescriptor{})} {
		if got != "native-descriptor" {
			t.Fatalf("native descriptor fmt behavior changed: %q", got)
		}
	}
}
