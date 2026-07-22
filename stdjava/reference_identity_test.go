package stdjava

import "testing"

type referenceIdentityProbe struct {
	value int
}

func TestJavaReferenceEqualNormalizesTypedNil(t *testing.T) {
	var pointer *referenceIdentityProbe
	var mapping map[string]int
	var slice []int
	var function func()

	for name, value := range map[string]any{
		"pointer":  pointer,
		"map":      mapping,
		"slice":    slice,
		"function": function,
	} {
		if !JavaReferenceEqual(value, nil) || !JavaReferenceEqual(nil, value) {
			t.Fatalf("typed nil %s did not compare equal to Java null", name)
		}
	}
}

func TestJavaReferenceEqualPreservesAvailableIdentity(t *testing.T) {
	first := &referenceIdentityProbe{value: 7}
	alias := first
	second := &referenceIdentityProbe{value: 7}
	if !JavaReferenceEqual(first, alias) {
		t.Fatal("pointer alias did not retain identity")
	}
	if JavaReferenceEqual(first, second) {
		t.Fatal("distinct pointers collapsed to one identity")
	}

	firstMap := map[string]int{"x": 1}
	mapAlias := firstMap
	secondMap := map[string]int{"x": 1}
	if !JavaReferenceEqual(firstMap, mapAlias) {
		t.Fatal("map alias did not retain identity")
	}
	if JavaReferenceEqual(firstMap, secondMap) {
		t.Fatal("distinct maps collapsed to one identity")
	}

	firstSlice := []int{1}
	sliceAlias := firstSlice
	secondSlice := []int{1}
	if !JavaReferenceEqual(firstSlice, sliceAlias) {
		t.Fatal("slice alias did not retain identity")
	}
	if JavaReferenceEqual(firstSlice, secondSlice) {
		t.Fatal("distinct slices collapsed to one identity")
	}
}

func TestJavaReferenceEqualHandlesUncomparableValuesWithoutPanic(t *testing.T) {
	firstFunction := func() {}
	secondFunction := func() {}
	if JavaReferenceEqual(firstFunction, secondFunction) {
		t.Fatal("distinct functions collapsed to one identity")
	}

	type uncomparable struct {
		values []int
	}
	if JavaReferenceEqual(uncomparable{values: []int{1}}, uncomparable{values: []int{1}}) {
		t.Fatal("uncomparable values unexpectedly compared identical")
	}
}

func TestJavaReferenceEqualHonorsNullStringRepresentation(t *testing.T) {
	if !JavaReferenceEqual(NullString(), nil) || !JavaReferenceEqual(nil, NullString()) {
		t.Fatal("null String sentinel did not compare equal to Java null")
	}
	if !JavaReferenceEqual(NullString(), NullString()) {
		t.Fatal("two null String representations did not compare equal")
	}
	if JavaReferenceEqual(NullString(), "") {
		t.Fatal("null String sentinel compared equal to empty String")
	}
	if !JavaReferenceEqual("same", "same") {
		t.Fatal("value-backed equal String representations compared unequal")
	}
}
