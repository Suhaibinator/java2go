//go:build (darwin || linux) && f1_os_profile

package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFileChannelWriteRelativeAndPositionalActualPartial(t *testing.T) {
	path := os.Getenv("WRITE_PARTIAL_FILE")
	if path == "" {
		t.Fatal("explicit bounded file-size profile required")
	}
	var out strings.Builder
	for _, relative := range []bool{false, true} {
		if err := os.WriteFile(path, []byte{99}, 0600); err != nil {
			t.Fatal(err)
		}
		e := NewExecution()
		r := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(path), JavaStringFromHostUTF8("rw"))
		c := r.GetChannel()
		cursor := int64(7)
		if relative {
			cursor = 4093
		}
		r.SeekPosition(cursor)
		root := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19)).SetPosition(2).SetLimit(9)
		src := root.Slice().AsReadOnlyBuffer().SetPosition(1).SetLimit(6).Mark()
		var count int32
		if relative {
			count = FileChannelWriteVirtualExecution(e, c, src)
		} else {
			count = FileChannelWriteAtVirtualExecution(e, c, src, 4093)
		}
		cursor = r.GetFilePointer()
		text := func(position int64, count int) string {
			data := make([]byte, count)
			if n, err := r.state.file.ReadAt(data, position); err != nil || n != count {
				t.Fatalf("readback %d/%v", n, err)
			}
			result := ""
			for i, b := range data {
				if i > 0 {
					result += ","
				}
				result += fmt.Sprint(b)
			}
			return result
		}
		fmt.Fprintf(&out, "partial:%t|count=%d|src=%d:%d|cursor=%d|length=%d|prefix=%s|tail=%s\n", relative, count, src.position, src.limit, cursor, r.Length(), text(0, 3), text(4092, 4))
		result := writeMatrixCall(t, e, c, src, relative, 4096)
		fmt.Fprintf(&out, "full:%t|%s|src=%d|cursor=%d|length=%d\n", relative, strings.TrimPrefix(result, "throw:"), src.position, r.GetFilePointer(), r.Length())
		if src.mark != 1 || root.position != 2 {
			t.Fatal("partial changed mark or owner cursor")
		}
		r.Close()
	}
	if out.String() != writerPartialJDK {
		t.Fatalf("actual partial JDK mismatch\ngot:\n%s\nwant:\n%s", out.String(), writerPartialJDK)
	}
	fmt.Print(out.String())
}

const writerPartialJDK = `partial:false|count=3|src=4:6|cursor=7|length=4096|prefix=99,0,0|tail=0,13,14,15
full:false|IOException:File too large|src=4|cursor=7|length=4096
partial:true|count=3|src=4:6|cursor=4096|length=4096|prefix=99,0,0|tail=0,13,14,15
full:true|IOException:File too large|src=4|cursor=4096|length=4096
`
