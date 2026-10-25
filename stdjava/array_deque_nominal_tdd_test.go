package stdjava

import "testing"

// Canonical JDK21 ArrayDeque ancestry must be supplied by the runtime carrier.
// Public Go method shape never supplies Java nominal Deque membership.
func TestArrayDequeCanonicalNominal224(t *testing.T) {
	for _, parent := range []TypeID{"java.util.AbstractCollection", "java.util.Deque", "java.util.Queue", "java.util.Collection", "java.lang.Iterable", "java.lang.Cloneable", "java.io.Serializable"} {
		t.Run(string(parent), func(t *testing.T) {
			if !JavaTypeAssignable("java.util.ArrayDeque", parent) {
				t.Fatalf("missing canonical ArrayDeque ancestry: %s", parent)
			}
		})
	}
	t.Run("Object", func(t *testing.T) {
		if !JavaTypeAssignable("java.util.ArrayDeque", ObjectTypeID) {
			t.Fatal("ArrayDeque is not an Object")
		}
	})
	for _, unrelated := range []TypeID{"java.util.List", "java.util.RandomAccess"} {
		t.Run(string(unrelated), func(t *testing.T) {
			if JavaTypeAssignable("java.util.ArrayDeque", unrelated) {
				t.Fatalf("ArrayDeque admitted unrelated nominal type: %s", unrelated)
			}
		})
	}
	t.Run("structural_lookalike", func(t *testing.T) {
		value := &arrayDeque224ShapeOnly{}
		if ObjectInstanceOf(value, "java.util.Deque") {
			t.Fatal("public Go method shape admitted as nominal Deque")
		}
	})
}

type arrayDeque224ShapeOnly struct{}

func (*arrayDeque224ShapeOnly) IteratorJava2goExecution(*Execution) JavaIterator { return nil }
func (*arrayDeque224ShapeOnly) AddFirst(any)                                     {}
func (*arrayDeque224ShapeOnly) AddLast(any)                                      {}
func (*arrayDeque224ShapeOnly) RemoveFirst() any                                 { return nil }
func (*arrayDeque224ShapeOnly) RemoveLast() any                                  { return nil }
func (*arrayDeque224ShapeOnly) Size() int32                                      { return 0 }
func (*arrayDeque224ShapeOnly) IsEmpty() bool                                    { return true }
