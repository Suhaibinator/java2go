package stdjava

import (
	"bufio"
	"io"
	"os"
	"sync"
)

// InputStream is the common byte-reader view shared by Java input streams.
// Java overloads of read are lowered to InputStreamReadByte/Into because Go's
// io.Reader.Read has a different result and buffer representation.
type InputStream interface {
	io.Reader
	Close()
}

func InputStreamReadByte(stream InputStream) int32 {
	ReferenceRequireNonNull(stream)
	var one [1]byte
	count, err := stream.Read(one[:])
	if count > 0 {
		return int32(one[0])
	}
	if err == io.EOF {
		return -1
	}
	if err != nil {
		throwIOException(err)
	}
	return 0
}
func InputStreamReadInto(stream InputStream, array *PrimitiveArray[int8], bounds ...int32) int32 {
	ReferenceRequireNonNull(stream)
	ReferenceRequireNonNull(array)
	offset, length := int32(0), int32(len(array.Elements))
	if len(bounds) == 2 {
		offset, length = bounds[0], bounds[1]
	}
	if offset < 0 || length < 0 || offset > int32(len(array.Elements))-length {
		panic(NewIndexOutOfBoundsException("InputStream.read range"))
	}
	if length == 0 {
		return 0
	}
	bytes := make([]byte, length)
	count, err := stream.Read(bytes)
	for i := 0; i < count; i++ {
		array.Elements[int(offset)+i] = int8(bytes[i])
	}
	if count > 0 {
		return int32(count)
	}
	if err == io.EOF {
		return -1
	}
	if err != nil {
		throwIOException(err)
	}
	return 0
}
func InputStreamReadAllBytes(stream InputStream) *PrimitiveArray[int8] {
	ReferenceRequireNonNull(stream)
	bytes, err := io.ReadAll(stream)
	if err != nil {
		throwIOException(err)
	}
	return signedByteArray(bytes)
}

// BufferedInputStream retains its source and closes that source exactly once.
type BufferedInputStream struct {
	source InputStream
	reader *bufio.Reader
	closed bool
}

func NewBufferedInputStream(source InputStream, size ...int32) *BufferedInputStream {
	capacity := int32(8192)
	if len(size) > 0 {
		capacity = size[0]
	}
	if capacity <= 0 {
		panic(NewIllegalArgumentException("Buffer size <= 0"))
	}
	return &BufferedInputStream{source: source, reader: bufio.NewReaderSize(source, int(capacity))}
}
func (s *BufferedInputStream) Read(bytes []byte) (int, error) {
	if s.closed || javaReferenceIsNull(s.source) {
		return 0, os.ErrClosed
	}
	return s.reader.Read(bytes)
}
func (s *BufferedInputStream) Close() {
	if !s.closed {
		s.closed = true
		if !javaReferenceIsNull(s.source) {
			s.source.Close()
		}
	}
}
func (s *BufferedInputStream) ReadByteValue() int32                { return InputStreamReadByte(s) }
func (s *BufferedInputStream) ReadAllBytes() *PrimitiveArray[int8] { return InputStreamReadAllBytes(s) }

// OpenOption is a nominal marker, matching java.nio.file.OpenOption.
type OpenOption interface{ JavaOpenOption() }
type StandardOpenOption string

func (StandardOpenOption) JavaOpenOption()           {}
func (StandardOpenOption) JavaDynamicTypeID() TypeID { return "StandardOpenOption" }

const (
	StandardOpenOptionRead             StandardOpenOption = "READ"
	StandardOpenOptionWrite            StandardOpenOption = "WRITE"
	StandardOpenOptionAppend           StandardOpenOption = "APPEND"
	StandardOpenOptionCreate           StandardOpenOption = "CREATE"
	StandardOpenOptionCreateNew        StandardOpenOption = "CREATE_NEW"
	StandardOpenOptionTruncateExisting StandardOpenOption = "TRUNCATE_EXISTING"
	StandardOpenOptionDeleteOnClose    StandardOpenOption = "DELETE_ON_CLOSE"
	StandardOpenOptionSparse           StandardOpenOption = "SPARSE"
	StandardOpenOptionSync             StandardOpenOption = "SYNC"
	StandardOpenOptionDsync            StandardOpenOption = "DSYNC"
)

