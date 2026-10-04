package stdjava

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
	"unsafe"
	"weak"
)

// Prepared before implementation. This new core is independent of the old
// generated String ABI; integration is a separate coordinated migration.
func TestCampaignStringCoreIdentityAndUnits(t *testing.T) {
	input := []uint16{'A', 0xD800, 0, 0xDC00, 0xD83D, 0xDE00}
	first := NewJavaStringUTF16(input)
	if unsafe.Sizeof(*first) == 0 {
		t.Fatal("String wrapper has zero size")
	}
	second := CopyJavaString(first)
	alias := first
	if first == second || first != alias {
		t.Fatal("copy/allocation identity violated")
	}
	if !first.Equals(second) || first.HashCode() != second.HashCode() {
		t.Fatal("equal copies lost content equality/hash")
	}
	if first.Length() != 6 {
		t.Fatalf("length=%d, want six UTF16 units", first.Length())
	}
	want := []uint16{'A', 0xD800, 0, 0xDC00, 0xD83D, 0xDE00}
	input[0] = 'Z'
	exported := first.UTF16Copy()
	if !reflect.DeepEqual(exported, want) {
		t.Fatalf("constructor did not copy: %x", exported)
	}
	exported[1] = 'x'
	if !reflect.DeepEqual(first.UTF16Copy(), want) || !reflect.DeepEqual(second.UTF16Copy(), want) {
		t.Fatal("mutable UTF16 view escaped")
	}
	for index, unit := range want {
		if got := first.CharAt(int32(index)); got != rune(unit) {
			t.Fatalf("charAt(%d)=%x want %x", index, got, unit)
		}
	}
	empty1, empty2 := NewJavaStringUTF16(nil), NewJavaStringUTF16([]uint16{})
	if empty1 == empty2 || !empty1.Equals(empty2) || empty1.Length() != 0 {
		t.Fatal("empty allocation/content contract")
	}
	if first.Equals(nil) || first.Equals("native Go text") {
		t.Fatal("non-JavaString admitted by equals")
	}
	var absent *JavaString
	if !JavaReferenceEqual(absent, nil) || JavaReferenceEqual(first, second) || !JavaReferenceEqual(any(first), alias) {
		t.Fatal("cross-view/null reference identity")
	}
	assertBoxedReferencePanic(t, "NullPointerException", func() { CopyJavaString(absent) })
	assertBoxedReferencePanic(t, "NullPointerException", func() { absent.Length() })
	assertBoxedReferencePanic(t, "NullPointerException", func() { first.CompareTo(absent) })
	assertBoxedReferencePanic(t, "StringIndexOutOfBoundsException", func() { first.CharAt(-1) })
	assertBoxedReferencePanic(t, "StringIndexOutOfBoundsException", func() { first.CharAt(first.Length()) })
	assertBoxedReferencePanic(t, "StringIndexOutOfBoundsException", func() { empty1.CharAt(0) })
}

func TestCampaignStringCoreContentHashAndOrder(t *testing.T) {
	abc := NewJavaStringUTF16([]uint16{'a', 'b', 'c'})
	if abc.HashCode() != 96354 {
		t.Fatalf("abc hash=%d, want 96354", abc.HashCode())
	}
	if NewJavaStringUTF16(nil).HashCode() != 0 || NewJavaStringUTF16([]uint16{0}).HashCode() != 0 {
		t.Fatal("empty/NUL content hash")
	}
	cases := []struct {
		left, right []uint16
		difference  int32
	}{
		{[]uint16{'a'}, []uint16{'z'}, -25},
		{[]uint16{'a', 'b'}, []uint16{'a', 'b', 'c', 'd', 'e', 'f'}, -4},
		{[]uint16{0xD83D, 0xDE00}, []uint16{0xE000}, -1987},
		{[]uint16{0xD800}, []uint16{0xD801}, -1},
	}
	for _, c := range cases {
		left, right := NewJavaStringUTF16(c.left), NewJavaStringUTF16(c.right)
		if got := left.CompareTo(right); got != c.difference {
			t.Fatalf("compare %x/%x=%d want %d", c.left, c.right, got, c.difference)
		}
	}
}

func TestCampaignStringCorePoolCollisionAndLiteralRoots(t *testing.T) {
	// Force every content into one bucket: equality must inspect complete units.
	pool := newStringInternPool(func(*JavaString) uint64 { return 7 })
	first := NewJavaStringUTF16([]uint16{0xD800, 'a'})
	unequal := NewJavaStringUTF16([]uint16{0xD800, 'b'})
	if pool.intern(first) != first || pool.intern(unequal) != unequal {
		t.Fatal("distinct content collapsed by hash collision")
	}
	if pool.intern(CopyJavaString(first)) != first {
		t.Fatal("equal content did not canonicalize")
	}
	if got := pool.literal(first.UTF16Copy()); got != first {
		t.Fatal("literal did not adopt existing live dynamic representative")
	}
	w := func() weak.Pointer[JavaString] {
		literal := pool.literal([]uint16{'r', 'o', 'o', 't'})
		return weak.Make(literal)
	}()
	runtime.GC()
	rooted := w.Value()
	if rooted == nil || pool.intern(NewJavaStringUTF16([]uint16{'r', 'o', 'o', 't'})) != rooted {
		t.Fatal("resolved literal not strongly rooted")
	}
	runtime.KeepAlive(pool)
	runtime.KeepAlive(first)
	runtime.KeepAlive(unequal)
}

