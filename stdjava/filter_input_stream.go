package stdjava

import (
	"io"
	"os"

	"golang.org/x/text/transform"
)

// FilterInputStream exposes the protected Java `in` field. The generated
// subclass supplies typed Java read bridges, separate from Go's io.Reader.
type FilterInputStream struct{ In InputStream }

func NewFilterInputStream(source InputStream) *FilterInputStream {
	return &FilterInputStream{In: source}
}
func (s *FilterInputStream) Close()                     { InputStreamCloseExecution(nil, s.In) }
func (s *FilterInputStream) Read(p []byte) (int, error) { return inputStreamReader(nil, s.In).Read(p) }
func InputStreamCloseExecution(execution *Execution, stream InputStream) {
	if execution == nil {
		execution = NewExecution()
	}
	ReferenceRequireNonNull(stream)
	if override, ok := stream.(interface{ CloseJava2goExecution(*Execution) }); ok {
		override.CloseJava2goExecution(execution)
		return
	}
	stream.Close()
}

type javaInputStreamReader struct {
	execution *Execution
	source    InputStream
}

func inputStreamReader(execution *Execution, source InputStream) io.Reader {
	ReferenceRequireNonNull(source)
	if execution == nil {
		execution = NewExecution()
	}
	if _, ok := source.(interface {
		JavaInputStreamRead(*Execution, *PrimitiveArray[int8], int32, int32) int32
	}); ok {
		return &javaInputStreamReader{execution, source}
	}
	if reader, ok := source.(io.Reader); ok {
		return reader
	}
	panic(NewUnsupportedOperationException("InputStream has no read implementation"))
}
func (r *javaInputStreamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	array := PrimitiveArrayLiteral(PrimitiveByteTypeID, make([]int8, len(p))...)
	n := InputStreamReadIntoExecution(r.execution, r.source, array, 0, int32(len(p)))
	if n < 0 {
		return 0, io.EOF
	}
	for i := 0; i < int(n); i++ {
		p[i] = byte(array.Elements[i])
	}
	return int(n), nil
}

// Character decoding is streaming, so construction does not consume input and
// reader buffering still flows through the Java subclass's read overrides.
func NewInputStreamReaderExecution(execution *Execution, source any, charsets ...any) *InputStreamReader {
	charset := UTF_8
	if len(charsets) > 0 {
		ReferenceRequireNonNull(charsets[0])
		switch c := charsets[0].(type) {
		case *Charset:
			charset = c
		case string:
			charset = charsetForEncodingName(c)
		default:
			panic(NewIllegalArgumentException("InputStreamReader charset"))
		}
	}
	var src ioSource
	if stream, ok := source.(InputStream); ok {
		src = ioSource{r: inputStreamReader(execution, stream), close: func() { InputStreamCloseExecution(execution, stream) }}
	} else {
		src = ioSourceOf(source)
	}
	src.r = transform.NewReader(src.r, newInputStreamDecoder(charset))
	return &InputStreamReader{src: src}
}

func NewInputStreamReaderStdinExecution(execution *Execution, charset any) *InputStreamReader {
	return NewInputStreamReaderExecution(execution, os.Stdin, charset)
}

// US-ASCII replaces each unmappable input byte independently, as the JDK does.
type asciiReaderDecoder struct{}

func (asciiReaderDecoder) Reset() {}
func (asciiReaderDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		if src[nSrc] < 128 {
			if nDst == len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[nDst] = src[nSrc]
			nDst++
		} else {
			if len(dst)-nDst < 3 {
				return nDst, nSrc, transform.ErrShortDst
			}
			copy(dst[nDst:], "\ufffd")
			nDst += 3
		}
		nSrc++
	}
	return
}
