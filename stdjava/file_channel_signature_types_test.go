package stdjava

import (
	"reflect"
	"runtime"
	"testing"
)

func TestFileChannelSignatureNominality(t *testing.T) {
	for _, pair := range [][2]TypeID{{JavaNIOFileChannelType, JavaNIOReadableByteChannelType}, {JavaNIOFileChannelType, JavaNIOWritableByteChannelType}, {JavaNIOFileChannelType, JavaIOCloseableType}, {JavaNIOFileLockType, JavaLangAutoCloseableSignatureType}, {JavaNIOMappedByteBufferType, JavaNIOByteBufferType}, {JavaNIOByteBufferType, ComparableTypeID}} {
		if !JavaTypeAssignable(pair[0], pair[1]) {
			t.Fatalf("declaration edge missing: %s -> %s", pair[0], pair[1])
		}
	}
	for _, id := range []TypeID{JavaNIOFileLockType, JavaNIOFileChannelMapModeType, JavaNIOMappedByteBufferType} {
		if JavaTypeAssignable(id, JavaNIOFileChannelType) || JavaTypeAssignable(JavaNIOFileChannelType, id) {
			t.Fatalf("unrelated signature types merged: %s", id)
		}
	}
	types := []reflect.Type{reflect.TypeFor[*JavaFileChannelMapMode](), reflect.TypeFor[*JavaMappedByteBuffer](), reflect.TypeFor[*JavaFileLock](), reflect.TypeFor[*ByteBuffer](), reflect.TypeFor[JavaReadableByteChannel](), reflect.TypeFor[JavaWritableByteChannel]()}
	for i, typ := range types {
		for _, other := range types[i+1:] {
			if typ == other {
				t.Fatal("physical declaration types erased together")
			}
		}
	}
	for _, value := range []any{&JavaFileChannelMapMode{}, &JavaMappedByteBuffer{}, &JavaFileLock{}} {
		if _, ok := ObjectDynamicType(value); ok {
			t.Fatal("opaque signature value fabricated a Java dynamic class")
		}
	}
	var mode *JavaFileChannelMapMode
	var mapped *JavaMappedByteBuffer
	var lock *JavaFileLock
	if !nilJavaReference(mode) || !nilJavaReference(mapped) || !nilJavaReference(lock) {
		t.Fatal("typed signature null changed")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		// Explicit unsupported native target refusal, not a platform success claim.
		c := &FileChannel{state: &randomAccessState{}}
		if failure := channelProbePanic(func() { FileChannelReadAtVirtualExecution(NewExecution(), c, nil, -1) }); !CaughtAs(failure, "UnsupportedOperationException") {
			t.Fatalf("unsupported target synthesized success: %v", failure)
		}
	}
}
