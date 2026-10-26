package stdjava

import "testing"

// This guard uses only APIs already present in the published runtime core.
// The canonical names and declaration flags are checked by a fresh JDK oracle
// before this test's public-core RED is accepted.
func TestThreadLocalMetadataExistingAPIBoundary(t *testing.T) {
	const root TypeID = "java.lang.ThreadLocal"
	const suppliedID TypeID = "java.lang.ThreadLocal$SuppliedThreadLocal"
	execution := NewExecution()
	calls, selections := 0, 0
	marker := JavaStringFromHostUTF8("metadata-reference")
	choose := func() any {
		selections++
		return NewSupplierFuncAdapter(func(got *Execution) *JavaString {
			if got != execution || ThreadCurrentThread(got) != ThreadCurrentThread(execution) {
				t.Fatal("metadata changed consuming execution/thread")
			}
			calls++
			return marker
		})
	}
	local := ThreadLocalWithInitialReference(choose())
	t.Run("canonical_allocation_descriptor", func(t *testing.T) {
		id, known := ObjectDynamicType(local)
		if !known || id != suppliedID {
			t.Errorf("supplied allocation dynamic type=%q known=%v, want %q", id, known, suppliedID)
		}
		class := ObjectGetClass(local)
		if class.GetName() != string(suppliedID) || class != ClassLiteral(suppliedID) {
			t.Errorf("supplied allocation class=%q, want canonical supplied class", class.GetName())
		}
		if calls != 0 || selections != 1 {
			t.Fatal("metadata observation entered supplier or repeated selection")
		}
	})
	t.Run("canonical_root_cast_and_array", func(t *testing.T) {
		if !ObjectInstanceOf(local, root) || !ObjectInstanceOf(local, ObjectTypeID) {
			t.Error("supplied allocation lacks canonical ThreadLocal/Object ancestry")
		}
		var alias *ThreadLocal[any]
		failure := threadLocalReferenceObservedPanic(func() { alias = ReferenceCheckCast[*ThreadLocal[any]](local, root) })
		if failure != nil {
			t.Errorf("canonical ThreadLocal root cast failed: %T %v", failure, failure)
			return
		}
		array := NewReferenceArrayOf[*ThreadLocal[any]](1, root)
		ReferenceArrayAssign[*ThreadLocal[any]](array, 0, alias, root)
		if alias != local || ReferenceArrayGet[*ThreadLocal[any]](array, 0, root) != local || !JavaReferenceEqual(alias, local) {
			t.Fatal("canonical cast/array lost allocation identity")
		}
		if calls != 0 || selections != 1 {
			t.Fatal("cast/array observation entered supplier or repeated selection")
		}
	})
	t.Run("supplied_declaration_modifiers", func(t *testing.T) {
		if got := ObjectGetClass(local).GetModifiers(); got != 24 {
			t.Errorf("SuppliedThreadLocal modifiers=%d, want static|final=24", got)
		}
		if calls != 0 || selections != 1 {
			t.Fatal("modifier observation entered supplier or repeated selection")
		}
	})
	t.Run("lazy_cache_class_stability", func(t *testing.T) {
		class := ObjectGetClass(local)
		first, second := local.GetObject(execution), local.GetObject(execution)
		if first != marker || second != marker || calls != 1 {
			t.Fatal("metadata changed lazy cache publication/identity")
		}
		local.SetObject(execution, nil)
		if local.GetObject(execution) != nil || calls != 1 {
			t.Fatal("metadata lost cached null or invoked supplier")
		}
		local.Remove(execution)
		if local.GetObject(execution) != marker || calls != 2 || ObjectGetClass(local) != class {
			t.Fatal("metadata changed remove/retry/class identity")
		}
	})
	t.Run("legacy_native_metadata", func(t *testing.T) {
		legacy := NewThreadLocal[int32]()
		id, known := ObjectDynamicType(legacy)
		if !known || id != "ThreadLocal" || ObjectGetClass(legacy).GetName() != "ThreadLocal" || ObjectGetClass(legacy).GetModifiers() != 1 || ObjectInstanceOf(legacy, root) {
			t.Fatal("canonical service metadata changed legacy native Go descriptor")
		}
		legacy.Set(execution, 7)
		if legacy.Get(execution) != 7 {
			t.Fatal("metadata changed legacy typed storage")
		}
		legacy.Remove(execution)
		if legacy.Get(execution) != 0 {
			t.Fatal("metadata changed legacy remove/default")
		}
	})
	t.Run("null_supplier_validation", func(t *testing.T) {
		nullSelections := 0
		chooseNull := func() any { nullSelections++; return nil }
		failure := threadLocalReferenceObservedPanic(func() { ThreadLocalWithInitialReference(chooseNull()) })
		if !CaughtAs(failure, "NullPointerException") || nullSelections != 1 || calls != 2 {
			t.Fatal("metadata moved null validation or entered initializer")
		}
	})
}
