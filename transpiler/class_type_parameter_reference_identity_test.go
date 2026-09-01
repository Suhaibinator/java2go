package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func sourceScopeNamed(t *testing.T, root *symbol.ClassScope, name string) *symbol.ClassScope {
	t.Helper()
	var found *symbol.ClassScope
	visitClassScopes(root, func(scope *symbol.ClassScope) bool {
		if scope != nil && scope.Class != nil && scope.Class.OriginalName == name {
			found = scope
			return true
		}
		return false
	})
	if found == nil {
		t.Fatalf("source scope %q was not parsed", name)
	}
	return found
}

func TestReferenceIdentityScopes_ClassTypeParameterABISitesSeedAssignableHierarchies(t *testing.T) {
	helper := setupParseHelper(t, `
public class ErasedIdentitySeeds {
    interface FieldBound {}
    static class FieldLeft implements FieldBound {}
    static class FieldLeftLeaf extends FieldLeft {}
    static class FieldRight implements FieldBound {}
    static class FieldRightLeaf extends FieldRight {}

    interface ParameterBound {}
    static class ParameterImpl implements ParameterBound {}
    static class ParameterLeaf extends ParameterImpl {}

    static class ResultBase {}
    static class ResultLeft extends ResultBase {}
    static class ResultRight extends ResultBase {}

    static class FieldHolder<T extends FieldBound> {
        T value;
    }

    static class ParameterHolder<T extends ParameterBound> {
        void accept(T value) {}
    }

    static class ResultHolder<T extends ResultBase> {
        T read() { return null; }
    }

    static class UnrelatedBase {}
    static class UnrelatedLeaf extends UnrelatedBase {}
}
`)

	relevant := referenceIdentityScopes(helper.Ctx)
	requireRelevant := func(name string, wantObjectInfo bool) {
		t.Helper()
		scope := sourceScopeNamed(t, helper.File.Symbols.BaseClass, name)
		if _, ok := relevant[scope]; !ok {
			t.Fatalf("%s was not seeded by its class type-parameter ABI use", name)
		}
		if got := classNeedsReferenceObjectInfo(scope, helper.Ctx); got != wantObjectInfo {
			t.Fatalf("%s ObjectInfo requirement = %t, want %t", name, got, wantObjectInfo)
		}
	}

	// Both sibling implementations of an interface bound and their concrete
	// descendants need views; neither sibling may be selected based on which one
	// happens to appear first in source.
	for _, name := range []string{"FieldLeft", "FieldLeftLeaf", "FieldRight", "FieldRightLeaf"} {
		requireRelevant(name, true)
	}
	// A direct method parameter is an erased class-parameter ABI site too.
	for _, name := range []string{"ParameterImpl", "ParameterLeaf"} {
		requireRelevant(name, true)
	}
	// A concrete first bound must seed every assignable subclass, including
	// siblings whose names never occur in the generic holder declaration.
	for _, name := range []string{"ResultBase", "ResultLeft", "ResultRight"} {
		requireRelevant(name, true)
	}

	for _, name := range []string{"UnrelatedBase", "UnrelatedLeaf"} {
		scope := sourceScopeNamed(t, helper.File.Symbols.BaseClass, name)
		if _, ok := relevant[scope]; ok {
			t.Fatalf("unrelated hierarchy %s was over-instrumented", name)
		}
	}
}

func TestReferenceIdentityScopes_UnboundedClassTypeParameterSeedsEveryConcreteClass(t *testing.T) {
	helper := setupParseHelper(t, `
public class UnboundedErasedIdentitySeeds {
    static class Holder<T> {
        T value;
        T read() { return value; }
        void accept(T next) { value = next; }
    }

    interface Marker {}
    static class Standalone implements Marker {}
    static class Root {}
    static class Leaf extends Root {}
}
`)

	relevant := referenceIdentityScopes(helper.Ctx)
	for _, name := range []string{"UnboundedErasedIdentitySeeds", "Holder", "Standalone", "Root", "Leaf"} {
		scope := sourceScopeNamed(t, helper.File.Symbols.BaseClass, name)
		if scope.IsInterface || scope.IsAbstract {
			t.Fatalf("test setup expected concrete scope %s", name)
		}
		if _, ok := relevant[scope]; !ok {
			t.Fatalf("unbounded Object erasure did not seed concrete class %s", name)
		}
	}

	// Standalone needs only lightweight nominal identity, while the class
	// hierarchy needs shared ObjectInfo to recover exact superclass/subclass views.
	standalone := sourceScopeNamed(t, helper.File.Symbols.BaseClass, "Standalone")
	if classNeedsReferenceObjectInfo(standalone, helper.Ctx) {
		t.Fatal("standalone leaf unexpectedly requires hierarchy ObjectInfo")
	}
	for _, name := range []string{"Root", "Leaf"} {
		scope := sourceScopeNamed(t, helper.File.Symbols.BaseClass, name)
		if !classNeedsReferenceObjectInfo(scope, helper.Ctx) {
			t.Fatalf("unbounded Object erasure did not give %s shared ObjectInfo", name)
		}
	}
}

func TestReferenceIdentityScopes_MethodTypeParameterShadowDoesNotSeedClassErasure(t *testing.T) {
	helper := setupParseHelper(t, `
public class ShadowedErasedIdentitySeed<T extends ClassBound> {
    interface ClassBound {}
    interface MethodBound {}
    static class MethodImpl implements MethodBound {}
    static class MethodLeaf extends MethodImpl {}

    <T extends MethodBound> T echo(T value) { return value; }
}
`)

	relevant := referenceIdentityScopes(helper.Ctx)
	for _, name := range []string{"ClassBound", "MethodBound", "MethodImpl", "MethodLeaf"} {
		scope := sourceScopeNamed(t, helper.File.Symbols.BaseClass, name)
		if _, ok := relevant[scope]; ok {
			t.Fatalf("method-shadowed type parameter incorrectly seeded %s", name)
		}
	}
}
