package stdjava

import (
	"runtime"
	"slices"
	"sync"
	"weak"
)

// The index never points back to its pool or the pool's strong literal roots.
// Cleanup arguments may retain this weak-only index without retaining strings.
type stringInternIndex struct {
	mu      sync.Mutex
	buckets map[uint64][]weak.Pointer[JavaString]
}

type stringInternPool struct {
	index       *stringInternIndex
	hash        func(*JavaString) uint64
	defaultHash bool
	// Protected by index.mu; only explicitly resolved literals receive roots.
	literals map[uint64][]*JavaString
}

type stringInternCleanup struct {
	index    *stringInternIndex
	bucket   uint64
	identity weak.Pointer[JavaString]
}

func newStringInternPool(hash func(*JavaString) uint64) *stringInternPool {
	defaultHash := hash == nil
	if defaultHash {
		hash = javaStringInternHash
	}
	return &stringInternPool{
		index:       &stringInternIndex{buckets: make(map[uint64][]weak.Pointer[JavaString])},
		hash:        hash,
		defaultHash: defaultHash,
		literals:    make(map[uint64][]*JavaString),
	}
}

// Hashing chooses a bucket only. Equality always compares every UTF16 unit.
func javaStringInternHash(value *JavaString) uint64 {
	return javaStringInternHashUnits(value.units)
}

func javaStringInternHashUnits(units []uint16) uint64 {
	const prime = uint64(1099511628211)
	hash := uint64(14695981039346656037)
	for _, unit := range units {
		hash = (hash ^ uint64(unit&255)) * prime
		hash = (hash ^ uint64(unit>>8)) * prime
	}
	return hash
}

func (pool *stringInternPool) intern(value *JavaString) *JavaString {
	ReferenceRequireNonNull(value)
	bucket := pool.hash(value)
	pool.index.mu.Lock()
	result := pool.internLocked(bucket, value)
	pool.index.mu.Unlock()
	runtime.KeepAlive(value)
	return result
}

// Caller holds index.mu. Strong references from Value stay live through full
// content comparison and returning the selected representative.
func (pool *stringInternPool) internLocked(bucket uint64, value *JavaString) *JavaString {
	result, live := pool.findInternedLocked(bucket, value.units)
	if result != nil {
		return result
	}
	return pool.addInternedLocked(bucket, value, live)
}

// Caller holds index.mu. Compare complete immutable UTF16 content and retain
// live weak handles while removing stale handles from spare backing capacity.
func (pool *stringInternPool) findInternedLocked(bucket uint64, units []uint16) (*JavaString, []weak.Pointer[JavaString]) {
	entries := pool.index.buckets[bucket]
	live := entries[:0]
	var result *JavaString
	for _, entry := range entries {
		existing := entry.Value()
		if existing == nil {
			continue
		}
		live = append(live, entry)
		if result == nil && slices.Equal(existing.units, units) {
			result = existing
		}
	}
	clear(entries[len(live):])
	pool.index.buckets[bucket] = live
	return result, live
}

func (pool *stringInternPool) addInternedLocked(bucket uint64, value *JavaString, live []weak.Pointer[JavaString]) *JavaString {
	identity := weak.Make(value)
	pool.index.buckets[bucket] = append(live, identity)
	runtime.AddCleanup(value, cleanupStringInternEntry, stringInternCleanup{pool.index, bucket, identity})
	runtime.KeepAlive(value)
	return value
}

func (pool *stringInternPool) literal(units []uint16) *JavaString {
	var candidate, result *JavaString
	var bucket uint64
	if pool.defaultHash {
		// The default hash consumes units directly. An existing representative
		// needs neither a temporary wrapper nor a defensive payload copy.
		bucket = javaStringInternHashUnits(units)
		pool.index.mu.Lock()
		var live []weak.Pointer[JavaString]
		result, live = pool.findInternedLocked(bucket, units)
		if result == nil {
			candidate = NewJavaStringUTF16(units)
			result = pool.addInternedLocked(bucket, candidate, live)
		}
	} else {
		// Custom hash callbacks retain the original fresh, isolated candidate
		// and callback timing, including collisions and candidate mutation.
		candidate = NewJavaStringUTF16(units)
		bucket = pool.hash(candidate)
		pool.index.mu.Lock()
		result = pool.internLocked(bucket, candidate)
	}
	rooted := false
	for _, literal := range pool.literals[bucket] {
		if literal == result {
			rooted = true
			break
		}
	}
	if !rooted {
		pool.literals[bucket] = append(pool.literals[bucket], result)
	}
	pool.index.mu.Unlock()
	runtime.KeepAlive(candidate)
	return result
}

// Package-level cleanup has no receiver closure. Its argument contains only a
// weak-only index, a numeric bucket and a weak identity, never String content.
func cleanupStringInternEntry(entry stringInternCleanup) {
	entry.index.removeWeak(entry.bucket, entry.identity)
}

func (pool *stringInternPool) removeWeak(bucket uint64, identity weak.Pointer[JavaString]) {
	pool.index.removeWeak(bucket, identity)
}

func (index *stringInternIndex) removeWeak(bucket uint64, identity weak.Pointer[JavaString]) {
	index.mu.Lock()
	defer index.mu.Unlock()
	entries := index.buckets[bucket]
	kept := entries[:0]
	for _, entry := range entries {
		if entry != identity {
			kept = append(kept, entry)
		}
	}
	clear(entries[len(kept):])
	if len(kept) == 0 {
		delete(index.buckets, bucket)
	} else {
		index.buckets[bucket] = kept
	}
}

var javaStringPool = newStringInternPool(nil)

// InternJavaString canonicalizes one live String object by full UTF16 content.
func InternJavaString(value *JavaString) *JavaString { return javaStringPool.intern(value) }

// JavaStringLiteralUTF16 resolves and roots a Java literal in the shared intern
// universe. Compiler literal/constant lowering has not yet switched to this API.
func JavaStringLiteralUTF16(units []uint16) *JavaString { return javaStringPool.literal(units) }
