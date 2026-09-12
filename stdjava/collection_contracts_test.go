package stdjava

import (
	"sync"
	"testing"
)

type contractKey struct {
	id        int32
	label     string
	execution *Execution
}

func (k *contractKey) Equals(other any) bool {
	value, ok := other.(*contractKey)
	return ok && value != nil && k.id == value.id
}
func (k *contractKey) HashCode() int32 { return 7 }
func (k *contractKey) EqualsJava2goExecution1(execution *Execution, other any) bool {
	k.execution = execution
	return k.Equals(other)
}
func (k *contractKey) HashCodeJava2goExecution1(execution *Execution) int32 {
	k.execution = execution
	return k.HashCode()
}

func TestCollectionContractsCollisionAndExecution(t *testing.T) {
	execution := &Execution{}
	first, equal, collision := &contractKey{id: 1, label: "first"}, &contractKey{id: 1, label: "equal"}, &contractKey{id: 2}
	m := NewMap[*contractKey, string]()
	if !StringIsNull(m.Put(first, "old", execution)) {
		t.Fatal("new key must return Java null")
	}
	if m.Put(equal, "new", execution) != "old" || equal.execution != execution {
		t.Fatal("virtual hash/equals or previous value")
	}
	m.Put(collision, "collision", execution)
	if m.Size() != 2 || m.KeySet()[0] != first || m.Get(equal, execution) != "new" {
		t.Fatal("collision lookup or key identity")
	}
	if m.Remove(equal, execution) != "new" || m.Get(collision, execution) != "collision" {
		t.Fatal("collision removal")
	}
	if !StringIsNull(m.Get(first, execution)) || m.ContainsKey(first, execution) {
		t.Fatal("absent lookup")
	}
}

// A generated or external object can have a Go representation that is not
// comparable while still implementing a lawful Java equals/hashCode contract.
type nonComparableKey struct{ ids []int32 }

func (k nonComparableKey) Equals(other any) bool {
	value, ok := other.(nonComparableKey)
	return ok && len(k.ids) == len(value.ids) && k.ids[0] == value.ids[0]
}
func (k nonComparableKey) HashCode() int32 { return k.ids[0] }
func TestCollectionContractsNonComparableKeysAndArrays(t *testing.T) {
	m := NewMap[nonComparableKey, int32]()
	m.Put(nonComparableKey{[]int32{1}}, 2)
	if m.Get(nonComparableKey{[]int32{1}}) != 2 {
		t.Fatal("non-comparable Java key")
	}
	arrays := NewSet[any]()
	a, b := NewPrimitiveArray[int32](1, PrimitiveIntTypeID), NewPrimitiveArray[int32](1, PrimitiveIntTypeID)
	if !arrays.Add(a) || !arrays.Add(b) || arrays.Add(a) || ObjectsEqual(a, b) {
		t.Fatal("arrays require identity")
	}
}

func TestCollectionContractsNestedValues(t *testing.T) {
	left := NewListFrom[any](BoxInteger(1), nil)
	right := NewListFrom[*Integer](NewInteger(1), nil)
	if !ObjectsEqual(left, right) || left.HashCode() != right.HashCode() {
		t.Fatal("erased list equality/hash")
	}
	m := NewMap[*List[any], string]()
	m.Put(left, "nested")
	if m.Get(right) != "nested" {
		t.Fatal("nested collection key")
	}
	a, b := NewMap[string, any](), NewMap[string, *Integer]()
	a.Put("x", nil)
	b.Put("y", nil)
	if a.Equals(b) {
		t.Fatal("null value is distinct from absent key")
	}
	b.Clear()
	b.Put("x", nil)
	if !a.Equals(b) || a.HashCode() != b.HashCode() {
		t.Fatal("map structural contract")
	}
}

type asymmetricKey struct{ accepts bool }

func (k *asymmetricKey) Equals(other any) bool {
	_, ok := other.(*asymmetricKey)
	return ok && k.accepts
}
func (k *asymmetricKey) HashCode() int32 { return 1 }
func TestCollectionContractsLookupEqualityDirection(t *testing.T) {
	stored, query := &asymmetricKey{}, &asymmetricKey{accepts: true}
	m := NewMap[*asymmetricKey, int32]()
	m.Put(stored, 1)
	if m.Get(query) != 1 || !NewListFrom(stored).Contains(query) {
		t.Fatal("query.equals(stored) must determine lookup")
	}
}

func TestCollectionContractsSortedComparisonEquality(t *testing.T) {
	compare := Comparator[*contractKey](func(a, b *contractKey) int32 { return a.id/10 - b.id/10 })
	tree := NewTreeMapWith[*contractKey, string](compare)
	first := &contractKey{id: 21}
	tree.Put(first, "old")
	tree.Put(&contractKey{id: 11}, "first")
	if tree.Put(&contractKey{id: 29}, "new") != "old" || tree.Size() != 2 || tree.KeySet()[1] != first {
		t.Fatal("tree comparison equality/key retention")
	}
	if tree.Remove(&contractKey{id: 22}) != "new" || tree.Size() != 1 {
		t.Fatal("tree comparison removal")
	}
}

