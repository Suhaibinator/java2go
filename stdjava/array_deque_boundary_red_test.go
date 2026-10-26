package stdjava

import "testing"

func TestArrayDequeBoundaryRuntime224(t *testing.T) {
	t.Run("raw_concrete_class", func(t *testing.T) {
		d := NewArrayDeque[*JavaString]()
		var view *ArrayDeque
		observed := arrayDequeBoundary224Capture(func() { view = ObjectView[*ArrayDeque](d, ArrayDequeTypeID) })
		if observed != nil {
			if !ObjectInstanceOf(observed, "java.lang.ClassCastException") {
				t.Fatalf("unexpected raw-cast fixture panic: %T", observed)
			}
			t.Fatal("legal raw ArrayDeque class cast rejected erased allocation")
		}
		if !JavaReferenceEqual(d, view) {
			t.Fatal("raw class view changed allocation identity")
		}
	})

	for _, setup := range []struct {
		name  string
		exact bool
	}{{"exact_copy", true}, {"spare_default", false}} {
		for _, matching := range []bool{false, true} {
			setup, matching := setup, matching
			name := setup.name + "_nonmatching"
			if matching {
				name = setup.name + "_matching"
			}
			t.Run(name, func(t *testing.T) {
				e := NewExecution()
				values := NewArrayList[*JavaString]()
				values.Add(JavaStringFromHostUTF8("a"))
				values.Add(JavaStringFromHostUTF8("b"))
				var d *ArrayDeque
				if setup.exact {
					d = NewArrayDequeFromCollection[*JavaString](values, e)
				} else {
					d = NewArrayDeque[*JavaString]()
					d.AddAll(values, e)
				}
				expectedCapacity := 17
				if setup.exact {
					expectedCapacity = 3
				}
				if len(d.elements) != expectedCapacity || d.Size() != 2 {
					t.Fatal("capacity-aligned native fixture setup mismatch")
				}
				calls := 0
				other := &arrayDequeBoundary224Contains{contains: func(got *Execution, value any) bool {
					if got != e {
						t.Fatal("contains lost original Execution")
					}
					calls++
					if calls == 1 {
						d.AddLast(JavaStringFromHostUTF8("side"))
					}
					return matching && calls == 1
				}}
				changed := false
				observed := arrayDequeBoundary224Capture(func() { changed = d.RemoveAll(other, e) })
				expectedException := TypeID("java.util.ConcurrentModificationException")
				expectedState := "[a, b, side]"
				expectedCalls := 2
				if matching && setup.exact {
					expectedException = "java.lang.ArrayIndexOutOfBoundsException"
					expectedCalls = 1
				}
				if matching && !setup.exact {
					expectedException = ""
					expectedState = "[b, side]"
					expectedCalls = 3
				}
				if expectedException != "" {
					if observed == nil || !ObjectInstanceOf(observed, expectedException) {
						t.Logf("observed %s callback outcome: %T", name, observed)
						t.Fatalf("capacity-aligned callback exception mismatch: %s", name)
					}
				} else if observed != nil || !changed {
					t.Logf("observed %s callback outcome: %T", name, observed)
					t.Fatalf("capacity-aligned callback success mismatch: %s", name)
				}
				if d.String() != expectedState || calls != expectedCalls {
					t.Fatalf("capacity-aligned callback state mismatch: %s", name)
				}
			})
		}
	}
	t.Run("canonical_string_control", func(t *testing.T) {
		d := NewArrayDeque[*JavaString]()
		d.AddLast(JavaStringFromHostUTF8("a"))
		if d.String() != "[a]" {
			t.Fatal("canonical Java string fixture is invalid")
		}
	})
	t.Run("raw_deque_interface_control", func(t *testing.T) {
		d := NewArrayDeque[*JavaString]()
		view := ObjectView[JavaDeque](d, DequeTypeID)
		if !JavaReferenceEqual(d, view) {
			t.Fatal("existing erased Deque alias lost identity")
		}
	})
}

func arrayDequeBoundary224Capture(action func()) (observed any) {
	defer func() { observed = recover() }()
	action()
	return
}

type arrayDequeBoundary224Contains struct{ contains func(*Execution, any) bool }

func (*arrayDequeBoundary224Contains) IteratorJava2goExecution(*Execution) JavaIterator { return nil }
func (c *arrayDequeBoundary224Contains) CollectionContainsJava2goExecution(e *Execution, v any) bool {
	return c.contains(e, v)
}