func TestCampaignStringCoreConcurrentIntern(t *testing.T) {
	pool := newStringInternPool(func(*JavaString) uint64 { return 9 })
	const count = 32
	inputs := make([]*JavaString, count)
	results := make([]*JavaString, count)
	for i := range inputs {
		inputs[i] = NewJavaStringUTF16([]uint16{'s', 0xDFFF, 0})
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := range inputs {
		workers.Add(1)
		go func(i int) { defer workers.Done(); <-start; results[i] = pool.intern(inputs[i]) }(i)
	}
	close(start)
	workers.Wait()
	canonical := results[0]
	found := false
	for _, input := range inputs {
		found = found || input == canonical
	}
	if !found {
		t.Fatal("intern fabricated a representative instead of selecting a receiver")
	}
	for _, result := range results {
		if result != canonical {
			t.Fatal("concurrent equal content received distinct representatives")
		}
	}
	runtime.KeepAlive(inputs)
}

func TestCampaignStringCoreDelayedCleanupReplacement(t *testing.T) {
	pool := newStringInternPool(func(*JavaString) uint64 { return 11 })
	original := NewJavaStringUTF16([]uint16{'k'})
	pool.intern(original)
	originalWeak := weak.Make(original)
	// Deterministically simulate retirement/replacement without relying on GC
	// timing. This helper removes only the recorded weak identity. Calling it
	// here is test setup, not permission to evict live interned strings in Java.
	pool.removeWeak(11, originalWeak)
	replacement := CopyJavaString(original)
	pool.intern(replacement)
	other := NewJavaStringUTF16([]uint16{'x'})
	pool.intern(other)
	// A delayed/repeated notification for the old entry must not remove either
	// the equal-content replacement or the unequal colliding entry.
	pool.removeWeak(11, originalWeak)
	if pool.intern(CopyJavaString(original)) != replacement || pool.intern(CopyJavaString(other)) != other {
		t.Fatal("stale cleanup deleted replacement or colliding entry")
	}
	runtime.KeepAlive(original)
	runtime.KeepAlive(replacement)
	runtime.KeepAlive(other)
}

// These are nominal facts about this new object, not an assertion that existing
// generated String arrays or all CharSequence call adapters have migrated.
func TestCampaignStringCoreDynamicDescriptor(t *testing.T) {
	value := NewJavaStringUTF16([]uint16{'x'})
	if got, ok := ObjectDynamicType(value); !ok || got != StringTypeID {
		t.Fatalf("dynamic type=(%s,%t), want canonical String", got, ok)
	}
	for _, target := range []TypeID{ObjectTypeID, StringTypeID, SerializableTypeID, ComparableTypeID, CharSequenceTypeID} {
		if !ObjectInstanceOf(value, target) {
			t.Fatalf("String missing nominal edge %s", target)
		}
	}
	var absent *JavaString
	if _, ok := ObjectDynamicType(absent); ok || ObjectInstanceOf(absent, StringTypeID) {
		t.Fatal("null acquired a dynamic String type")
	}
}

// Diagnostic only: Go does not guarantee prompt weak clearing or cleanup.
// Keep the pool alive while dropping all dynamic representatives, and log what
// this runtime observes. Passing does not assert a Java GC timing contract.
func TestCampaignStringCoreLifetimeObservation(t *testing.T) {
	pool := newStringInternPool(nil)
	observed := func() []weak.Pointer[JavaString] {
		strong := make([]*JavaString, 8)
		weakRefs := make([]weak.Pointer[JavaString], len(strong))
		for i := range strong {
			units := make([]uint16, 65536)
			units[0] = uint16(i)
			strong[i] = pool.intern(NewJavaStringUTF16(units))
			weakRefs[i] = weak.Make(strong[i])
		}
		runtime.KeepAlive(strong)
		return weakRefs
	}()
	for attempt := 0; attempt < 3; attempt++ {
		runtime.GC()
		live := 0
		for _, reference := range observed {
			if reference.Value() != nil {
				live++
			}
		}
		pool.index.mu.Lock()
		buckets := len(pool.index.buckets)
		pool.index.mu.Unlock()
		t.Logf("after GC %d: live dynamic wrappers=%d/8, weak-index buckets=%d", attempt+1, live, buckets)
	}
	runtime.KeepAlive(pool)
}
