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
	Close()
}

func InputStreamReadByte(stream InputStream) int32 { return InputStreamReadByteExecution(nil, stream) }
func InputStreamReadByteExecution(execution *Execution, stream InputStream) int32 {
	if execution == nil {
		execution = NewExecution()
	}
	ReferenceRequireNonNull(stream)
	if override, ok := stream.(interface{ JavaInputStreamReadByte(*Execution) int32 }); ok {
		return override.JavaInputStreamReadByte(execution)
	}
	var one [1]byte
	count, err := inputStreamReader(execution, stream).Read(one[:])
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
	return InputStreamReadIntoExecution(nil, stream, array, bounds...)
}
func InputStreamReadIntoExecution(execution *Execution, stream InputStream, array *PrimitiveArray[int8], bounds ...int32) int32 {
	if execution == nil {
		execution = NewExecution()
	}
	ReferenceRequireNonNull(stream)
	offset, length := int32(0), int32(0)
	if len(bounds) == 2 {
		offset, length = bounds[0], bounds[1]
	} else {
		ReferenceRequireNonNull(array)
		length = int32(len(array.Elements))
	}
	if override, ok := stream.(interface {
		JavaInputStreamRead(*Execution, *PrimitiveArray[int8], int32, int32) int32
	}); ok {
		return override.JavaInputStreamRead(execution, array, offset, length)
	}
	ReferenceRequireNonNull(array)
	if offset < 0 || length < 0 || offset > int32(len(array.Elements))-length {
		panic(NewIndexOutOfBoundsException("InputStream.read range"))
	}
	if length == 0 {
		return 0
	}
	bytes := make([]byte, length)
	count, err := inputStreamReader(execution, stream).Read(bytes)
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
	return InputStreamReadAllBytesExecution(nil, stream)
}
func InputStreamReadAllBytesExecution(execution *Execution, stream InputStream) *PrimitiveArray[int8] {
	ReferenceRequireNonNull(stream)
	bytes, err := io.ReadAll(inputStreamReader(execution, stream))
	if err != nil {
		throwIOException(err)
	}
	return signedByteArray(bytes)
}

// BufferedInputStream retains its source and closes that source exactly once.
type BufferedInputStream struct {
	source    InputStream
	execution *Execution
	reader    *bufio.Reader
	closed    bool
}

func NewBufferedInputStream(source InputStream, size ...int32) *BufferedInputStream {
	return NewBufferedInputStreamExecution(nil, source, size...)
}
func NewBufferedInputStreamExecution(execution *Execution, source InputStream, size ...int32) *BufferedInputStream {
	capacity := int32(8192)
	if len(size) > 0 {
		capacity = size[0]
	}
	if capacity <= 0 {
		panic(NewIllegalArgumentException("Buffer size <= 0"))
	}
	return &BufferedInputStream{source: source, execution: execution, reader: bufio.NewReaderSize(inputStreamReader(execution, source), int(capacity))}
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
			InputStreamCloseExecution(s.execution, s.source)
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
	mu       sync.Mutex
	file     *os.File
	closed   bool
	writable bool
}

func (s *randomAccessState) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	file := s.file
	s.mu.Unlock()
	// A retained RawConn lease may delay Close. Publish the closed state without
	// holding mu over the descriptor wait. Concurrent close completion ordering
	// remains outside this finite service's contract.
	if err := file.Close(); err != nil {
		throwIOException(err)
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
	state := &randomAccessState{file: file, writable: randomAccessFileWritableFromOpenFlags(flags)}
	return &RandomAccessFile{state: state, channel: &FileChannel{state: state}}
}
func (r *RandomAccessFile) GetChannel() *FileChannel       { return r.channel }
func (r *RandomAccessFile) Read(bytes []byte) (int, error) { return r.state.read(bytes) }
func (r *RandomAccessFile) ReadByteValue() int32           { return InputStreamReadByte(r) }
func (r *RandomAccessFile) Close()                         { r.state.close() }
func (r *RandomAccessFile) SeekPosition(position int64) {
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
	return FileChannelReadExecution(NewExecution(), c, buffer)
}

func (c *FileChannel) Close() { c.state.close() }
func (c *FileChannel) IsOpen() bool {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	return !c.state.closed
}

func init() {
	RegisterJavaType("InputStream", ObjectTypeID)
	RegisterJavaType("FileInputStream", "InputStream")
	RegisterJavaType("FilterInputStream", "InputStream")
	RegisterJavaType("ByteArrayInputStream", "InputStream")
	RegisterJavaType("BufferedInputStream", "InputStream")
	RegisterJavaType("OpenOption", ObjectTypeID)
	RegisterJavaType("StandardOpenOption", "OpenOption")
	RegisterJavaType("RandomAccessFile", ObjectTypeID)
	RegisterJavaType("FileChannel", ObjectTypeID)
	RegisterException("ClosedChannelException", "IOException")
}
