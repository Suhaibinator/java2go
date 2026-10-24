package transpiler

import (
	"sync"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDeclaredBoundMetadataConcurrentReads(t *testing.T) {
	owners := []string{"java.lang.Class", "java.lang.reflect.TypeVariable", "java.lang.reflect.GenericDeclaration", "java.util.List", "java.util.Map"}
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for n := 0; n < 32; n++ {
				for _, owner := range owners {
					if _, ok := certifiedJDK21DeclarationBounds(owner); !ok {
						t.Errorf("missing immutable declaration %s", owner)
						return
					}
				}
			}
		}()
	}
	workers.Wait()
}

func TestDeclaredBoundProofSourceSlots(t *testing.T) {
	for _, r := range declaredBoundSourceControls {
		t.Run(r.name, func(t *testing.T) {
			a, f, ctx, _, _ := declaredBoundControlOrigins(t, declaredBoundControlSource(r.setup, r.actual, r.formal))
			want := declaredArgumentEquivalent
			if r.name == "lower-positive" {
				want = declaredArgumentUnknown
			}
			if got := declaredInvariantArgumentEquivalence(a, f, ctx); got != want {
				t.Fatalf("proof=%d want=%d", got, want)
			}
		})
	}
}

func TestDeclaredBoundProofNestedStrictness(t *testing.T) {
	a, f, ctx, _, _ := declaredBoundControlOrigins(t, declaredBoundControlSource("", "java.lang.reflect.TypeVariable<?>", "java.lang.reflect.TypeVariable<? extends java.lang.Class<?>>"))
	if got := declaredInvariantArgumentEquivalence(a.arguments[0], f.arguments[0], ctx); got != declaredArgumentDifferent {
		t.Fatalf("strict effective upper proof=%d", got)
	}
	class := f.arguments[0].arguments[0]
	class.wildcard = false
	declaration := inferredJavaTypeOrigin{externalName: "java.lang.reflect.GenericDeclaration", javaType: "java.lang.reflect.GenericDeclaration"}
	if got := jdk21DeclaredBoundSubtype(class, declaration); got != declaredBoundYes {
		t.Fatalf("class -> declaration=%d", got)
	}
	if got := jdk21DeclaredBoundSubtype(declaration, class); got != declaredBoundNo {
		t.Fatalf("declaration -> class=%d", got)
	}
}

func TestDeclaredBoundFactsActualIdentity(t *testing.T) {
	r, ok := certifiedJDK21DeclarationBounds("java.lang.reflect.TypeVariable")
	if !ok || len(r.parameters) != 1 || !r.parameters[0].classBoundAbsent || r.parameters[0].classBound != nil || len(r.parameters[0].interfaceBounds) != 1 || r.parameters[0].interfaceBounds[0].externalName != "java.lang.reflect.GenericDeclaration" {
		t.Fatal("actual interface-only bound lost")
	}
	if r.parameters[0].declaration != jdk21GenericParentRecords["java.lang.reflect.TypeVariable"].parameters[0] {
		t.Fatal("native declaration token changed")
	}
	c, ok := certifiedJDK21DeclarationBounds("java.lang.Class")
	if !ok || len(c.parameters) != 1 || c.parameters[0].classBound.externalName != "java.lang.Object" {
		t.Fatal("Class declaration bound must be independent of typed parent admission")
	}
	if _, ok := jdk21GenericParentRecords["java.lang.Class"]; ok {
		t.Fatal("held wildcard parent graph unexpectedly widened")
	}
	for _, kind := range []string{"certificate", "classSHA", "slot"} {
		altered := r
		altered.parameters = append([]jdk21GenericFormalBound{}, r.parameters...)
		switch kind {
		case "certificate":
			altered.certificate = "stale"
		case "classSHA":
			altered.classSHA = "stale"
		case "slot":
			altered.parameters[0].declaration = &symbol.TypeParamDeclaration{SourceName: "D"}
		}
		if _, ok := certifiedJDK21DeclarationBoundsWithRecords("java.lang.reflect.TypeVariable", map[string]jdk21GenericDeclarationBoundRecord{"java.lang.reflect.TypeVariable": altered}, jdk21GenericDeclarationBoundCertificates); ok {
			t.Fatalf("accepted altered %s", kind)
		}
	}
}

