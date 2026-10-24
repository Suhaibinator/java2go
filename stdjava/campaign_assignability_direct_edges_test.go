package stdjava

import (
	"slices"
	"sync"
	"testing"
)

// Independent depth-first reachability oracle for this test's declared graph.
func campaignDeclaredReachable(edges map[TypeID][]TypeID, actual, expected TypeID) bool {
	if actual == "" || expected == "" {
		return false
	}
	visited := make(map[TypeID]bool)
	var visit func(TypeID) bool
	visit = func(current TypeID) bool {
		if current == expected {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true
		for _, parent := range edges[current] {
			if visit(parent) {
				return true
			}
		}
		return false
	}
	return visit(actual)
}

func TestCampaignAssignabilityDeclaredReachability(t *testing.T) {
	base := TypeID("campaign.edges.graph.Base")
	left := TypeID("campaign.edges.graph.Left")
	right := TypeID("campaign.edges.graph.Right")
	diamond := TypeID("campaign.edges.graph.Diamond")
	cycleA := TypeID("campaign.edges.graph.CycleA")
	cycleB := TypeID("campaign.edges.graph.CycleB")
	unknown := TypeID("campaign.edges.graph.Unregistered")
	child := TypeID("campaign.edges.graph.Child")
	absent := TypeID("campaign.edges.graph.Absent")
	edges := map[TypeID][]TypeID{
		base: {}, left: {base}, right: {base}, diamond: {left, right},
		cycleA: {cycleB}, cycleB: {cycleA}, child: {unknown},
	}
	for actual, parents := range edges {
		var super TypeID
		var interfaces []TypeID
		if len(parents) > 0 {
			super, interfaces = parents[0], parents[1:]
		}
		RegisterJavaType(actual, super, interfaces...)
	}
	nodes := []TypeID{base, left, right, diamond, cycleA, cycleB, unknown, child, absent, ""}
	for _, actual := range nodes {
		for _, expected := range nodes {
			if got, want := JavaTypeAssignable(actual, expected), campaignDeclaredReachable(edges, actual, expected); got != want {
				t.Fatalf("assignability %q -> %q = %v, declared reachability = %v", actual, expected, got, want)
			}
		}
	}
}

func TestCampaignAssignabilityRegistryUpdatesAndNominalIdentity(t *testing.T) {
	child := TypeID("campaign.edges.lifetime.Child")
	late := TypeID("campaign.edges.lifetime.Late")
	other := TypeID("campaign.edges.lifetime.Other")
	inter := TypeID("campaign.edges.lifetime.Interface")
	RegisterJavaType(child, late, inter)
	if !JavaTypeAssignable(child, late) || !JavaTypeAssignable(child, inter) || JavaTypeAssignable(child, ObjectTypeID) {
		t.Fatal("direct unregistered descriptor facts were lost or invented transitive edges")
	}
	RegisterJavaType(late, ObjectTypeID)
	if !JavaTypeAssignable(child, ObjectTypeID) {
		t.Fatal("late registration did not expose its transitive superclass")
	}
	RegisterJavaType(other, ObjectTypeID)
	RegisterJavaType(child, other)
	RegisterJavaSourceType(child)
	if JavaTypeAssignable(child, late) || JavaTypeAssignable(child, inter) || !JavaTypeAssignable(child, other) || !JavaTypeAssignable(child, ObjectTypeID) {
		t.Fatal("replacement/source metadata update retained stale hierarchy answers")
	}
	foreign := TypeID("campaign.edges.foreign.String")
	RegisterJavaType(foreign, ObjectTypeID)
	if JavaTypeAssignable(foreign, CharSequenceTypeID) || JavaTypeAssignable(foreign, StringTypeID) || JavaTypeAssignable(StringTypeID, foreign) {
		t.Fatal("nominal source owner borrowed a canonical String hierarchy")
	}
	RegisterJavaType(foreign, ObjectTypeID, CharSequenceTypeID)
	if !JavaTypeAssignable(foreign, CharSequenceTypeID) {
		t.Fatal("explicit source interface replacement was ignored")
	}
}

func TestCampaignAssignabilityArrayAndPrimitiveBoundaries(t *testing.T) {
	ints := ArrayTypeID(PrimitiveTypeID("int"))
	longs := ArrayTypeID(PrimitiveTypeID("long"))
	objects := ArrayTypeID(ObjectTypeID)
	strings := ArrayTypeID(StringTypeID)
	for _, test := range []struct {
		actual, expected TypeID
		want             bool
	}{
		{"", ObjectTypeID, false}, {ObjectTypeID, "", false},
		{PrimitiveTypeID("int"), PrimitiveTypeID("int"), true},
		{PrimitiveTypeID("int"), ObjectTypeID, false},
		{ObjectTypeID, PrimitiveTypeID("int"), false},
		{ints, ints, true}, {ints, longs, false},
		{ints, ObjectTypeID, true}, {ints, CloneableTypeID, true}, {ints, SerializableTypeID, true},
		{strings, objects, true}, {objects, strings, false},
		{ints, objects, false}, {ArrayTypeID(ints), objects, true},
		{strings, ArrayTypeID(objects), false},
	} {
		if got := JavaTypeAssignable(test.actual, test.expected); got != test.want {
			t.Fatalf("%q -> %q = %v, want %v", test.actual, test.expected, got, test.want)
		}
	}
}

func TestCampaignAssignabilityConcurrentRegistryReaders(t *testing.T) {
	child := TypeID("campaign.edges.concurrent.Child")
	base := TypeID("campaign.edges.concurrent.Base")
	left := TypeID("campaign.edges.concurrent.Left")
	right := TypeID("campaign.edges.concurrent.Right")
	absent := TypeID("campaign.edges.concurrent.Absent")
	RegisterJavaType(base, ObjectTypeID)
	RegisterJavaType(child, base, left)
	var workers sync.WaitGroup
	workers.Add(3)
	go func() {
		defer workers.Done()
		for index := 0; index < 2000; index++ {
			inter := left
			if index%2 != 0 {
				inter = right
			}
			RegisterJavaType(child, base, inter)
			RegisterJavaSourceType(child)
		}
	}()
	for reader := 0; reader < 2; reader++ {
		go func() {
			defer workers.Done()
			for index := 0; index < 2000; index++ {
				if !JavaTypeAssignable(child, base) || !JavaTypeAssignable(child, ObjectTypeID) || JavaTypeAssignable(child, absent) {
					t.Error("concurrent registry traversal violated stable superclass or absent-target facts")
					return
				}
			}
		}()
	}
	workers.Wait()
}

var campaignAssignabilityBoolSink bool
var campaignAssignabilityStreamSink Stream[*JavaString]

func BenchmarkCampaignAssignabilityEdges(b *testing.B) {
	base := TypeID("campaign.edges.bench.Base")
	child := TypeID("campaign.edges.bench.Child")
	inter := TypeID("campaign.edges.bench.Interface")
	grand := TypeID("campaign.edges.bench.Grand")
	absent := TypeID("campaign.edges.bench.Absent")
	cycleA := TypeID("campaign.edges.bench.CycleA")
	cycleB := TypeID("campaign.edges.bench.CycleB")
	RegisterJavaType(base, ObjectTypeID)
	RegisterJavaType(child, base, inter)
	RegisterJavaType(grand, child)
	RegisterJavaType(cycleA, cycleB)
	RegisterJavaType(cycleB, cycleA)
	for _, test := range []struct {
		name             string
		actual, expected TypeID
		want             bool
	}{
		{"directSuperclass", child, base, true},
		{"directInterface", child, inter, true},
		{"transitiveSuperclass", grand, ObjectTypeID, true},
		{"cyclicMiss", cycleA, absent, false},
		{"arrayCovariance", ArrayTypeID(child), ArrayTypeID(base), true},
	} {
		b.Run(test.name, func(b *testing.B) {
			if got := JavaTypeAssignable(test.actual, test.expected); got != test.want {
				b.Fatalf("initial assignability=%v, want %v", got, test.want)
			}
			b.ReportAllocs()
			b.ResetTimer()
			var answer bool
			for iteration := 0; iteration < b.N; iteration++ {
				answer = JavaTypeAssignable(test.actual, test.expected)
			}
			b.StopTimer()
			campaignAssignabilityBoolSink = answer
			if answer != test.want {
				b.Fatalf("assignability=%v, want %v", answer, test.want)
			}
		})
	}
	b.Run("stream128StringViews", func(b *testing.B) {
		array := NewReferenceArray(128, StringTypeID)
		text := NewJavaStringUTF16([]uint16{'A', 0xd800, 0, 0xdfff})
		for index := int32(0); index < 128; index++ {
			ReferenceArraySet(array, index, text)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for iteration := 0; iteration < b.N; iteration++ {
			campaignAssignabilityStreamSink = StreamOfArray[*JavaString](array)
		}
		b.StopTimer()
		values := campaignAssignabilityStreamSink.ToSlice()
		if len(values) != 128 {
			b.Fatalf("stream length=%d", len(values))
		}
		for _, value := range values {
			if value != text || !slices.Equal(value.UTF16Copy(), []uint16{'A', 0xd800, 0, 0xdfff}) {
				b.Fatal("view conversion changed identity or UTF16")
			}
		}
	})
}
