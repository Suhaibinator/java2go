package stdjava

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestListCanonicalDeclarationJDK21(t *testing.T) {
	rows := []struct {
		id        TypeID
		parent    string
		modifiers int32
	}{
		{"java.util.ArrayList", "java.util.AbstractList", 1},
		{"java.util.LinkedList", "java.util.AbstractSequentialList", 1},
		{"java.util.AbstractList", "java.util.AbstractCollection", 1025},
		{"java.util.AbstractSequentialList", "java.util.AbstractList", 1025},
		{"java.util.AbstractCollection", "java.lang.Object", 1025},
		{"java.util.List", "null", 1537},
		{"java.util.Collection", "null", 1537},
		{"java.util.Deque", "null", 1537},
		{"java.util.Queue", "null", 1537},
		{"java.util.RandomAccess", "null", 1537},
		{"java.util.SequencedCollection", "null", 1537},
		{"java.lang.Iterable", "null", 1537},
		{"java.lang.Cloneable", "null", 1537},
		{"java.io.Serializable", "null", 1537},
	}
	var errors strings.Builder
	for _, row := range rows {
		class := ClassLiteral(row.id)
		parent := "null"
		if super := class.GetSuperclass(); super != nil {
			parent = super.GetName()
		}
		if parent != row.parent || class.GetModifiers() != row.modifiers {
			fmt.Fprintf(&errors, "%s: parent=%s modifiers=%d want parent=%s modifiers=%d\n", row.id, parent, class.GetModifiers(), row.parent, row.modifiers)
		}
	}
	if errors.Len() != 0 {
		t.Fatal("canonical list declaration metadata differs from fresh JDK21:\n" + errors.String())
	}
}

func TestListCanonicalAllocationIdentity(t *testing.T) {
	element := new(int)
	source := NewListFrom[any](element)
	array := NewArrayList[any]()
	linked := NewLinkedList[any]()
	for _, row := range []struct {
		value *List[any]
		want  TypeID
	}{
		{array, "java.util.ArrayList"}, {NewArrayListWithArgument[any](int32(4)), "java.util.ArrayList"},
		{NewArrayListWithArgument[any](source), "java.util.ArrayList"}, {linked, "java.util.LinkedList"},
		{NewLinkedListFromCollection[any](source), "java.util.LinkedList"},
	} {
		actual, known := ObjectDynamicType(row.value)
		if !known || actual != row.want || ObjectGetClass(row.value) != ClassLiteral(row.want) {
			t.Fatalf("allocation metadata = %q/%v want %q", actual, known, row.want)
		}
		alias := row.value
		alias.Add(element)
		alias.Set(0, element)
		alias.Clear()
		if row.value.JavaDynamicTypeID() != row.want {
			t.Fatal("element operations changed allocation identity")
		}
	}
	arrayCopy := NewArrayListWithArgument[any](source)
	linkedCopy := NewLinkedListFromCollection[any](source)
	source.Set(0, new(int))
	if arrayCopy == source || linkedCopy == source || arrayCopy.Get(0) != element || linkedCopy.Get(0) != element {
		t.Fatal("copy storage or element reference identity changed")
	}
	if ObjectInstanceOf(array, "java.util.LinkedList") || ObjectInstanceOf(linked, "java.util.ArrayList") {
		t.Fatal("canonical allocation crossed concrete classes")
	}
	for _, id := range []TypeID{"java.util.List", "java.util.AbstractList", "java.util.Collection", "java.util.SequencedCollection", "java.lang.Iterable", "java.lang.Cloneable", "java.io.Serializable"} {
		if !ObjectInstanceOf(array, id) || !ObjectInstanceOf(linked, id) {
			t.Fatalf("missing inherited canonical contract %s", id)
		}
	}
	if !ObjectInstanceOf(array, "java.util.RandomAccess") || ObjectInstanceOf(linked, "java.util.RandomAccess") || !ObjectInstanceOf(linked, "java.util.Deque") || ObjectInstanceOf(array, "java.util.Deque") {
		t.Fatal("JDK allocation-specific interfaces changed")
	}
	metadataPanic(t, "IllegalArgumentException", func() { NewArrayListWithArgument[any](int32(-1)) })
	metadataPanic(t, "NullPointerException", func() { NewArrayListWithArgument[any](nil) })
	metadataPanic(t, "NullPointerException", func() { NewLinkedListFromCollection[any](nil) })
}

func TestListLegacyAllocationRemainsUnknown(t *testing.T) {
	source := NewArrayList[any]()
	source.Add("one")
	legacy := NewList[any]()
	values := []*List[any]{nil, {}, legacy, NewListFrom[any]("one"), NewListWithArgument[any](int32(3)), NewListWithArgument[any](source), newListWithCapacity[any](2), AsList[any]("one"), AsListArray[any](NewReferenceArray(0, ObjectTypeID), ObjectTypeID), source.SubList(0, 1), EmptyList[any](), SingletonList[any]("one"), UnmodifiableList(legacy)}
	for i, value := range values {
		if value.JavaDynamicTypeID() != "" {
			t.Fatalf("legacy factory/view %d fabricated concrete metadata", i)
		}
	}
	if source.JavaDynamicTypeID() != "java.util.ArrayList" {
		t.Fatal("view creation retagged backing allocation")
	}
}

