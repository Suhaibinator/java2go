package stdjava

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestNIOHeapDigestConsumesOnlyViewWindow(t *testing.T) {
	array := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(91), int8(92), int8(10), int8(20), int8(30), int8(40), int8(50), int8(93))
	original := ByteBufferWrap(array, 2, 5)
	view := original.Slice().SetPosition(1).SetLimit(4)
	digest := JavaMessageDigestGetInstance(JavaStringLiteralUTF16([]uint16{'S', 'H', 'A', '-', '2', '5', '6'}))
	digest.UpdateReference(view)
	got := digest.Digest()
	want := sha256.Sum256([]byte{20, 30, 40})
	if len(got.Elements) != len(want) {
		t.Fatalf("digest length = %d", len(got.Elements))
	}
	for index, value := range want {
		if got.Elements[index] != int8(value) {
			t.Fatalf("digest byte %d = %d, want %d", index, got.Elements[index], int8(value))
		}
	}
	if view.Position() != 4 || view.Limit() != 4 || original.Position() != 2 || original.Limit() != 7 {
		t.Fatalf("consumer changed wrong cursor/window: view=%d/%d original=%d/%d", view.Position(), view.Limit(), original.Position(), original.Limit())
	}
	if view.Array() != array || !slices.Equal(array.Elements, []int8{91, 92, 10, 20, 30, 40, 50, 93}) {
		t.Fatalf("digest mutated/replaced backing array: %v", array.Elements)
	}
}

func TestNIOHeapFileChannelWritesOnlyViewWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view-window.bin")
	if err := os.WriteFile(path, []byte{11, 22, 33, 44, 55}, 0600); err != nil {
		t.Fatal(err)
	}
	random := NewRandomAccessFile(NewJavaFile(path), "r")
	defer random.Close()
	channel := random.GetChannel()
	random.SeekPosition(1)
	array := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(80), int8(81), int8(82), int8(83), int8(84), int8(85), int8(86), int8(87), int8(88))
	original := ByteBufferWrap(array, 2, 5)
	view := original.Slice().SetPosition(1).SetLimit(4)
	if count := channel.Read(view); count != 3 {
		t.Fatalf("channel read count = %d", count)
	}
	if !slices.Equal(array.Elements, []int8{80, 81, 82, 22, 33, 44, 86, 87, 88}) {
		t.Fatalf("channel wrote outside view window: %v", array.Elements)
	}
	if view.Position() != 4 || original.Position() != 2 || random.GetFilePointer() != 4 {
		t.Fatalf("wrong view/source/file cursor: %d/%d/%d", view.Position(), original.Position(), random.GetFilePointer())
	}
	view.Clear().SetLimit(1)
	random.SeekPosition(0)
	if view.Capacity() != 5 || view.Position() != 0 || channel.Read(view) != 1 {
		t.Fatal("clear did not retain view capacity and offset")
	}
	view.Flip()
	output := NewPrimitiveArray[int8](1, PrimitiveByteTypeID)
	view.GetInto(output)
	if output.Elements[0] != 11 || array.Elements[0] != 80 || array.Elements[1] != 81 || array.Elements[2] != 11 {
		t.Fatalf("view clear/read/flip/get crossed backing prefix: output=%v backing=%v", output.Elements, array.Elements)
	}
}
