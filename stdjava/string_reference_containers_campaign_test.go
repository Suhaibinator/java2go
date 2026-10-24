package stdjava

import (
	"reflect"
	"sync"
	"testing"
)

// These native-runtime tests exercise canonical pointers, not generated ABI
// parity. Their Java oracle is campaign/probes/canonical-string-containers.
func TestCampaignCanonicalStringContainerProtocols(t *testing.T) {
	a := NewJavaStringUTF16([]uint16{'x', 0xd800, 0, 0xdc00})
	b := CopyJavaString(a)
	m := NewMap[*JavaString, *JavaString]()
	m.Put(a, a)
	m.Put(b, b)
	if m.Size() != 1 || m.KeySet()[0] != a || m.Get(a) != b || m.Get(NewJavaStringUTF16([]uint16{'?'})) != nil {
		t.Fatal("HashMap content keys, first key identity, or missing null")
	}
	s := NewSet[*JavaString]()
	s.Add(a)
	s.Add(b)
	if s.Size() != 1 || JavaReferenceEqual(a, b) {
		t.Fatal("HashSet content equality or reference identity")
	}
	arr := NewReferenceArrayOf[*JavaString](2, StringTypeID)
	if ReferenceArrayGet[any](arr, 0, StringTypeID) != nil || ObjectView[*JavaString](nil, StringTypeID) != nil || ObjectView[any](nil, StringTypeID) != nil {
		t.Fatal("erased canonical String array null leaked native sentinel")
	}
	ReferenceArraySet(arr, 0, a)
	ReferenceArraySet[*JavaString](arr, 1, nil)
	if ReferenceArrayGet[*JavaString](arr, 0, StringTypeID) != a || ReferenceArrayGet[any](arr, 1, StringTypeID) != nil {
		t.Fatal("String array alias/null")
	}
	assertBoxedReferencePanic(t, "ArrayStoreException", func() { ReferenceArraySet(arr, 1, NewStringBuilder()) })
	assertBoxedReferencePanic(t, "ClassCastException", func() { ObjectView[*JavaString](NewStringBuilder(), StringTypeID) })
	pair, bmp := NewJavaStringUTF16([]uint16{0xd83d, 0xde00}), NewJavaStringUTF16([]uint16{0xe000})
	if got := NaturalOrder[*JavaString]()(pair, bmp); got != -1987 {
		t.Fatalf("UTF16 compare=%d", got)
	}
	assertBoxedReferencePanic(t, "ClassCastException", func() { ComparableCompareTo(a, NewStringBuilder()) })
	assertBoxedReferencePanic(t, "NullPointerException", func() { NaturalOrder[*JavaString]()(a, nil) })
	if CharSequenceLength(nil, a) != 4 || CharSequenceCharAt(nil, a, 1) != 0xd800 || CharSequenceCharAt(nil, a, 3) != 0xdc00 {
		t.Fatal("CharSequence UTF16 units")
	}
}

func TestCampaignCanonicalStringBuilderIngestion(t *testing.T) {
	a := NewJavaStringUTF16([]uint16{'x', 0xd800, 0, 0xdc00})
	b := NewStringBuilder().Append(a).Insert(1, CopyJavaString(a))
	first, second := b.ToJavaString(), b.ToJavaString()
	b.AppendChar('z')
	want := []uint16{'x', 'x', 0xd800, 0, 0xdc00, 0xd800, 0, 0xdc00}
	if !reflect.DeepEqual(first.UTF16Copy(), want) || first == second || !first.Equals(second) {
		t.Fatalf("builder=%x", first.UTF16Copy())
	}
}

func TestCampaignCanonicalStringConcurrentContentKeys(t *testing.T) {
	a := NewJavaStringUTF16([]uint16{'x', 0xd800, 0, 0xdc00})
	m := NewConcurrentHashMap[*JavaString, *JavaString]()
	m.Put(a, a)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 128; j++ {
				copy := CopyJavaString(a)
				if copy.HashCode() != a.HashCode() {
					t.Error("immutable concurrent hash")
				}
				m.Put(copy, copy)
			}
		}()
	}
	workers.Wait()
	if m.Size() != 1 || m.KeySet()[0] != a || !m.Get(CopyJavaString(a)).Equals(a) {
		t.Fatal("concurrent content key identity")
	}
}

func TestCampaignCanonicalStringJoinAndRegionAdapters(t *testing.T) {
	execution := NewExecution()
	a := NewJavaStringUTF16([]uint16{'x', 0xd800, 0, 0xdc00})
	b := CopyJavaString(a)
	separator := NewJavaStringUTF16([]uint16{0xdfff})
	want := []uint16{'x', 0xd800, 0, 0xdc00, 0xdfff, 'n', 'u', 'l', 'l', 0xdfff, 'x', 0xd800, 0, 0xdc00}
	array := ReferenceArrayLiteral(CharSequenceTypeID, a, nil, b)
	list := NewList[any]()
	list.Add(a)
	list.Add(nil)
	list.Add(b)
	results := []*JavaString{
		JavaStringJoinValuesExecution(execution, separator, a, nil, b),
		JavaStringJoinArrayExecution(execution, separator, array),
		JavaStringJoinIterableExecution(execution, separator, list),
	}
	for index, result := range results {
		if !reflect.DeepEqual(result.UTF16Copy(), want) {
			t.Fatalf("join %d=%x", index, result.UTF16Copy())
		}
	}
	stream := JavaStringStreamJoining(NewStream(a, (*JavaString)(nil), b), NewJavaStringUTF16([]uint16{'|'}), NewJavaStringUTF16([]uint16{'['}), NewJavaStringUTF16([]uint16{']'}))
	streamWant := []uint16{'[', 'x', 0xd800, 0, 0xdc00, '|', 'n', 'u', 'l', 'l', '|', 'x', 0xd800, 0, 0xdc00, ']'}
	if !reflect.DeepEqual(stream.UTF16Copy(), streamWant) {
		t.Fatalf("stream join=%x", stream.UTF16Copy())
	}
	if !JavaStringRegionMatches(a, false, 1, b, 1, 3) || !JavaStringRegionMatches(a, false, 0, b, 0, -1) {
		t.Fatal("regionMatch units/negative length")
	}
}
