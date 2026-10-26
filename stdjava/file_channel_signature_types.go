package stdjava

// These types close declaration signatures only. They do not admit channel
// interface services or map/lock/transfer/constructors. Opaque signature pointers
// deliberately expose no JavaDynamicTypeID or fabricated abstract-class values.
type JavaReadableByteChannel interface{ JavaReadableByteChannelMarker() }
type JavaWritableByteChannel interface{ JavaWritableByteChannelMarker() }

func (*FileChannel) JavaReadableByteChannelMarker() {}
func (*FileChannel) JavaWritableByteChannelMarker() {}

type JavaFileChannelMapMode struct{ _ byte }
type JavaMappedByteBuffer struct{ _ byte }
type JavaFileLock struct{ _ byte }

const (
	JavaLangAutoCloseableSignatureType      TypeID = "java.lang.AutoCloseable"
	JavaNIOAbstractInterruptibleChannelType TypeID = "java.nio.channels.spi.AbstractInterruptibleChannel"
	JavaNIOBufferType                       TypeID = "java.nio.Buffer"
	JavaNIOByteBufferType                   TypeID = "java.nio.ByteBuffer"
	JavaNIOByteChannelType                  TypeID = "java.nio.channels.ByteChannel"
	JavaNIOChannelType                      TypeID = "java.nio.channels.Channel"
	JavaNIOFileChannelMapModeType           TypeID = "java.nio.channels.FileChannel$MapMode"
	JavaNIOFileChannelType                  TypeID = "java.nio.channels.FileChannel"
	JavaNIOFileLockType                     TypeID = "java.nio.channels.FileLock"
	JavaNIOGatheringByteChannelType         TypeID = "java.nio.channels.GatheringByteChannel"
	JavaNIOInterruptibleChannelType         TypeID = "java.nio.channels.InterruptibleChannel"
	JavaNIOMappedByteBufferType             TypeID = "java.nio.MappedByteBuffer"
	JavaNIOReadableByteChannelType          TypeID = "java.nio.channels.ReadableByteChannel"
	JavaNIOScatteringByteChannelType        TypeID = "java.nio.channels.ScatteringByteChannel"
	JavaNIOSeekableByteChannelType          TypeID = "java.nio.channels.SeekableByteChannel"
	JavaNIOWritableByteChannelType          TypeID = "java.nio.channels.WritableByteChannel"
)

func init() {
	RegisterJavaType(JavaLangAutoCloseableSignatureType, ObjectTypeID)
	RegisterJavaType(JavaNIOAbstractInterruptibleChannelType, ObjectTypeID, TypeID("java.nio.channels.Channel"), TypeID("java.nio.channels.InterruptibleChannel"))
	RegisterJavaType(JavaNIOBufferType, ObjectTypeID)
	RegisterJavaType(JavaNIOByteBufferType, TypeID("java.nio.Buffer"), ComparableTypeID)
	RegisterJavaType(JavaNIOByteChannelType, ObjectTypeID, TypeID("java.nio.channels.ReadableByteChannel"), TypeID("java.nio.channels.WritableByteChannel"))
	RegisterJavaType(JavaNIOChannelType, ObjectTypeID, TypeID("java.io.Closeable"), TypeID("java.lang.AutoCloseable"))
	RegisterJavaType(JavaNIOFileChannelMapModeType, ObjectTypeID)
	RegisterJavaType(JavaNIOFileChannelType, TypeID("java.nio.channels.spi.AbstractInterruptibleChannel"), TypeID("java.nio.channels.SeekableByteChannel"), TypeID("java.nio.channels.GatheringByteChannel"), TypeID("java.nio.channels.ScatteringByteChannel"))
	RegisterJavaType(JavaNIOFileLockType, ObjectTypeID, TypeID("java.lang.AutoCloseable"))
	RegisterJavaType(JavaNIOGatheringByteChannelType, ObjectTypeID, TypeID("java.nio.channels.WritableByteChannel"))
	RegisterJavaType(JavaNIOInterruptibleChannelType, ObjectTypeID, TypeID("java.nio.channels.Channel"))
	RegisterJavaType(JavaNIOMappedByteBufferType, TypeID("java.nio.ByteBuffer"))
	RegisterJavaType(JavaNIOReadableByteChannelType, ObjectTypeID, TypeID("java.nio.channels.Channel"))
	RegisterJavaType(JavaNIOScatteringByteChannelType, ObjectTypeID, TypeID("java.nio.channels.ReadableByteChannel"))
	RegisterJavaType(JavaNIOSeekableByteChannelType, ObjectTypeID, TypeID("java.nio.channels.ByteChannel"))
	RegisterJavaType(JavaNIOWritableByteChannelType, ObjectTypeID, TypeID("java.nio.channels.Channel"))

	RegisterClassDescriptor(ClassDescriptor{Type: JavaLangAutoCloseableSignatureType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOAbstractInterruptibleChannelType, HasModifiers: true, Modifiers: 1025})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOBufferType, HasModifiers: true, Modifiers: 1025})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOByteBufferType, HasModifiers: true, Modifiers: 1025})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOByteChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOFileChannelMapModeType, HasModifiers: true, Modifiers: 9})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOFileChannelType, HasModifiers: true, Modifiers: 1025})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOFileLockType, HasModifiers: true, Modifiers: 1025})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOGatheringByteChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOInterruptibleChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOMappedByteBufferType, HasModifiers: true, Modifiers: 1025})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOReadableByteChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOScatteringByteChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOSeekableByteChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
	RegisterClassDescriptor(ClassDescriptor{Type: JavaNIOWritableByteChannelType, Interface: true, HasModifiers: true, Modifiers: 1537})
}
