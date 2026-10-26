package stdjava

import (
	"fmt"
	"strings"
	"testing"
)

func TestFileChannelSignatureReflectionJDK21(t *testing.T) {
	e := NewExecution()
	var out strings.Builder
	names := []TypeID{
		JavaLangAutoCloseableSignatureType, JavaNIOAbstractInterruptibleChannelType,
		JavaNIOBufferType, JavaNIOByteBufferType, JavaNIOByteChannelType, JavaNIOChannelType,
		JavaNIOFileChannelMapModeType, JavaNIOFileChannelType, JavaNIOFileLockType,
		JavaNIOGatheringByteChannelType, JavaNIOInterruptibleChannelType,
		JavaNIOMappedByteBufferType, JavaNIOReadableByteChannelType,
		JavaNIOScatteringByteChannelType, JavaNIOSeekableByteChannelType,
		JavaNIOWritableByteChannelType,
	}
	for _, name := range names {
		value := ClassForName(e, string(name))
		parent := value.GetSuperclass()
		parentName := "null"
		if parent != nil {
			parentName = parent.GetName()
		}
		fmt.Fprintf(&out, "%s|%s|%d|%t\n", value.GetName(), parentName, value.GetModifiers(), value == ClassForName(e, string(name)))
	}
	if out.String() != channelSignatureReflectionJDK21 {
		t.Fatalf("full JDK21 declaration reflection mismatch\ngot:\n%s\nwant:\n%s", out.String(), channelSignatureReflectionJDK21)
	}
}

// Complete stdout from three fresh pinned JDK21 runs before any descriptor repair.
const channelSignatureReflectionJDK21 = `java.lang.AutoCloseable|null|1537|true
java.nio.channels.spi.AbstractInterruptibleChannel|java.lang.Object|1025|true
java.nio.Buffer|java.lang.Object|1025|true
java.nio.ByteBuffer|java.nio.Buffer|1025|true
java.nio.channels.ByteChannel|null|1537|true
java.nio.channels.Channel|null|1537|true
java.nio.channels.FileChannel$MapMode|java.lang.Object|9|true
java.nio.channels.FileChannel|java.nio.channels.spi.AbstractInterruptibleChannel|1025|true
java.nio.channels.FileLock|java.lang.Object|1025|true
java.nio.channels.GatheringByteChannel|null|1537|true
java.nio.channels.InterruptibleChannel|null|1537|true
java.nio.MappedByteBuffer|java.nio.ByteBuffer|1025|true
java.nio.channels.ReadableByteChannel|null|1537|true
java.nio.channels.ScatteringByteChannel|null|1537|true
java.nio.channels.SeekableByteChannel|null|1537|true
java.nio.channels.WritableByteChannel|null|1537|true
`
