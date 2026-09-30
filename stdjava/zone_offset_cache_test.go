package stdjava

import (
	"sync"
	"testing"
)

var zoneOffsetCacheControlSink *ZoneOffset

func TestZoneOffsetCacheCanonicalIdentity(t *testing.T) {
	for _, seconds := range []int32{0, 3600, 56700, -64800, 64800} {
		first := ZoneOffsetOfTotalSeconds(seconds)
		for i := 0; i < 32; i++ {
			got := ZoneOffsetOfTotalSeconds(seconds)
			if got != first || got.GetID() != first.GetID() || got.GetTotalSeconds() != seconds {
				t.Fatalf("offset %d lost canonical offset/ID identity or value", seconds)
			}
		}
	}
}

func TestZoneOffsetCacheConcurrentMiss(t *testing.T) {
	const seconds int32 = 55800
	previous, existed := javaTimeOffsets.LoadAndDelete(seconds)
	t.Cleanup(func() {
		if existed {
			javaTimeOffsets.Store(seconds, previous)
		} else {
			javaTimeOffsets.Delete(seconds)
		}
	})
	start := make(chan struct{})
	results := make([]*ZoneOffset, 32)
	var workers sync.WaitGroup
	for i := range results {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			results[index] = ZoneOffsetOfTotalSeconds(seconds)
		}(i)
	}
	close(start)
	workers.Wait()
	first := results[0]
	for _, got := range results {
		if got != first || got.GetID() != first.GetID() || got.GetTotalSeconds() != seconds {
			t.Fatal("concurrent cold cache calls did not return the same canonical reference")
		}
	}
}

func TestZoneOffsetCacheHitAllocs(t *testing.T) {
	zoneOffsetCacheControlSink = ZoneOffsetOfTotalSeconds(3600)
	allocations := testing.AllocsPerRun(100, func() {
		zoneOffsetCacheControlSink = ZoneOffsetOfTotalSeconds(3600)
	})
	if allocations != 0 {
		t.Fatalf("warm cached offset lookup allocates %g objects/call; want 0", allocations)
	}
}

// Measures only the allocation cost of the warmed runtime cache lookup. This is
// neither a Java comparison nor evidence for generated application speedup.
func BenchmarkZoneOffsetCachedLookup(b *testing.B) {
	zoneOffsetCacheControlSink = ZoneOffsetOfTotalSeconds(3600)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zoneOffsetCacheControlSink = ZoneOffsetOfTotalSeconds(3600)
	}
}
