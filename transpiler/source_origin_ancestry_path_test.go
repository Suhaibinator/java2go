package transpiler

import (
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// These API controls deliberately include invalid synthetic hierarchies. They
// exercise refusal/termination separately from the unchanged Java integration
// parent, rather than treating synthetic sources as javac-valid programs.
func ancestryOriginFixture(t *testing.T, sources ...string) (map[string]*symbol.ClassScope, Ctx) {
	t.Helper()
	previous, previousIndex := symbol.GlobalScope, activeResolutionFiles
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	activeResolutionFiles = nil
	t.Cleanup(func() { symbol.GlobalScope, activeResolutionFiles = previous, previousIndex })
	scopes := map[string]*symbol.ClassScope{}
	var ctx Ctx
	for _, text := range sources {
		file := parsing.SourceFile{Name: "Ancestry.java", Source: []byte(text)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		f := file.ParseSymbols()
		symbol.AddSymbolsToPackage(f)
		if ctx.currentFile == nil {
			ctx.currentFile = f
		}
		var visit func(*symbol.ClassScope, string)
		visit = func(scope *symbol.ClassScope, prefix string) {
			if scope == nil || scope.Class == nil {
				return
			}
			name := prefix + scope.Class.OriginalName
			scopes[name] = scope
			for _, child := range scope.Subclasses {
				visit(child, name+".")
			}
		}
		prefix := ""
		if f.Package != "" {
			prefix = f.Package + "."
		}
		for _, scope := range f.TopLevelClasses {
			visit(scope, prefix)
		}
	}
	return scopes, ctx
}

func ancestryScope(t *testing.T, scopes map[string]*symbol.ClassScope, name string) *symbol.ClassScope {
	t.Helper()
	scope := scopes[name]
	if scope == nil {
		t.Fatalf("actual declaration %s absent", name)
	}
	return scope
}

func TestSourceOriginAncestryReorderedNestedAndBinderIdentity(t *testing.T) {
	scopes, ctx := ancestryOriginFixture(t, `
class Box<T> {}
class Base<A,B> {}
class Mid<X,Y> extends Base<Y,Box<X[]>> {}
class Child<P,Q> extends Mid<Q,P> {}
class Value {}
`)
	child := ancestryScope(t, scopes, "Child")
	base := ancestryScope(t, scopes, "Base")
	ctx.currentClass = child
	foreign := &symbol.TypeParamDeclaration{SourceName: "P", GoName: "P"}
	value := inferredJavaTypeOrigin{nominalScope: scopes["Value"], javaType: "Value"}
	carried := inferredJavaTypeOrigin{parameter: foreign, javaType: "P"}
	got := sourceClassOriginBindings(child, []inferredJavaTypeOrigin{carried, value}, base, ctx)
	if got[base.TypeParameters[0].Declaration].parameter != foreign {
		t.Fatal("reordered ancestor slot lost the actual foreign declaration identity")
	}
	nested := got[base.TypeParameters[1].Declaration]
	if nested.nominalScope != scopes["Box"] || len(nested.arguments) != 1 ||
		nested.arguments[0].nominalScope != scopes["Value"] || nested.arguments[0].arrayRank != 1 {
		t.Fatalf("nested/reordered array argument was not substituted exactly: %+v", nested)
	}
	if got[base.TypeParameters[0].Declaration].parameter == child.TypeParameters[0].Declaration {
		t.Fatal("actual method binder was recaptured by a same-spelled class binder")
	}
}

func TestSourceOriginAncestryInterfacesCyclesAndUnrelatedBranches(t *testing.T) {
	t.Run("interface", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `interface Face<T> {} interface Link<X> extends Face<X> {} class Child<Z> implements Link<Z> {}`)
		child, face := scopes["Child"], scopes["Face"]
		ctx.currentClass = child
		got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), face, ctx)
		if got[face.TypeParameters[0].Declaration].parameter != child.TypeParameters[0].Declaration {
			t.Fatal("interface ancestry lost exact declaration binding")
		}
	})
	t.Run("synthetic-cycle", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class A<T> extends B<T> {} class B<U> extends A<U> {} class Other<V> {}`)
		a, b := scopes["A"], scopes["B"]
		ctx.currentClass = a
		if _, reachable := sourceOriginAncestryPaths(a, scopes["Other"], ctx); reachable {
			t.Fatal("unrelated declaration admitted through a cycle")
		}
		got := sourceClassOriginBindings(a, classParameterTypeOrigins(a), b, ctx)
		if got[b.TypeParameters[0].Declaration].parameter != a.TypeParameters[0].Declaration {
			t.Fatal("reachable edge in synthetic cycle did not preserve its binder")
		}
	})
	t.Run("irrelevant-unresolved-branch", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `interface Face<T> {} interface Unknown<T> {} class Child<Z> implements Unknown<Missing>,Face<Z> {}`)
		child, face := scopes["Child"], scopes["Face"]
		ctx.currentClass = child
		paths, ok := sourceOriginAncestryPaths(child, face, ctx)
		if !ok || len(paths[child]) != 1 || paths[child][0].scope != face {
			t.Fatal("nominal proof retained an unrelated branch")
		}
		got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), face, ctx)
		if got[face.TypeParameters[0].Declaration].parameter != child.TypeParameters[0].Declaration {
			t.Fatal("unrelated unresolved argument poisoned a proved interface path")
		}
	})
}

func TestSourceOriginAncestryRejectsIncompleteCarriedAndNestedArguments(t *testing.T) {
	t.Run("partial-carried", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Item {} class Outer<T> { class Inner<U> {} } class ExactOuter extends Outer<Item> { class ExactInner extends Inner<Item> {} }`)
		inner, parent := scopes["ExactOuter.ExactInner"], scopes["Outer.Inner"]
		ctx.currentClass = inner
		if _, ok := sourceOriginAncestryPaths(inner, parent, ctx); !ok {
			t.Fatal("actual nominal member superclass was not reachable")
		}
		if got := sourceClassOriginBindings(inner, nil, parent, ctx); len(got) != 0 {
			t.Fatalf("partial own argument shifted into enclosing T: %+v", got)
		}
	})
	t.Run("unresolved-nested", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Box<T> {} class Base<T> {} class Child extends Base<Box<Missing>> {}`)
		ctx.currentClass = scopes["Child"]
		if got := sourceClassOriginBindings(scopes["Child"], nil, scopes["Base"], ctx); len(got) != 0 {
			t.Fatalf("known outer nominal hid an unresolved nested argument: %+v", got)
		}
	})
	t.Run("lexical-capture-positive", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Outer<T> { class Inner<U> {} class Child<V> extends Inner<V> {} }`)
		child, parent := scopes["Outer.Child"], scopes["Outer.Inner"]
		ctx.currentClass = child
		got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), parent, ctx)
		if len(got) != 2 {
			t.Fatalf("complete lexical capture lost slots: %+v", got)
		}
		for i, parameter := range parent.TypeParameters {
			if got[parameter.Declaration].parameter != child.TypeParameters[i].Declaration {
				t.Fatalf("captured slot %d lost its exact declaration", i)
			}
		}
	})
}

