package stdjava

import (
	"math"
	"sync"
	"testing"
)

func atomicArrayPanic(t *testing.T, kind string, action func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		switch kind {
		case "null":
			if _, ok := recovered.(NullPointerException); !ok {
				t.Fatalf("want NullPointerException, got %T", recovered)
			}
		case "size":
			if _, ok := recovered.(NegativeArraySizeException); !ok {
				t.Fatalf("want NegativeArraySizeException, got %T", recovered)
			}
		case "index":
			if _, ok := recovered.(ArrayIndexOutOfBoundsException); !ok {
				t.Fatalf("want ArrayIndexOutOfBoundsException, got %T", recovered)
			}
		}
	}()
	action()
}

func TestAtomicArraysCopyWidthIdentityAndBounds(t *testing.T) {
	inputI := PrimitiveArrayLiteral[int32](PrimitiveIntTypeID, math.MinInt32, math.MaxInt32)
	inputL := PrimitiveArrayLiteral[int64](PrimitiveLongTypeID, math.MinInt64, 9007199254740993)
	ints, longs := NewAtomicIntegerArrayFromArray(inputI), NewAtomicLongArrayFromArray(inputL)
	inputI.Elements[0] = 7
	inputL.Elements[1] = 7
	if ints.Get(0) != math.MinInt32 || longs.Get(1) != 9007199254740993 {
		t.Fatal("constructor aliases input or loses long precision")
	}
	ints.Set(0, 31)
	longs.Set(0, 37)
	if inputI.Elements[0] != 7 || inputL.Elements[0] != math.MinInt64 {
		t.Fatal("atomic mutation aliases input")
	}
	firstEmptyInts, secondEmptyInts := NewAtomicIntegerArray(0), NewAtomicIntegerArray(0)
	firstEmptyLongs, secondEmptyLongs := NewAtomicLongArray(0), NewAtomicLongArray(0)
	if ints.Length() != 2 || longs.Length() != 2 || firstEmptyInts == secondEmptyInts || firstEmptyLongs == secondEmptyLongs {
		t.Fatal("length or object identity")
	}
	if !JavaTypeAssignable(AtomicIntegerArrayTypeID, SerializableTypeID) || !JavaTypeAssignable(AtomicLongArrayTypeID, ObjectTypeID) || JavaTypeAssignable(AtomicIntegerArrayTypeID, AtomicLongArrayTypeID) {
		t.Fatal("incorrect nominal hierarchy")
	}
	for _, action := range []func(){func() { NewAtomicIntegerArray(-1) }, func() { NewAtomicLongArray(-1) }} {
		atomicArrayPanic(t, "size", action)
	}
	for _, action := range []func(){func() { NewAtomicIntegerArrayFromArray(nil) }, func() { NewAtomicLongArrayFromArray(nil) }, func() { (*AtomicIntegerArray)(nil).Length() }, func() { (*AtomicLongArray)(nil).Get(0) }} {
		atomicArrayPanic(t, "null", action)
	}
	for _, action := range []func(){func() { ints.Get(-1) }, func() { ints.Set(2, 3) }, func() { ints.CompareAndSet(2, 0, 1) }, func() { ints.GetAndSet(-1, 1) }, func() { ints.GetAndAdd(2, 1) }, func() { ints.AddAndGet(-1, 1) }, func() { longs.Get(2) }, func() { longs.Set(-1, 3) }, func() { longs.CompareAndSet(2, 0, 1) }, func() { longs.GetAndSet(-1, 1) }, func() { longs.GetAndAdd(2, 1) }, func() { longs.AddAndGet(-1, 1) }} {
		atomicArrayPanic(t, "index", action)
	}
}

func TestAtomicArraysOverflowExchangeAndCAS(t *testing.T) {
	ints, longs := NewAtomicIntegerArray(1), NewAtomicLongArray(1)
	ints.Set(0, math.MaxInt32)
	longs.Set(0, math.MaxInt64)
	if ints.GetAndAdd(0, 1) != math.MaxInt32 || ints.Get(0) != math.MinInt32 || ints.AddAndGet(0, -1) != math.MaxInt32 {
		t.Fatal("int32 arithmetic does not wrap")
	}
	if longs.GetAndAdd(0, 1) != math.MaxInt64 || longs.Get(0) != math.MinInt64 || longs.AddAndGet(0, -1) != math.MaxInt64 {
		t.Fatal("int64 arithmetic does not wrap")
	}
	if ints.GetAndSet(0, 3) != math.MaxInt32 || longs.GetAndSet(0, 5) != math.MaxInt64 {
		t.Fatal("swap does not return previous value")
	}
	if !ints.CompareAndSet(0, 3, 7) || ints.CompareAndSet(0, 3, 11) || !longs.CompareAndSet(0, 5, 13) || longs.CompareAndSet(0, 5, 17) {
		t.Fatal("CAS failure overwrites value")
	}
	if ints.Get(0) != 7 || longs.Get(0) != 13 {
		t.Fatal("CAS stored wrong value")
	}
}

func TestAtomicArraysControlledContention(t *testing.T) {
	ints, longs := NewAtomicIntegerArray(2), NewAtomicLongArray(2)
	start := make(chan struct{})
	var cas, wg sync.WaitGroup
	cas.Add(4)
	wg.Add(4)
	for range 4 {
		go func() {
			defer wg.Done()
			<-start
			if ints.CompareAndSet(0, 0, 1) {
				ints.GetAndAdd(1, 1)
			}
			if longs.CompareAndSet(0, 0, 1) {
				longs.GetAndAdd(1, 1)
			}
			cas.Done()
			cas.Wait()
			for range 1000 {
				ints.GetAndAdd(0, 1)
				longs.GetAndAdd(0, 1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if ints.Get(0) != 4001 || longs.Get(0) != 4001 || ints.Get(1) != 1 || longs.Get(1) != 1 {
		t.Fatal("lost RMW update or multiple CAS winners")
	}
}
