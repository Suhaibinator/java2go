package stdjava

import "testing"

func TestThreadLocalReferenceCanonicalAllocationMetadata(t *testing.T) {
	e := NewExecution()
	plain := NewThreadLocalReference()
	calls := 0
	marker := JavaStringFromHostUTF8("supplied")
	supplied := ThreadLocalWithInitialReference(NewSupplierFuncAdapter(func(got *Execution) *JavaString {
		if got != e {
			t.Fatal("metadata replaced initializer execution")
		}
		calls++
		return marker
	}))
	if plain.key == supplied.key || calls != 0 {
		t.Fatal("canonical factories reused a key or invoked Supplier eagerly")
	}
	for _, test := range []struct {
		name  string
		local *ThreadLocal[any]
		id    TypeID
	}{{"plain", plain, ThreadLocalTypeID}, {"supplied", supplied, threadLocalSuppliedTypeID}} {
		t.Run(test.name, func(t *testing.T) {
			var object any = test.local
			if actual, ok := ObjectDynamicType(object); !ok || actual != test.id {
				t.Fatal("Java allocation has wrong canonical descriptor")
			}
			if !ObjectInstanceOf(object, ThreadLocalTypeID) || !ObjectInstanceOf(object, ObjectTypeID) {
				t.Fatal("canonical allocation lost root/Object ancestry")
			}
			alias := ReferenceCheckCast[*ThreadLocal[any]](object, ThreadLocalTypeID)
			if alias != test.local || ObjectView[*ThreadLocal[any]](object, ObjectTypeID) != test.local || !JavaReferenceEqual(alias, object) || monitorIdentityFor(alias) != monitorIdentityFor(object) {
				t.Fatal("canonical casts changed allocation/key/monitor identity")
			}
			if ObjectGetClass(object) != ClassLiteral(test.id) || (ObjectGetClass(object) == ClassLiteral(ThreadLocalTypeID)) != (test.name == "plain") {
				t.Fatal("plain and supplied Java classes collapsed")
			}
			array := NewReferenceArrayOf[*ThreadLocal[any]](1, ThreadLocalTypeID)
			ReferenceArrayAssign[*ThreadLocal[any]](array, 0, alias, ThreadLocalTypeID)
			if ReferenceArrayGet[*ThreadLocal[any]](array, 0, ThreadLocalTypeID) != test.local {
				t.Fatal("canonical root array changed allocation identity")
			}
		})
	}
	if calls != 0 || plain.GetObject(e) != nil {
		t.Fatal("metadata observation invoked Supplier or plain Java default was not null")
	}
	first, second := supplied.GetObject(e), supplied.GetObject(e)
	if first != marker || second != marker || calls != 1 {
		t.Fatal("supplied Java metadata changed lazy cache")
	}
	supplied.SetObject(e, nil)
	supplied.Remove(e)
	if supplied.GetObject(e) != marker || calls != 2 || ObjectGetClass(supplied) != ClassLiteral(threadLocalSuppliedTypeID) {
		t.Fatal("cache mutation changed supplied allocation class")
	}
	if plain.referenceKind != threadLocalReferencePlain || supplied.referenceKind != threadLocalReferenceSupplied {
		t.Fatal("Java factory did not retain private allocation kind")
	}
}

func TestThreadLocalReferenceMetadataPreservesLegacyShortDescriptor(t *testing.T) {
	e := NewExecution()
	plain := NewThreadLocal[int32]()
	supplied := ThreadLocalWithInitial[string](func() string { return "legacy" })
	for _, value := range []any{plain, supplied} {
		if actual, ok := ObjectDynamicType(value); !ok || actual != "ThreadLocal" || ObjectGetClass(value) != ClassLiteral("ThreadLocal") || ObjectInstanceOf(value, ThreadLocalTypeID) {
			t.Fatal("legacy native Go factory descriptor/class changed")
		}
	}
	if ObjectView[*ThreadLocal[int32]](plain, "ThreadLocal") != plain || ObjectView[*ThreadLocal[string]](supplied, "ThreadLocal") != supplied {
		t.Fatal("legacy short descriptor cast lost allocation identity")
	}
	if plain.referenceKind != threadLocalReferenceLegacy || supplied.referenceKind != threadLocalReferenceLegacy || plain.Get(e) != 0 || supplied.Get(e) != "legacy" {
		t.Fatal("legacy zero kind or typed Get changed")
	}
	plain.Set(e, 17)
	if plain.Get(e) != 17 {
		t.Fatal("legacy typed Set changed")
	}
	plain.Remove(e)
	if plain.Get(e) != 0 {
		t.Fatal("legacy Remove/default changed")
	}
	var absent *ThreadLocal[any]
	if absent.JavaDynamicTypeID() != "ThreadLocal" {
		t.Fatal("legacy nil receiver metadata fallback changed")
	}
}

type threadLocalReferenceMetadataShadow struct{ *ObjectInfo }

func TestThreadLocalReferenceCanonicalMetadataRejectsSourceHomonym(t *testing.T) {
	const sourceID TypeID = "ThreadLocal"
	javaTypeRegistry.Lock()
	prior, present := javaTypeRegistry.types[sourceID]
	javaTypeRegistry.Unlock()
	t.Cleanup(func() {
		javaTypeRegistry.Lock()
		defer javaTypeRegistry.Unlock()
		if present {
			javaTypeRegistry.types[sourceID] = prior
		} else {
			delete(javaTypeRegistry.types, sourceID)
		}
	})
	RegisterJavaType(sourceID, ObjectTypeID)
	RegisterJavaSourceType(sourceID)
	source := &threadLocalReferenceMetadataShadow{}
	source.ObjectInfo = NewObjectInfo(sourceID, func(id TypeID) any {
		if id == sourceID || id == ObjectTypeID {
			return source
		}
		return nil
	})
	plain := NewThreadLocalReference()
	supplied := ThreadLocalWithInitialReference(NewSupplierFuncAdapter(func(*Execution) any { return nil }))
	if ObjectInstanceOf(source, ThreadLocalTypeID) || JavaTypeAssignable(sourceID, ThreadLocalTypeID) {
		t.Fatal("short source homonym acquired canonical native ancestry")
	}
	for _, local := range []*ThreadLocal[any]{plain, supplied} {
		if ObjectInstanceOf(local, sourceID) || JavaTypeAssignable(local.JavaDynamicTypeID(), sourceID) || ObjectGetClass(local) == ObjectGetClass(source) {
			t.Fatal("canonical allocation borrowed short source homonym identity")
		}
		if ReferenceCheckCast[*ThreadLocal[any]](local, ThreadLocalTypeID) != local {
			t.Fatal("source homonym interfered with canonical cast")
		}
	}
	if !CaughtAs(threadLocalReferenceObservedPanic(func() { ReferenceCheckCast[*threadLocalReferenceMetadataShadow](plain, sourceID) }), "ClassCastException") {
		t.Fatal("canonical native value passed unrelated source homonym cast")
	}
}
