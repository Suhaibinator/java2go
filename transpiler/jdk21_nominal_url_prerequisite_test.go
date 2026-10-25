package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

const url701LateOwnerJava = `import java.net.URL;
interface URLView { URL read(URL value); }
class Host {
 static class Nested {}
 URLView build(){return new URLView(){Object marker;public URL read(URL value){return value;}};}
}`

// The historical URL181 prerequisite admitted a reconstructed context without
// a method/frame. These new tests use only the current real producer fixture.
// They never restore the superseded lexical-ancestry-only admission shortcut.
func TestJDK21NominalLiveLexicalOwnerAdmission(t *testing.T) {
	for name, source := range map[string]string{
		"anonymous":        url701LateOwnerJava,
		"field":            `import java.net.URL;interface URLView{URL read(URL value);}class Host{static URLView view=new URLView(){Object marker;public URL read(URL value){return value;}};}`,
		"local":            `import java.net.URL;interface URLView{URL read(URL value);}class Host{URLView build(){class Local implements URLView{public URL read(URL value){return value;}}return new Local();}}`,
		"nested_synthetic": `import java.net.URL;interface URLView{URL read(URL value);}interface Maker{URLView make();}class Host{Maker build(){return new Maker(){Object marker;public URLView make(){return new URLView(){Object marker;public URL read(URL value){return value;}};}};}}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, frame, _ := lateSourceOwnershipFixture(t, source)
			if findFileScopeForClassScope(ctx.currentClass, ctx) != nil || !frame.active || !jdk21NominalDeclaringUseOwned(ctx, "URL") || declaredExternalNominalOwner("URL", nil, ctx) != "java.net.URL" {
				t.Fatal("actual source method/frame did not own imported URL")
			}
			if name == "anonymous" {
				ordinary := ctx.currentClass.Enclosing
				query := ctx.Clone()
				query.currentClass, query.localScope = ordinary, nil
				if ordinary == nil || !jdk21NominalDeclaringUseOwned(query, "URL") {
					t.Fatal("ordinary source route changed")
				}
				var nested *symbol.ClassScope
				for _, scope := range ordinary.Subclasses {
					if scope.Class.OriginalName == "Nested" {
						nested = scope
					}
				}
				query.currentClass = nested
				if nested == nil || !jdk21NominalDeclaringUseOwned(query, "URL") {
					t.Fatal("ordinary named nested source route changed")
				}
			}
		})
	}
}

func TestJDK21NominalLiveLexicalOwnerRefusals(t *testing.T) {
	for _, name := range []string{"nil_scope", "missing_file", "missing_method", "missing_frame", "closed_frame", "copied_scope", "missing_registry", "wrong_registry_pointer", "copied_file", "missing_node", "missing_enclosing", "cycle", "cycle_after_actual_owner", "foreign_original_root", "detached_equal_span_node_with_forged_registry", "stale_graph"} {
		t.Run(name, func(t *testing.T) {
			// Every negative starts from a fresh genuine admitted caller, preventing a
			// previous stored-witness retirement from making later checks vacuous.
			ctx, frame, _ := lateSourceOwnershipFixture(t, url701LateOwnerJava)
			if !jdk21NominalDeclaringUseOwned(ctx, "URL") {
				t.Fatal("genuine negative-control parent unavailable")
			}
			scope := ctx.currentClass
			query := ctx.Clone()
			switch name {
			case "nil_scope":
				query.currentClass = nil
			case "missing_file":
				query.currentFile = nil
			case "missing_method":
				query.localScope = nil
			case "missing_frame":
				query.lateSourceOwnership = nil
			case "closed_frame":
				frame.close()
			case "copied_scope":
				copied := *scope
				query.currentClass = &copied
			case "missing_registry":
				query.anonymousClasses = nil
			case "wrong_registry_pointer":
				copied := *scope
				key, ok := anonymousClassSourceKey(scope.Class.DeclarationNode)
				if !ok {
					t.Fatal("actual source key unavailable")
				}
				query.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{key: {scope: &copied}}
			case "copied_file":
				copied := *ctx.currentFile
				query.currentFile = &copied
			case "missing_node":
				original := scope.Class.DeclarationNode
				scope.Class.DeclarationNode = nil
				defer func() { scope.Class.DeclarationNode = original }()
			case "missing_enclosing":
				original := scope.Enclosing
				scope.Enclosing = nil
				defer func() { scope.Enclosing = original }()
			case "cycle":
				original := scope.Enclosing
				scope.Enclosing = scope
				defer func() { scope.Enclosing = original }()
			case "cycle_after_actual_owner":
				outer := scope.Enclosing
				original := outer.Enclosing
				outer.Enclosing = scope
				defer func() { outer.Enclosing = original }()
			case "foreign_original_root", "detached_equal_span_node_with_forged_registry":
				other := parsing.SourceFile{Name: "Detached.java", Source: []byte(url701LateOwnerJava)}
				if err := other.ParseAST(); err != nil {
					t.Fatal(err)
				}
				if name == "foreign_original_root" {
					query.currentFile = other.ParseSymbols()
				} else {
					creation := findNode(other.Ast, "object_creation_expression")
					copiedScope, copiedClass := *scope, *scope.Class
					copiedClass.DeclarationNode = creation
					copiedScope.Class = &copiedClass
					query.currentClass = &copiedScope
					key, ok := anonymousClassSourceKey(creation)
					if !ok {
						t.Fatal("detached source key unavailable")
					}
					query.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{key: {scope: &copiedScope}}
				}
			case "stale_graph":
				graph := symbol.GlobalScope
				symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
				defer func() { symbol.GlobalScope = graph }()
			}
			if jdk21NominalDeclaringUseOwned(query, "URL") {
				t.Fatal("unproved source owner acquired native URL identity", name)
			}
			if name == "missing_registry" || name == "wrong_registry_pointer" {
				if !frame.active || !jdk21NominalDeclaringUseOwned(ctx, "URL") {
					t.Fatal("foreign query poisoned real caller")
				}
			}
		})
	}
}

func TestJDK21NominalLiveLexicalBinderAndSourceShields(t *testing.T) {
	for name, source := range map[string]string{
		"class_binder":  `import java.net.URL;interface View<Value>{Value read(Value value);}class Host<URL>{View<URL> build(){return new View<URL>(){Object marker;public URL read(URL value){return value;}};}}`,
		"method_binder": `import java.net.URL;interface View<Value>{Value read(Value value);}class Host{<URL>View<URL> build(){return new View<URL>(){Object marker;public URL read(URL value){return value;}};}}`,
		"source":        `class URL{}interface View{URL read(URL value);}class Host{View build(){return new View(){Object marker;public URL read(URL value){return value;}};}}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, _, _ := lateSourceOwnershipFixture(t, source)
			if declaredExternalNominalOwner("URL", symbol.VisibleTypeParamBindings(ctx.currentClass.TypeParameters), ctx) != "" {
				t.Fatal("source or actual captured binder acquired native URL identity")
			}
		})
	}
}
