package stdjava

// Semantic preservation tests; baseline is expected to pass.
import (
	"slices"
	"sync"
	"testing"
)

func TestUTF8OwnedOutput181FreshAndIndependent(t *testing.T) {
	for _, units := range [][]uint16{{}, {'A'}, {0, 0xe9, 0x65e5, 0xd83d, 0xde00, 0xd800, 'B', 0xdc00}} {
		value := NewJavaStringUTF16(units)
		first := JavaStringGetBytes(value, UTF_8)
		second := JavaStringGetBytes(value, UTF_8)
		if first == second || first.ComponentType() != PrimitiveByteTypeID || second.ComponentType() != PrimitiveByteTypeID {
			t.Fatal("fresh byte-array identity/type lost")
		}
		saved := append([]int8(nil), second.Elements...)
		if len(first.Elements) > 0 {
			first.Elements[0] ^= 127
		}
		for range 20 {
			JavaStringGetBytes(value, UTF_8)
		}
		if !slices.Equal(second.Elements, saved) || !slices.Equal(JavaStringGetBytes(value, UTF_8).Elements, saved) {
			t.Fatal("encoding results share mutable storage")
		}
		if !slices.Equal(value.UTF16Copy(), units) {
			t.Fatal("encoding modified immutable input")
		}
	}
}
func TestUTF8OwnedOutput181ConcurrentImmutableInput(t *testing.T) {
	value := NewJavaStringUTF16([]uint16{0, 'A', 0xe9, 0x65e5, 0xd83d, 0xde00, 0xd800, 'B', 0xdc00})
	want := JavaStringGetBytes(value, UTF_8).Elements
	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 32 {
				result := JavaStringGetBytes(value, UTF_8)
				if !slices.Equal(result.Elements, want) {
					t.Error("concurrent independent encoding changed contents")
				}
				result.Elements[0] = 127
			}
		}()
	}
	group.Wait()
	if !slices.Equal(JavaStringGetBytes(value, UTF_8).Elements, want) {
		t.Fatal("output mutation reached shared input/cache")
	}
}