func TestDeclaredBoundProofUnknownShields(t *testing.T) {
	a, f, ctx, _, _ := declaredBoundControlOrigins(t, declaredBoundControlSource("interface Face{} class Cell<D extends Face>{}", "Cell<?>", "Cell<? extends Face>"))
	for _, kind := range []string{"lost-variance", "live-binder", "detached", "deep-cycle", "unresolved"} {
		t.Run(kind, func(t *testing.T) {
			actual, formal := a.arguments[0], f.arguments[0]
			switch kind {
			case "lost-variance":
				actual.javaType = "Cell<Object>"
			case "live-binder":
				actual.arguments = []inferredJavaTypeOrigin{{parameter: &symbol.TypeParamDeclaration{SourceName: "D"}, javaType: "D"}}
				actual.javaType = "Cell<D>"
			case "detached":
				clone := *actual.nominalScope
				actual.nominalScope = &clone
			case "deep-cycle":
				actual = inferredJavaTypeOrigin{externalName: "java.util.List", javaType: "java.util.List<java.util.List<?>>", arguments: make([]inferredJavaTypeOrigin, 1)}
				actual.arguments[0] = actual
			case "unresolved":
				actual.unresolved = true
			}
			if got := declaredInvariantArgumentEquivalence(actual, formal, ctx); got != declaredArgumentUnknown {
				t.Fatalf("%s proof=%d", kind, got)
			}
		})
	}
	view, ok := declaredArgumentProof(a.arguments[0], a.arguments[0].javaType, ctx, 0)
	if !ok {
		t.Fatal("valid proof absent")
	}
	for _, slot := range []int{-1, 1, 99} {
		if _, ok := declaredGenericFormalBounds(view, slot, ctx, 0); ok {
			t.Fatal("unknown formal slot accepted")
		}
	}
	recipe := &declaredBoundRecipe{kind: "nominal", externalName: "java.util.List"}
	recipe.arguments = []*declaredBoundRecipe{recipe}
	if _, ok := declaredBoundRecipeProof(recipe, nil, ctx, map[*declaredBoundRecipe]bool{}, 0); ok {
		t.Fatal("cyclic bound recipe admitted")
	}
	object := &declaredBoundRecipe{kind: "nominal", externalName: "java.lang.Object"}
	for _, recipe := range []*declaredBoundRecipe{{kind: "wildcard", variance: "unbounded"}, {kind: "wildcard", variance: "super", bound: object}, {kind: "array", arrayRank: 1, bound: object}} {
		if !validDeclaredBoundRecipeShape(recipe, map[*declaredBoundRecipe]bool{}, 0) {
			t.Fatal("retained complete recipe discarded")
		}
		if _, ok := declaredBoundRecipeProof(recipe, nil, ctx, map[*declaredBoundRecipe]bool{}, 0); ok {
			t.Fatal("unsupported complete recipe granted a proof")
		}
	}
	for _, recipe := range []*declaredBoundRecipe{{kind: "wildcard"}, {kind: "wildcard", variance: "super"}, {kind: "array", arrayRank: 256, bound: object}, {kind: "array", arrayRank: 1}} {
		if validDeclaredBoundRecipeShape(recipe, map[*declaredBoundRecipe]bool{}, 0) {
			t.Fatal("malformed recipe attested")
		}
	}
}

func TestDeclaredBoundClosedRelationUnknownIsNotNo(t *testing.T) {
	base := map[string]jdk21MemberOwner{"A": {certificate: "a", superclass: "B"}, "B": {certificate: "b"}, "C": {certificate: "c"}}
	if got := jdk21DeclaredBoundSubtypeWithRecords("A", "B", base, base); got != declaredBoundYes {
		t.Fatal(got)
	}
	if got := jdk21DeclaredBoundSubtypeWithRecords("A", "C", base, base); got != declaredBoundNo {
		t.Fatal(got)
	}
	for _, kind := range []string{"missing", "stale", "cycle", "target"} {
		rows := map[string]jdk21MemberOwner{}
		for k, v := range base {
			rows[k] = v
		}
		switch kind {
		case "missing":
			delete(rows, "B")
		case "stale":
			r := rows["B"]
			r.certificate = "stale"
			rows["B"] = r
		case "cycle":
			r := rows["B"]
			r.superclass = "A"
			rows["B"] = r
		case "target":
			delete(rows, "C")
		}
		proof := base
		if kind == "cycle" {
			proof = rows
		}
		if got := jdk21DeclaredBoundSubtypeWithRecords("A", "C", rows, proof); got != declaredBoundUnknown {
			t.Fatalf("%s=%d", kind, got)
		}
	}
}
