package stdjava

import "os"

func outputStreamRange(array *PrimitiveArray[int8], offset, length int32) []byte {
	ReferenceRequireNonNull(array)
	if offset < 0 || length < 0 || offset > int32(len(array.Elements))-length {
		panic(NewIndexOutOfBoundsException("OutputStream.write range"))
	}
	return unsignedBytes(array.Elements[offset : offset+length])
}
func (s *ByteArrayOutputStream) WriteRange(array *PrimitiveArray[int8], offset, length int32) {
	s.buf.Write(outputStreamRange(array, offset, length))
}
func (s *FileOutputStream) WriteRange(array *PrimitiveArray[int8], offset, length int32) {
	if _, err := s.buf.Write(outputStreamRange(array, offset, length)); err != nil {
		throwIOException(err)
	}
}
func FilesReadAllBytes(path any) *PrimitiveArray[int8] {
	ReferenceRequireNonNull(path)
	data, err := os.ReadFile(ioPathOf(path))
	if err != nil {
		throwIOException(err)
	}
	return signedByteArray(data)
}
