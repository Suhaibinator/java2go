package stdjava

import (
	"io"
	"testing"
)

type lifecycleInput struct {
	*ByteArrayInputStream
	closes int
}

func (s *lifecycleInput) Close() { s.closes++ }
func TestInputStreamReaderCloseLifecycle(t *testing.T) {
	source := &lifecycleInput{ByteArrayInputStream: NewByteArrayInputStream([]byte("x"))}
	reader := NewInputStreamReaderExecution(nil, source, UTF_8)
	reader.Close()
	reader.Close()
	if source.closes != 1 {
		t.Fatalf("close count %d", source.closes)
	}
	if _, err := reader.Read(make([]byte, 1)); err == nil || err == io.EOF {
		t.Fatalf("closed reader read: %v", err)
	}
}
