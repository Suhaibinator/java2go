package stdjava

import "testing"

// Public collection-shaped methods alone must never grant nominal List membership.
type listProjectionImpostorSol61 struct{}
func (*listProjectionImpostorSol61) Size() int32 { return 0 }
func (*listProjectionImpostorSol61) Get(int32) string { return "wrong" }
func (*listProjectionImpostorSol61) Add(string) bool { return true }

func listProjectionExpectFailureSol61(t *testing.T, kind string, work func()) {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); work() }()
	if !CaughtAs(failure, kind) { t.Fatalf("failure = %v, want %s", failure, kind) }
}

func TestListReferenceProjectionNominalSol61(t *testing.T) {
	t.Run("list-pointer-and-mutations", func(t *testing.T) {
		list := NewListFrom("one")
		view := ObjectView[*List[string]](list, nativeListTypeID)
		if view != list { t.Fatal("projection changed Java reference identity") }
		view.Add("two")
		if list.Size() != 2 || list.Get(1) != "two" { t.Fatal("projection lost shared mutable state") }
	})
	t.Run("interface-ancestors", func(t *testing.T) {
		list := NewListFrom("one")
		for _, descriptor := range []TypeID{nativeCollectionTypeID, IterableTypeID, ObjectTypeID} {
			if ObjectView[any](list, descriptor) != list { t.Fatalf("%s projection changed identity", descriptor) }
		}
	})
	t.Run("typed-null", func(t *testing.T) {
		var list *List[string]
		if ObjectView[*List[string]](list, nativeListTypeID) != nil || ObjectView[any](list, nativeListTypeID) != nil { t.Fatal("typed null gained identity") }
	})
	t.Run("unqualified-descriptor-rejected", func(t *testing.T) {
		listProjectionExpectFailureSol61(t, "ClassCastException", func() { ObjectView[any](NewList[string](), TypeID("List")) })
	})
	t.Run("source-name-not-an-alias", func(t *testing.T) {
		listProjectionExpectFailureSol61(t, "ClassCastException", func() { ObjectView[any](NewList[string](), TypeID("source.List")) })
	})
	t.Run("structural-impostor-rejected", func(t *testing.T) {
		listProjectionExpectFailureSol61(t, "ClassCastException", func() { ObjectView[any](&listProjectionImpostorSol61{}, nativeListTypeID) })
	})
}

func TestListReferenceArrayNominalGuardsSol61(t *testing.T) {
	t.Run("list-array-preserves-pointer", func(t *testing.T) {
		list := NewListFrom("one")
		array := NewReferenceArray(1, nativeListTypeID)
		ReferenceArraySet(array, 0, list)
		view := ReferenceArrayGet[*List[string]](array, 0, nativeListTypeID)
		if view != list { t.Fatal("array read copied List") }
		view.Add("two")
		if list.Size() != 2 { t.Fatal("array read lost shared state") }
	})
	t.Run("impostor-array-store-rejected", func(t *testing.T) {
		array := NewReferenceArray(1, nativeListTypeID)
		listProjectionExpectFailureSol61(t, "ArrayStoreException", func() { ReferenceArraySet(array, 0, &listProjectionImpostorSol61{}) })
	})
	t.Run("nominal-array-components-stay-distinct", func(t *testing.T) {
		if JavaTypeAssignable(ArrayTypeID(TypeID("List")), ArrayTypeID(nativeListTypeID)) || JavaTypeAssignable(ArrayTypeID(TypeID("source.List")), ArrayTypeID(nativeListTypeID)) { t.Fatal("source/unqualified array became a native List array") }
		if JavaTypeAssignable(ArrayTypeID(PrimitiveIntTypeID), ArrayTypeID(PrimitiveLongTypeID)) { t.Fatal("primitive arrays became covariant") }
	})
}
