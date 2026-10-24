package stdjava

import (
	"sync"
	"testing"
)

func TestCIStringWriterConcurrentCanonicalSnapshots(t *testing.T) {
	writer := NewStringWriter()
	print := NewPrintWriter(writer)
	payload := NewJavaStringUTF16([]uint16{'A', 0, 0xd800, 0xdc00})
	const repetitions = 500
	var workers sync.WaitGroup
	for _, appendValue := range []func(){func() { print.Print(payload) }, func() { writer.WriteString("B") }, func() { _, _ = writer.Write([]byte{'C'}) }} {
		workers.Add(1)
		go func(appendValue func()) {
			defer workers.Done()
			for index := 0; index < repetitions; index++ {
				appendValue()
			}
		}(appendValue)
	}
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	for {
		snapshot := writer.StringReference()
		if snapshot == nil {
			t.Fatal("null StringWriter snapshot")
		}
		_ = writer.String()
		select {
		case <-done:
			goto finished
		default:
		}
	}
finished:
	counts := map[uint16]int{}
	snapshot := writer.StringReference()
	for _, unit := range snapshot.UTF16Copy() {
		counts[unit]++
	}
	if snapshot.Length() != 6*repetitions {
		t.Fatalf("snapshot length %d, want %d", snapshot.Length(), 6*repetitions)
	}
	for _, unit := range []uint16{'A', 0, 0xd800, 0xdc00, 'B', 'C'} {
		if counts[unit] != repetitions {
			t.Fatalf("unit %#x count %d, want %d", unit, counts[unit], repetitions)
		}
	}
}