func FilesNewInputStream(path any, options ...any) InputStream {
	ReferenceRequireNonNull(path)
	deleteOnClose := false
	var check func(any)
	check = func(option any) {
		ReferenceRequireNonNull(option)
		switch value := option.(type) {
		case *ReferenceArray:
			for _, element := range value.elements {
				check(element)
			}
		case []OpenOption:
			for _, element := range value {
				check(element)
			}
		case StandardOpenOption:
			switch value {
			case StandardOpenOptionRead:
			case StandardOpenOptionDeleteOnClose:
				deleteOnClose = true
			case StandardOpenOptionWrite, StandardOpenOptionAppend:
				panic(NewIllegalArgumentException("READ not allowed with WRITE or APPEND"))
			default:
				panic(NewUnsupportedOperationException("unsupported newInputStream option: " + string(value)))
			}
		default:
			panic(NewUnsupportedOperationException("unsupported newInputStream option"))
		}
	}
	for _, option := range options {
		check(option)
	}
	stream := NewFileInputStream(path)
	if deleteOnClose {
		return &deleteOnCloseInputStream{FileInputStream: stream, path: ioPathOf(path)}
	}
	return stream
}

type deleteOnCloseInputStream struct {
	*FileInputStream
	path string
}

func (s *deleteOnCloseInputStream) Close() {
	s.FileInputStream.Close()
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		throwIOException(err)
	}
}

type randomAccessState struct {
	mu     sync.Mutex
	file   *os.File
	closed bool
}

func (s *randomAccessState) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		if err := s.file.Close(); err != nil {
			throwIOException(err)
		}
	}
}
func (s *randomAccessState) read(bytes []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, os.ErrClosed
	}
	return s.file.Read(bytes)
}

// RandomAccessFile and its FileChannel share the same descriptor and cursor.
// Closing either view closes both, as required by the JDK.
type RandomAccessFile struct {
	state   *randomAccessState
	channel *FileChannel
}

func NewRandomAccessFile(path any, mode string) *RandomAccessFile {
	flags := os.O_RDONLY
	switch mode {
	case "r":
	case "rw":
		flags = os.O_RDWR | os.O_CREATE
	case "rws", "rwd":
		flags = os.O_RDWR | os.O_CREATE | os.O_SYNC
	default:
		panic(NewIllegalArgumentException("Illegal mode: " + mode))
	}
	ReferenceRequireNonNull(path)
	file, err := os.OpenFile(ioPathOf(path), flags, 0666)
	if err != nil {
		throwIOException(err)
	}
	state := &randomAccessState{file: file}
	return &RandomAccessFile{state: state, channel: &FileChannel{state: state}}
}
func (r *RandomAccessFile) GetChannel() *FileChannel       { return r.channel }
func (r *RandomAccessFile) Read(bytes []byte) (int, error) { return r.state.read(bytes) }
func (r *RandomAccessFile) ReadByteValue() int32           { return InputStreamReadByte(r) }
func (r *RandomAccessFile) Close()                         { r.state.close() }
func (r *RandomAccessFile) Seek(position int64) {
	if position < 0 {
		panic(NewIOException("Negative seek offset"))
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	if _, err := r.state.file.Seek(position, io.SeekStart); err != nil {
		throwIOException(err)
	}
}
func (r *RandomAccessFile) GetFilePointer() int64 {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	position, err := r.state.file.Seek(0, io.SeekCurrent)
	if err != nil {
		throwIOException(err)
	}
	return position
}
func (r *RandomAccessFile) Length() int64 {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	info, err := r.state.file.Stat()
	if err != nil {
		throwIOException(err)
	}
	return info.Size()
}

type FileChannel struct{ state *randomAccessState }

func (c *FileChannel) Read(buffer *ByteBuffer) int32 {
	ReferenceRequireNonNull(buffer)
	if buffer.Remaining() == 0 {
		return 0
	}
	bytes := make([]byte, buffer.Remaining())
	count, err := c.state.read(bytes)
	for i := 0; i < count; i++ {
		buffer.array.Elements[int(buffer.position)+i] = int8(bytes[i])
	}
	buffer.position += int32(count)
	if count > 0 {
		return int32(count)
	}
	if err == io.EOF {
		return -1
	}
	if err != nil {
		throwIOException(err)
	}
	return 0
}
func (c *FileChannel) Close() { c.state.close() }
func (c *FileChannel) IsOpen() bool {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	return !c.state.closed
}
func (b *ByteBuffer) Flip() *ByteBuffer { b.limit = b.position; b.position = 0; return b }
func (b *ByteBuffer) Clear() *ByteBuffer {
	b.position = 0
	b.limit = int32(len(b.array.Elements))
	return b
}

func init() {
	RegisterJavaType("InputStream", ObjectTypeID)
	RegisterJavaType("FileInputStream", "InputStream")
	RegisterJavaType("ByteArrayInputStream", "InputStream")
	RegisterJavaType("BufferedInputStream", "InputStream")
	RegisterJavaType("OpenOption", ObjectTypeID)
	RegisterJavaType("StandardOpenOption", "OpenOption")
	RegisterJavaType("RandomAccessFile", ObjectTypeID)
	RegisterJavaType("FileChannel", ObjectTypeID)
}
