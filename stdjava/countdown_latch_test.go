package stdjava

import (
	"sync"
	"testing"
)

func TestCountDownLatchPublishesAllArrivalsToAllWaiters(t *testing.T) {
	const count = 16
	latch := NewCountDownLatch(count)
	values := make([]int, count)
	var waiters sync.WaitGroup
	for range count {
		waiters.Go(func() {
			latch.Await()
			for index, value := range values {
				if value != index+1 {
					t.Errorf("arrival %d published %d", index, value)
				}
			}
		})
	}
	for index := range count {
		go func() {
			values[index] = index + 1
			latch.CountDown()
		}()
	}
	waiters.Wait()
	if latch.GetCount() != 0 {
		t.Fatal("latch did not open")
	}
}
