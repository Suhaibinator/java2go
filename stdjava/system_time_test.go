package stdjava

import (
	"testing"
	"time"
)

func TestSystemNanoTimeElapsed(t *testing.T) {
	start := SystemNanoTime()
	previous := start
	for range 1000 {
		now := SystemNanoTime()
		if now-previous < 0 {
			t.Fatal("successive elapsed difference moved backwards")
		}
		previous = now
	}
	time.Sleep(25 * time.Millisecond)
	if SystemNanoTime()-start <= 0 {
		t.Fatal("elapsed time did not advance across sleep")
	}
}

func TestSystemNanoTimeSharedOrigin(t *testing.T) {
	for range 32 {
		before := SystemNanoTime()
		reading := make(chan int64)
		go func() { reading <- SystemNanoTime() }()
		worker := <-reading
		after := SystemNanoTime()
		if worker-before < 0 || after-worker < 0 {
			t.Fatal("ordered cross-goroutine readings do not share a monotonic origin")
		}
	}
}