type reentrantCollectionKey struct {
	id   int32
	read func()
}

func (k *reentrantCollectionKey) HashCode() int32 {
	if k.read != nil {
		k.read()
	}
	return 1
}
func (k *reentrantCollectionKey) Equals(other any) bool {
	if k.read != nil {
		k.read()
	}
	value, ok := other.(*reentrantCollectionKey)
	return ok && k.id == value.id
}
func TestCollectionContractsConcurrentCallbacksAndRaces(t *testing.T) {
	m := NewConcurrentHashMap[*reentrantCollectionKey, int32]()
	first := &reentrantCollectionKey{id: 1}
	m.Put(first, 1)
	query := &reentrantCollectionKey{id: 1, read: func() { m.Size(); m.Get(first) }}
	if m.Put(query, 2) != 1 || m.Get(query) != 2 {
		t.Fatal("reentrant equals/hash lookup")
	}
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func(id int32) {
			defer workers.Done()
			for n := 0; n < 50; n++ {
				key := &reentrantCollectionKey{id: id}
				m.Put(key, id)
				m.Get(key)
				m.ContainsKey(key)
				m.KeySet()
				m.Remove(key)
			}
		}(int32(i + 10))
	}
	workers.Wait()
	if m.Size() != 1 || m.Remove(query) != 2 {
		t.Fatal("concurrent collision updates lost entries")
	}
}

type countedEquality struct{ calls int }

func (k *countedEquality) Equals(any) bool { k.calls++; return false }
func (k *countedEquality) HashCode() int32 { return 7 }
func TestCollectionContractsIdentityFastPathAndListCalls(t *testing.T) {
	value := &countedEquality{}
	m := NewMap[*countedEquality, int32]()
	m.Put(value, 1)
	if !ObjectsEqual(value, value) || m.Get(value) != 1 || value.calls != 0 {
		t.Fatal("Objects.equals/hash lookup must short circuit identity")
	}
	list := NewListFrom(value)
	if list.Contains(value) || value.calls != 1 {
		t.Fatal("List.contains must invoke query.equals even on identity")
	}
	if !list.Equals(list) || value.calls != 1 {
		t.Fatal("List.equals itself must short circuit identity")
	}
}

type overloadedCollectionEquality struct{ id int32 }

func (k *overloadedCollectionEquality) EqualsJava2goExecution(_ *Execution, other *overloadedCollectionEquality) bool {
	return k.id == other.id
}
func TestCollectionContractsIgnoreTypedEqualsOverload(t *testing.T) {
	left, right := &overloadedCollectionEquality{1}, &overloadedCollectionEquality{1}
	if ObjectsEqual(left, right) || NewListFrom(left).Contains(right) {
		t.Fatal("equals(T) overload is not equals(Object)")
	}
}

type collectionBaseView struct{ *ObjectInfo }
type collectionDerivedView struct {
	collectionBaseView
	id int32
}

func (v *collectionDerivedView) Equals(other any) bool {
	right, ok := other.(*collectionDerivedView)
	return ok && v.id == right.id
}
func (v *collectionDerivedView) HashCode() int32 { return v.id }
func newCollectionDerivedView(id int32) *collectionDerivedView {
	value := &collectionDerivedView{id: id}
	value.ObjectInfo = NewObjectInfo("CollectionDerivedView", func(TypeID) any { return value })
	return value
}
func TestCollectionContractsSuperclassViewsDispatchDynamicOverrides(t *testing.T) {
	first, equal := newCollectionDerivedView(7), newCollectionDerivedView(7)
	m := NewMap[any, int32]()
	m.Put(&first.collectionBaseView, 1)
	if m.Get(equal) != 1 || m.Put(&equal.collectionBaseView, 2) != 1 || m.Size() != 1 {
		t.Fatal("base view must invoke derived equality/hash")
	}
	if m.KeySet()[0] != &first.collectionBaseView {
		t.Fatal("retain original key view")
	}
}

func TestCollectionContractsEqualsNullStillInvokesNonNullReceiver(t *testing.T) {
	value := &countedEquality{}
	if ObjectsEqual(value, (*Integer)(nil)) || value.calls != 1 {
		t.Fatal("non-null receiver must observe equals(null)")
	}
	if ObjectsEqual(nil, value) || value.calls != 1 {
		t.Fatal("null query must not call equals")
	}
}

func TestCollectionContractsConcurrentMapValueInteroperability(t *testing.T) {
	concurrent := NewConcurrentHashMap[string, *Integer]()
	ordinary := NewMap[string, any]()
	concurrent.Put("x", NewInteger(1))
	ordinary.Put("x", BoxInteger(1))
	if !concurrent.Equals(ordinary) || !ordinary.Equals(concurrent) || concurrent.HashCode() != ordinary.HashCode() {
		t.Fatal("all map implementations share Java structural equality/hash")
	}
	outer := NewMap[any, string]()
	outer.Put(concurrent, "nested")
	if outer.Get(ordinary) != "nested" {
		t.Fatal("ConcurrentHashMap as map-valued key")
	}
}