func TestListDeclarationGraphAndLocation(t *testing.T) {
	rows := []struct {
		id, super TypeID
		direct    []TypeID
	}{
		{"java.util.ArrayList", "java.util.AbstractList", []TypeID{"java.util.List", "java.util.RandomAccess", "java.lang.Cloneable", "java.io.Serializable"}},
		{"java.util.LinkedList", "java.util.AbstractSequentialList", []TypeID{"java.util.List", "java.util.Deque", "java.lang.Cloneable", "java.io.Serializable"}},
		{"java.util.AbstractList", "java.util.AbstractCollection", []TypeID{"java.util.List"}},
		{"java.util.AbstractSequentialList", "java.util.AbstractList", nil},
		{"java.util.AbstractCollection", ObjectTypeID, []TypeID{"java.util.Collection"}},
		{"java.util.List", ObjectTypeID, []TypeID{"java.util.SequencedCollection"}},
		{"java.util.Collection", ObjectTypeID, []TypeID{"java.lang.Iterable"}},
		{"java.util.Deque", ObjectTypeID, []TypeID{"java.util.Queue", "java.util.SequencedCollection"}},
		{"java.util.Queue", ObjectTypeID, []TypeID{"java.util.Collection"}},
		{"java.util.RandomAccess", ObjectTypeID, nil},
		{"java.util.SequencedCollection", ObjectTypeID, []TypeID{"java.util.Collection"}},
		{"java.lang.Iterable", ObjectTypeID, nil}, {"java.lang.Cloneable", ObjectTypeID, nil}, {"java.io.Serializable", ObjectTypeID, nil},
	}
	for _, row := range rows {
		javaTypeRegistry.RLock()
		actual := javaTypeRegistry.types[row.id]
		javaTypeRegistry.RUnlock()
		if actual.super != row.super || !reflect.DeepEqual(actual.interfaces, row.direct) {
			t.Fatalf("%s graph %s/%v want %s/%v", row.id, actual.super, actual.interfaces, row.super, row.direct)
		}
		want := ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}
		if ClassRuntimeLocationOf(row.id) != want || ClassRuntimeLocationOf(ArrayTypeID(row.id)) != want {
			t.Fatalf("%s missing explicit JDK provenance", row.id)
		}
	}
}

func TestListDeclarationPreservesFullMetadata(t *testing.T) {
	calls := 0
	state := NewClassInitialization("java.util.ArrayList")
	before := ClassDescriptor{Type: "java.util.ArrayList", HasModifiers: true, Modifiers: 1, Initialization: state, Initialize: func(*Execution) { calls++ }, Construct: func(*Execution) any { return "instance" }, Fields: []FieldDescriptor{{Name: "member"}}, Methods: []MethodDescriptor{{Name: "method"}}, Constructors: []ConstructorDescriptor{{}}}
	after := listDeclarationDescriptor(before, listDeclarationFacts[0])
	if calls != 0 || after.Initialization != state || reflect.ValueOf(after.Initialize).Pointer() != reflect.ValueOf(before.Initialize).Pointer() || reflect.ValueOf(after.Construct).Pointer() != reflect.ValueOf(before.Construct).Pointer() || !reflect.DeepEqual(after.Fields, before.Fields) || !reflect.DeepEqual(after.Methods, before.Methods) || !reflect.DeepEqual(after.Constructors, before.Constructors) {
		t.Fatal("declaration flags replaced or initialized existing full metadata")
	}
	after.Initialize(NewExecution())
	if calls != 1 {
		t.Fatal("existing callback was lost")
	}
	for _, conflict := range []ClassDescriptor{{HasModifiers: true, Modifiers: 1025}, {HasModifiers: true, Modifiers: 1, Interface: true}} {
		failure := func() (failure any) {
			defer func() { failure = recover() }()
			listDeclarationDescriptor(conflict, listDeclarationFacts[0])
			return
		}()
		if failure != "conflicting canonical list declaration metadata" {
			t.Fatalf("metadata conflict accepted: %v", failure)
		}
	}
}

func TestListMetadataConcurrentLookup(t *testing.T) {
	array, linked := NewArrayList[any](), NewLinkedList[any]()
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 100 {
				if array.JavaDynamicTypeID() != "java.util.ArrayList" || linked.JavaDynamicTypeID() != "java.util.LinkedList" || ObjectGetClass(array) != ClassLiteral("java.util.ArrayList") || ClassRuntimeLocationOf("java.util.LinkedList").ModuleName != "java.base" {
					t.Error("immutable concurrent metadata changed")
					return
				}
			}
		}()
	}
	workers.Wait()
}
