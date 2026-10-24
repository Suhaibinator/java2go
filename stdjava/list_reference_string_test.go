package stdjava

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestCanonicalListStringUTF16NullAndSelfReference(t *testing.T) {
	execution := NewExecution()
	list := NewList[any]()
	list.Add(nil)
	list.Add(NewJavaStringUTF16([]uint16{0xD800, 0, 0xDFFF}))
	list.Add(list)
	want := append(utf16.Encode([]rune("[null, ")), 0xD800, 0, 0xDFFF)
	want = append(want, utf16.Encode([]rune(", (this Collection)]"))...)
	got := JavaStringValueOfExecution(execution, list)
	if got == nil || !slices.Equal(got.UTF16Copy(), want) {
		t.Fatal("List text lost null, isolated UTF16 units, or the self-reference sentinel")
	}
	if repeated := JavaStringValueOfExecution(execution, list); repeated == got {
		t.Fatal("nonempty List did not produce a fresh String result")
	}
	empty := NewList[any]()
	first, second := JavaStringValueOfExecution(execution, empty), JavaStringValueOfExecution(execution, empty)
	if first != second || first != JavaStringLiteralUTF16([]uint16{'[', ']'}) {
		t.Fatal("empty List text did not return AbstractCollection's exact interned literal")
	}
	expectBoxedException(t, "NullPointerException", func() {
		(*List[any])(nil).StringJava2goExecution(execution)
	})
}

type listSourceTextGuard struct {
	id      TypeID
	calls   int
	entered *Execution
	invoke  func() *JavaString
}

func (value *listSourceTextGuard) JavaDynamicTypeID() TypeID { return value.id }
func (value *listSourceTextGuard) DeclaredListText(execution *Execution) *JavaString {
	value.calls++
	value.entered = execution
	return value.invoke()
}
func (*listSourceTextGuard) String() string {
	panic("List source element used host text")
}

func newListSourceTextGuard(id TypeID, invoke func() *JavaString) *listSourceTextGuard {
	RegisterJavaType(id, ObjectTypeID)
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredListText")
	return &listSourceTextGuard{id: id, invoke: invoke}
}

func TestCanonicalListStringSourceOverrideExecutionAndLiveReplacement(t *testing.T) {
	execution := NewExecution()
	list := NewList[any]()
	firstText := NewJavaStringUTF16([]uint16{0xD800})
	first := newListSourceTextGuard("list.reference.Replace", func() *JavaString {
		list.Set(1, NewJavaStringUTF16([]uint16{0xDFFF}))
		return firstText
	})
	null := newListSourceTextGuard("list.reference.Null", func() *JavaString { return nil })
	list.Add(first)
	list.Add(NewJavaStringUTF16([]uint16{'o', 'l', 'd'}))
	list.Add(null)
	got := JavaStringValueOfExecution(execution, list)
	want := []uint16{'[', 0xD800, ',', ' ', 0xDFFF, ',', ' ', 'n', 'u', 'l', 'l', ']'}
	if got == nil || !slices.Equal(got.UTF16Copy(), want) || first.calls != 1 || null.calls != 1 || first.entered != execution || null.entered != execution {
		t.Fatal("List conversion lost exact source text, live replacement, null normalization, or caller execution")
	}
}

func TestCanonicalListStringCallbackExceptionAndFailFast(t *testing.T) {
	t.Run("exception", func(t *testing.T) {
		execution := NewExecution()
		failure := NewIllegalStateException("element conversion")
		first := newListSourceTextGuard("list.reference.Throw", func() *JavaString { panic(failure) })
		second := newListSourceTextGuard("list.reference.AfterThrow", func() *JavaString { return nil })
		list := NewListFrom[any](first, second)
		defer func() {
			if got := recover(); got != failure || first.calls != 1 || first.entered != execution || second.calls != 0 {
				t.Fatal("List conversion changed exception identity or invoked a later element")
			}
		}()
		JavaStringValueOfExecution(execution, list)
	})
	t.Run("structural mutation", func(t *testing.T) {
		execution := NewExecution()
		list := NewList[any]()
		first := newListSourceTextGuard("list.reference.Mutate", func() *JavaString {
			list.Add(nil)
			return NewJavaStringUTF16([]uint16{'f'})
		})
		second := newListSourceTextGuard("list.reference.AfterMutation", func() *JavaString { return nil })
		list.Add(first)
		list.Add(second)
		expectBoxedException(t, "ConcurrentModificationException", func() {
			JavaStringValueOfExecution(execution, list)
		})
		if first.calls != 1 || first.entered != execution || second.calls != 0 || list.Size() != 3 {
			t.Fatal("List conversion lost callback effects or failed at the wrong iterator boundary")
		}
	})
}