func TestSourceOriginAncestryPreservesKnownRawBounds(t *testing.T) {
	scopes, ctx := ancestryOriginFixture(t, `class Base<T extends Number> {} class Raw extends Base {}`)
	ctx.currentClass = scopes["Raw"]
	got := sourceClassOriginBindings(scopes["Raw"], nil, scopes["Base"], ctx)
	if got[scopes["Base"].TypeParameters[0].Declaration].externalName != "java.lang.Number" {
		t.Fatalf("known raw edge lost existing first-bound behavior: %+v", got)
	}
}

func TestSourceOriginAncestryValidDiamondAndQualifiedMember(t *testing.T) {
	t.Run("shared-interface-declaration", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `interface Face<T> {} interface Left<X> extends Face<X> {} interface Right<Y> extends Face<Y> {} class Child<Z> implements Left<Z>,Right<Z> {}`)
		child, face := scopes["Child"], scopes["Face"]
		ctx.currentClass = child
		got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), face, ctx)
		if got[face.TypeParameters[0].Declaration].parameter != child.TypeParameters[0].Declaration {
			t.Fatal("valid shared-ancestor diamond lost its declaration-owned argument")
		}
	})
	t.Run("qualified-member-header", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Outer<T> { class Base<U> {} class Child<V> extends Outer<T>.Base<V> {} }`)
		child, parent := scopes["Outer.Child"], scopes["Outer.Base"]
		ctx.currentClass = child
		got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), parent, ctx)
		if len(got) != 2 {
			t.Fatalf("qualified member header lost complete enclosing/own slots: %+v", got)
		}
		for i, parameter := range parent.TypeParameters {
			if got[parameter.Declaration].parameter != child.TypeParameters[i].Declaration {
				t.Fatalf("qualified member header slot %d lost exact binder", i)
			}
		}
	})
}

func TestSourceOriginAncestryUnknownDetachedAndLateLocal(t *testing.T) {
	t.Run("unknown-header", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Base<T> {} class Child extends Missing {}`)
		ctx.currentClass = scopes["Child"]
		if _, ok := sourceOriginAncestryPaths(scopes["Child"], scopes["Base"], ctx); ok {
			t.Fatal("unknown header received a nominal ancestry certificate")
		}
	})
	t.Run("detached-copy", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Base<T> {} class Child<X> extends Base<X> {}`)
		copyScope := *scopes["Child"]
		ctx.currentClass = &copyScope
		if _, ok := sourceOriginAncestryPaths(&copyScope, scopes["Base"], ctx); ok {
			t.Fatal("unregistered scope copy borrowed the donor file/tree certificate")
		}
	})
	t.Run("registered-late-local", func(t *testing.T) {
		_, ctx := ancestryOriginFixture(t, `class Host { void read() { class Local extends Base<String> {} } } class Base<T> {}`)
		host := ctx.currentFile.FindClassScope("Host")
		base := ctx.currentFile.FindClassScope("Base")
		if host == nil || base == nil {
			t.Fatal("actual source declarations missing")
		}
		ctx.currentClass, ctx.className = host, host.Class.Name
		ctx.localScope = host.FindMethodByName("read", nil)
		var declarations []ast.Decl
		ctx.hoistedDecls = &declarations
		ctx.localClasses = map[string]*localClassInfo{}
		ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{}
		hoistLocalClass(findNode(ctx.localScope.DeclarationNode, "class_declaration"), ctx.currentFile.Source, ctx)
		info := ctx.localClasses["Local"]
		if info == nil || findFileScopeForClassScope(info.scope, ctx) != nil ||
			!intrinsicLexicalDeclaringFile(info.scope, ctx.currentFile, ctx) {
			t.Fatal("actual late local was not independently certified")
		}
		ctx.currentClass = info.scope
		got := sourceClassOriginBindings(info.scope, nil, base, ctx)
		if got[base.TypeParameters[0].Declaration].externalName != "java.lang.String" {
			t.Fatalf("certified late local lost actual source superclass arguments: %+v", got)
		}
		ctx.localClasses = nil
		if _, ok := sourceOriginAncestryPaths(info.scope, base, ctx); ok {
			t.Fatal("missing late-local registry retained ancestry certificate")
		}
	})
}

func TestSourceOriginAncestrySourceAndBinderShields(t *testing.T) {
	t.Run("declaration-file-import", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t,
			`package wrong; public class Base<T> {}`,
			`package right; public class Base<T> {}`,
			`package child; import right.Base; public class Child<X> extends Base<X> {}`)
		child, parent := scopes["child.Child"], scopes["right.Base"]
		ctx.currentClass = child // ctx.currentFile deliberately belongs to wrong.Base.
		got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), parent, ctx)
		if got[parent.TypeParameters[0].Declaration].parameter != child.TypeParameters[0].Declaration {
			t.Fatal("caller file replaced actual declaration-owned import context")
		}
		if got := sourceClassOriginBindings(child, classParameterTypeOrigins(child), scopes["wrong.Base"], ctx); len(got) != 0 {
			t.Fatal("same-spelled foreign source declaration became an ancestor")
		}
	})
	t.Run("synthetic-binder-header", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Base<T> {} class Child<Base> extends Base {}`)
		ctx.currentClass = scopes["Child"]
		if _, ok := sourceOriginAncestryPaths(scopes["Child"], scopes["Base"], ctx); ok {
			t.Fatal("source binder header was reinterpreted as a nominal homonym")
		}
	})
	t.Run("on-demand-ambiguity", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t,
			`package a; public class Base<T> {}`,
			`package b; public class Base<T> {}`,
			`package child; import a.*; import b.*; public class Child<X> extends Base<X> {}`)
		ctx.currentClass = scopes["child.Child"]
		for _, parent := range []*symbol.ClassScope{scopes["a.Base"], scopes["b.Base"]} {
			if _, ok := sourceOriginAncestryPaths(scopes["child.Child"], parent, ctx); ok {
				t.Fatal("ambiguous source import supplied a nominal certificate")
			}
		}
	})
	t.Run("stale-graph-donor", func(t *testing.T) {
		scopes, ctx := ancestryOriginFixture(t, `class Base<T> {} class Child<X> extends Base<X> {}`)
		oldChild, oldBase := scopes["Child"], scopes["Base"]
		ctx.currentClass = oldChild
		ctx.sourceOwnership = newResolutionFileIndex()
		// A new graph may contain identical spellings, but owns none of these pointers.
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		if got := sourceClassOriginBindings(oldChild, classParameterTypeOrigins(oldChild), oldBase, ctx); len(got) != 0 {
			t.Fatal("stale donor declarations survived graph replacement")
		}
	})
}
