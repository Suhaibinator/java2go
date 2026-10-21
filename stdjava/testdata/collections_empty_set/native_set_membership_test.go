package emptysetmembership_test

import (
	j "github.com/NickyBoy89/java2go/stdjava"
	"testing"
)

type nativeSetMembershipLookalike struct{}

func (*nativeSetMembershipLookalike) IteratorJava2goExecution(*j.Execution) j.JavaIterator {
	return nil
}
func (*nativeSetMembershipLookalike) Size() int32   { return 0 }
func (*nativeSetMembershipLookalike) IsEmpty() bool { return true }
func nativeSetMembershipPanic(t *testing.T, name string, action func()) {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); action() }()
	if failure == nil || !j.CaughtAs(failure, name) {
		t.Fatalf("want %s got %T %v", name, failure, failure)
	}
}
func TestCollectionsEmptySetNativeSetMembershipPrerequisite(t *testing.T) {
	for _, set := range []*j.Set[any]{j.NewSet[any](), j.NewTreeSet[any]()} {
		for _, id := range []j.TypeID{j.SetTypeID, j.CollectionTypeID, j.IterableTypeID, j.ObjectTypeID} {
			if !j.ObjectInstanceOf(set, id) {
				t.Fatalf("runtime Set missing %s nominal membership", id)
			}
			if !j.JavaReferenceEqual(j.ObjectView[j.JavaIterable](set, id), set) {
				t.Fatalf("Set nominal view changed identity: %s", id)
			}
		}
		for _, id := range []j.TypeID{j.MapTypeID, "java.util.List", j.SerializableTypeID, "java.util.HashSet", "java.util.TreeSet"} {
			if j.ObjectInstanceOf(set, id) {
				t.Fatalf("backend invented unrelated/concrete descriptor %s", id)
			}
		}
		objects := j.NewReferenceArray(1, j.ObjectTypeID)
		sets := j.NewReferenceArray(1, j.SetTypeID)
		j.ReferenceArraySet(objects, 0, set)
		j.ReferenceArraySet(sets, 0, set)
		if !j.JavaReferenceEqual(j.ReferenceArrayGet[j.JavaIterable](objects, 0, j.SetTypeID), set) || !j.JavaReferenceEqual(j.ReferenceArrayGet[j.JavaIterable](sets, 0, j.SetTypeID), set) {
			t.Fatal("Object[]/Set[] identity")
		}
		lists := j.NewReferenceArray(1, j.TypeID("java.util.List"))
		nativeSetMembershipPanic(t, "ArrayStoreException", func() { j.ReferenceArraySet(lists, 0, set) })
	}
	fake := &nativeSetMembershipLookalike{}
	if j.ObjectInstanceOf(fake, j.SetTypeID) || j.ObjectInstanceOf(fake, j.CollectionTypeID) {
		t.Fatal("public shape admitted as nominal collection")
	}
	nativeSetMembershipPanic(t, "ClassCastException", func() { j.ObjectView[j.JavaIterable](fake, j.SetTypeID) })
	array := j.NewReferenceArray(1, j.SetTypeID)
	nativeSetMembershipPanic(t, "ArrayStoreException", func() { j.ReferenceArraySet(array, 0, fake) })
	var absent *j.Set[any]
	if j.ObjectInstanceOf(absent, j.SetTypeID) || j.ObjectView[j.JavaIterable](absent, j.SetTypeID) != nil {
		t.Fatal("typed null gained membership/view")
	}
}
