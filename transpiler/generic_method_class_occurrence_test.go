package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/parsing"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func methodClassOccurrenceType(t *testing.T, node ast.Expr) string {
	t.Helper()
	var b bytes.Buffer
	if err := printer.Fprint(&b, token.NewFileSet(), node); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func methodClassOccurrenceOwner(t *testing.T, source string) (Ctx, *symbol.ClassScope, *symbol.Definition) {
	t.Helper()
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Owner.java": source})
	owner := findQualifiedSourceClass("p.Owner")
	if owner == nil || owner.Class == nil {
		t.Fatal("actual Owner declaration absent")
	}
	method := owner.FindMethodByName("echo", nil)
	if method == nil || method.DeclarationNode == nil {
		t.Fatal("actual echo method absent")
	}
	ctx = classScopeCtx(owner, ctx)
	ctx.localScope = method
	return ctx, owner, method
}

// Class<T>'s T describes its represented Java type, while its physical value is
// the same Class reference. The method T remains distinct from the owner T.
func TestGenericMethodClassOccurrenceEntryABI(t *testing.T) {
	ctx, owner, method := methodClassOccurrenceOwner(t, `package p;public interface Owner<T>{<T>T echo(Class<T> token,T value);}`)
	if len(owner.TypeParameters) != 1 || len(method.TypeParameters) != 1 || owner.TypeParameters[0].Declaration == method.TypeParameters[0].Declaration {
		t.Fatal("class and method binder declaration premise lost")
	}
	formals := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
	if len(formals) != 2 {
		t.Fatal("actual formal count changed")
	}
	typ, _ := nodeutil.JavaParameterNodes(formals[0])
	if typ == nil || !canonicalClassArgumentIndependent(method.Parameters[0].OriginalType, ctx, typ) {
		t.Fatal("actual canonical Class occurrence premise absent")
	}
	if !genericMethodHasErasedEntry(method) {
		t.Fatal("declaration-owned Class<T> formal has no erased method entry")
	}
	plan := planGenericMethodEntryABI(owner, method, ctx)
	if plan == nil || plan.owner != owner || plan.method != method || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != method.TypeParameters[0].Declaration {
		t.Fatal("shared EntryABI did not retain actual method declaration")
	}
	if got := methodClassOccurrenceType(t, plan.parameterTypes[0]); got != "*stdjava.Class" {
		t.Fatalf("Class physical formal=%s", got)
	}
	if got := methodClassOccurrenceType(t, plan.parameterTypes[1]); got != "any" {
		t.Fatalf("method binder formal=%s", got)
	}
	if got := methodClassOccurrenceType(t, plan.resultType); got != "any" {
		t.Fatalf("method binder result=%s", got)
	}
}

func TestGenericMethodClassOccurrenceSlotsAndRanks(t *testing.T) {
	for _, test := range []struct {
		typ  string
		rank int
	}{{"Class<T>", 0}, {"java.lang.Class<T>", 0}, {"Class<? super T>", 0}, {"Class<java.util.List<T>>", 0}, {"Class<T>[]", 1}, {"Class<T>[][]", 2}} {
		t.Run(test.typ, func(t *testing.T) {
			ctx, owner, method := methodClassOccurrenceOwner(t, `package p;public interface Owner{<T>T echo(`+test.typ+` token,T value);}`)
			if !canonicalGenericMethodClassSlot(method, 0, ctx) {
				t.Fatal("actual canonical Class formal declined")
			}
			plan := planGenericMethodEntryABI(owner, method, ctx)
			if plan == nil {
				t.Fatal("native Class slot did not reach shared EntryABI")
			}
			_, rank := javaArrayTypeParts(plan.parameterJavaTypes[0])
			if rank != test.rank {
				t.Fatalf("erased Java rank=%d want%d", rank, test.rank)
			}
			want := "*stdjava.Class"
			if test.rank > 0 {
				want = "*stdjava.ReferenceArray"
			}
			if got := methodClassOccurrenceType(t, plan.parameterTypes[0]); got != want {
				t.Fatalf("physical formal=%s want%s", got, want)
			}
		})
	}
	ctx, owner, method := methodClassOccurrenceOwner(t, `package p;public interface Owner{<T>Class<T> echo(Class<T> token);}`)
	if !canonicalGenericMethodClassSlot(method, -1, ctx) {
		t.Fatal("actual canonical Class result declined")
	}
	plan := planGenericMethodEntryABI(owner, method, ctx)
	if plan == nil || methodClassOccurrenceType(t, plan.resultType) != "*stdjava.Class" {
		t.Fatal("Class result did not retain native reference ABI")
	}
}

func TestGenericMethodClassOccurrenceOwnerAndSlotRefusals(t *testing.T) {
	const source = `package p;public interface Owner<T>{<T>T echo(Class<T> token,T value);interface Inner{<T>T echo(Class<T> token,T value);}}`
	for _, name := range []string{"copiedDefinition", "foreignMethod", "foreignOwner", "innerRebound", "formalType", "formalName", "resultType", "missingFormal", "missingBinding", "nonselectedBinding", "ownerMethodIdentity", "wrongBound", "missingMethodBinders", "index"} {
		t.Run(name, func(t *testing.T) {
			ctx, owner, method := methodClassOccurrenceOwner(t, source)
			foreign := parsing.SourceFile{Name: "foreign.java", Source: []byte(source)}
			if err := foreign.ParseAST(); err != nil {
				t.Fatal(err)
			}
			foreign.ParseSymbols()
			other := foreign.Symbols.FindClassScope("Owner")
			if other == nil {
				t.Fatal("foreign identical tree premise absent")
			}
			switch name {
			case "copiedDefinition":
				copy := *method
				method = &copy
			case "foreignMethod":
				method.DeclarationNode = other.FindMethodByName("echo", nil).DeclarationNode
			case "foreignOwner":
				owner.Class.DeclarationNode = other.Class.DeclarationNode
			case "innerRebound":
				inner := owner.FindClassScope("Inner")
				if inner == nil {
					t.Fatal("same-tree inner premise missing")
				}
				method = inner.FindMethodByName("echo", nil)
				inner.Methods = nil
				owner.Methods = []*symbol.Definition{method}
			case "formalType":
				method.Parameters[0].OriginalType = "Class<T>[]"
			case "formalName":
				method.Parameters[0].OriginalName = "other"
			case "resultType":
				method.OriginalType = "String"
			case "missingFormal":
				method.Parameters = method.Parameters[:1]
			case "missingBinding":
				method.Parameters[0].TypeParameterBindings = nil
			case "nonselectedBinding":
				method.Parameters[1].TypeParameterBindings = nil
			case "ownerMethodIdentity":
				identity := owner.TypeParameters[0].Declaration
				method.TypeParameters[0].Declaration = identity
				method.TypeParameterBindings["T"] = identity
				for _, p := range method.Parameters {
					p.TypeParameterBindings["T"] = identity
				}
			case "wrongBound":
				method.TypeParameters[0].Bounds = []symbol.JavaType{{Original: "Number"}}
			case "missingMethodBinders":
				method.TypeParameters = nil
			case "index":
				if canonicalGenericMethodClassSlot(method, -2, ctx) || canonicalGenericMethodClassSlot(method, 2, ctx) {
					t.Fatal("out-of-range slot admitted")
				}
				return
			}
			if canonicalGenericMethodClassSlot(method, 0, ctx) {
				t.Fatal("non-owned or incoherent method slot acquired Class authority")
			}
		})
	}
	// A Class witness does not certify a different parameter's counterfeit type.
	ctx, _, method := methodClassOccurrenceOwner(t, source)
	method.Parameters[1].OriginalType = "String"
	if canonicalGenericMethodClassSlot(method, 0, ctx) || canonicalGenericMethodSignature(method, ctx) {
		t.Fatal("nonselected formal metadata borrowed valid Class witness")
	}
}

func TestGenericMethodClassOccurrenceNamespaceAndDimensions(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"sourceClass", `package p;class Class<T>{T value;}public interface Owner{<T>T echo(Class<T> token,T value);}`},
		{"methodClassBinder", `package p;public interface Owner{<Class,T>T echo(Class<T> token,T value);}`},
		{"classClassBinder", `package p;public interface Owner<Class>{<T>T echo(Class<T> token,T value);}`},
		{"unknownImport", `package p;import foreign.Class;public interface Owner{<T>T echo(Class<T> token,T value);}`},
		{"qualifiedSourcePrefix", `package p;class java{static class lang{static class Class<T>{}}}public interface Owner{<T>T echo(java.lang.Class<T> token,T value);}`},
		{"invalidArity", `package p;public interface Owner{<T>T echo(Class<T,T> token,T value);}`},
		{"ordinaryContainer", `package p;class Box<T>{T value;}public interface Owner{<T>T echo(Box<T> token,T value);}`},
		{"postFormalDimensions", `package p;public interface Owner{<T>T echo(Class<T> token[],T value);}`},
		{"postResultDimensions", `package p;public interface Owner{<T>Class<T> echo(Class<T> token)[];}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, _, method := methodClassOccurrenceOwner(t, test.source)
			if canonicalGenericMethodClassSlot(method, 0, ctx) || canonicalGenericMethodClassSlot(method, -1, ctx) {
				t.Fatal("uncertified Class syntax acquired native slot authority")
			}
		})
	}
}

func TestGenericMethodClassOccurrenceExistingCallableGuards(t *testing.T) {
	for _, source := range []string{
		`package p;public class Owner{private <T>T echo(Class<T> token,T value){return value;}}`,
		`package p;public class Owner{static <T>T echo(Class<T> token,T value){return value;}}`,
		`package p;public interface Owner{<T>T echo(Class<T> token,T...values);}`,
		`package p;public interface Owner{<T extends Number & Runnable>T echo(Class<T> token,T value);}`,
		`package p;public interface Owner{<T extends Comparable<T>>T echo(Class<T> token,T value);}`,
	} {
		ctx, owner, method := methodClassOccurrenceOwner(t, source)
		if genericMethodHasErasedEntry(method) || planGenericMethodEntryABI(owner, method, ctx) != nil {
			t.Fatalf("Class witness changed an existing callable refusal: %s", source)
		}
	}
}

func TestGenericMethodClassOccurrenceProofIsPure(t *testing.T) {
	ctx, owner, method := methodClassOccurrenceOwner(t, `package p;public interface Owner<T>{<T>T echo(Class<T> token,T value);}`)
	ctx.importAliases = map[string]string{"keep": "alias"}
	ctx.usedImports = map[string]bool{"keep": true}
	names := []string{owner.Class.Name, method.Name, method.Parameters[0].Name, method.Parameters[1].Name, method.TypeParameters[0].EmittedName()}
	bindings := map[string]*symbol.TypeParamDeclaration{}
	for key, value := range method.Parameters[0].TypeParameterBindings {
		bindings[key] = value
	}
	graph := symbol.GlobalScope
	for i := 0; i < 25; i++ {
		if !canonicalGenericMethodClassSlot(method, 0, ctx) {
			t.Fatal("repeat proof changed admission")
		}
	}
	if graph != symbol.GlobalScope || !reflect.DeepEqual(ctx.importAliases, map[string]string{"keep": "alias"}) || !reflect.DeepEqual(ctx.usedImports, map[string]bool{"keep": true}) || !reflect.DeepEqual(bindings, method.Parameters[0].TypeParameterBindings) || !reflect.DeepEqual(names, []string{owner.Class.Name, method.Name, method.Parameters[0].Name, method.Parameters[1].Name, method.TypeParameters[0].EmittedName()}) {
		t.Fatal("pure slot proof mutated imports, names, bindings or registered graph")
	}
}

func TestGenericMethodClassAnonymousCallbackDescriptor(t *testing.T) {
	out := renderGoFileFromJava(t, `interface Poly{<T>T echo(Class<T> token,T value);}class Owner{Poly make(){return new Poly(){public <T>T echo(Class<T> token,T value){return value;}};}}`)
	file, err := goparser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "NewpolyFuncAdapterJava2goExecution" {
			return true
		}
		found++
		if len(call.Args) != 1 {
			t.Fatal("actual adapter callback argument count changed")
		}
		callback, ok := call.Args[0].(*ast.FuncLit)
		if !ok || len(callback.Type.Params.List) != 3 || callback.Type.Results == nil || len(callback.Type.Results.List) != 1 {
			t.Fatal("actual callback descriptor absent")
		}
		if got := methodClassOccurrenceType(t, callback.Type.Params.List[1].Type); got != "*stdjava.Class" {
			t.Fatalf("actual callback Class formal=%s", got)
		}
		if got := methodClassOccurrenceType(t, callback.Type.Params.List[2].Type); got != "any" {
			t.Errorf("anonymous callback method binder formal=%s, want any", got)
		}
		if got := methodClassOccurrenceType(t, callback.Type.Results.List[0].Type); got != "any" {
			t.Errorf("anonymous callback method binder result=%s, want any", got)
		}
		return true
	})
	if found != 1 {
		t.Fatalf("actual anonymous adapter calls=%d, want1; generated:\n%s", found, out)
	}
}

type methodClassAnonymousFixture struct {
	ctx                              Ctx
	creation, typ, body, declaration *sitter.Node
	implementation, selected         *symbol.Definition
}

func methodClassAnonymousSource(t *testing.T, sources map[string]string, ownerName string) methodClassAnonymousFixture {
	t.Helper()
	ctx := sourceGenericViewDemandTestContext(t, sources)
	owner := findQualifiedSourceClass(ownerName)
	if owner == nil {
		t.Fatal("source lexical owner missing")
	}
	ctx = classScopeCtx(owner, ctx)
	ctx.localScope = owner.FindMethodByName("make", nil)
	if ctx.localScope == nil {
		t.Fatal("source make declaration missing")
	}
	var creation *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || creation != nil {
			return
		}
		if n.Type() == "object_creation_expression" {
			for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
				if c.Type() == "class_body" {
					creation = n
					return
				}
			}
		}
		for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
			walk(c)
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if creation == nil {
		t.Fatal("actual anonymous creation missing")
	}
	typ := creation.ChildByFieldName("type")
	var body *sitter.Node
	for _, c := range nodeutil.SemanticNamedChildrenOf(creation) {
		if c.Type() == "class_body" {
			body = c
		}
	}
	methods := anonymousClassMethods(body)
	if len(methods) != 1 {
		t.Fatal("actual single implementation premise missing")
	}
	selected, _ := resolveFunctionalInterfaceMethod(ctx, typ.Content(ctx.currentFile.Source))
	if selected == nil {
		t.Fatal("actual selected source SAM absent")
	}
	implementation := scopeForAnonymousMethod(methods[0], selected, ctx.currentFile.Source)
	return methodClassAnonymousFixture{ctx, creation, typ, body, methods[0], implementation, selected}
}
func (f methodClassAnonymousFixture) proof() *genericInterfaceMethodBodyABI {
	return canonicalAnonymousGenericClassBody(f.creation, f.typ, f.body, f.declaration, f.implementation, f.selected, f.ctx)
}

func TestGenericMethodClassAnonymousCertificate(t *testing.T) {
	for _, typ := range []string{"Class<U>", "Class<U>[]", "Class<U>[][]", "Class<? super U>"} {
		t.Run(typ, func(t *testing.T) {
			contract := strings.ReplaceAll(typ, "U", "T")
			f := methodClassAnonymousSource(t, map[string]string{"p/Owner.java": `package p;interface Poly{<T>T echo(` + contract + ` token,T value);}class Owner{Poly make(){return new Poly(){public<U>U echo(` + typ + ` token,U value){return value;}};}}`}, "p.Owner")
			plan := f.proof()
			if plan == nil || plan.implementation != f.implementation || plan.entry.method != f.selected || plan.emissionOwner != f.ctx.currentClass {
				t.Fatal("attested anonymous source method did not receive its own body plan")
			}
			if f.implementation.TypeParameters[0].Declaration == f.selected.TypeParameters[0].Declaration {
				t.Fatal("anonymous and contract declaration identities aliased")
			}
		})
	}
}

func TestGenericMethodClassAnonymousOwnershipRefusals(t *testing.T) {
	const source = `package p;interface Poly{<T>T echo(Class<T> token,T value);}class Owner{Poly make(){return new Poly(){public<U>U echo(Class<U> token,U value){return value;}};}}class Other{}`
	for _, name := range []string{"foreignCreation", "foreignBody", "foreignMethod", "foreignType", "copiedSelected", "missingImplementationNode", "wrongNearestOwner", "wrongExecutable", "resultType", "nonselectedFormal", "formalName", "missingBinding", "directBinder", "contractIdentity", "postFormalRank", "postResultRank"} {
		t.Run(name, func(t *testing.T) {
			src := source
			if name == "postFormalRank" {
				src = strings.Replace(src, "echo(Class<U> token,U value)", "echo(Class<U> token[],U value)", 1)
			}
			if name == "postResultRank" {
				src = strings.Replace(src, "<U>U echo(Class<U> token,U value)", "<U>U echo(Class<U> token,U value)[]", 1)
			}
			f := methodClassAnonymousSource(t, map[string]string{"p/Owner.java": src}, "p.Owner")
			foreign := parsing.SourceFile{Name: "foreign.java", Source: f.ctx.currentFile.Source}
			if err := foreign.ParseAST(); err != nil {
				t.Fatal(err)
			}
			foreign.ParseSymbols()
			other := foreign.Symbols.FindClassScope("Owner")
			if other == nil {
				t.Fatal("foreign identical owner missing")
			}
			var creation *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n == nil || creation != nil {
					return
				}
				if n.Type() == "object_creation_expression" {
					creation = n
					return
				}
				for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
					walk(c)
				}
			}
			walk(other.Class.DeclarationNode)
			if creation == nil {
				t.Fatal("foreign identical creation missing")
			}
			var body *sitter.Node
			for _, c := range nodeutil.SemanticNamedChildrenOf(creation) {
				if c.Type() == "class_body" {
					body = c
				}
			}
			switch name {
			case "foreignCreation":
				f.creation = creation
			case "foreignBody":
				f.body = body
			case "foreignMethod":
				f.declaration = anonymousClassMethods(body)[0]
				f.implementation.DeclarationNode = f.declaration
			case "foreignType":
				f.typ = creation.ChildByFieldName("type")
			case "copiedSelected":
				copy := *f.selected
				f.selected = &copy
			case "missingImplementationNode":
				f.implementation.DeclarationNode = nil
			case "wrongNearestOwner":
				f.ctx.currentClass = findQualifiedSourceClass("p.Other")
			case "wrongExecutable":
				copy := *f.ctx.localScope
				f.ctx.localScope = &copy
			case "resultType":
				f.implementation.OriginalType = "String"
			case "nonselectedFormal":
				f.implementation.Parameters[1].OriginalType = "String"
			case "formalName":
				f.implementation.Parameters[1].OriginalName = "other"
			case "missingBinding":
				f.implementation.Parameters[1].TypeParameterBindings = nil
			case "directBinder":
				f.implementation.Parameters[1].DirectTypeParameter = f.selected.TypeParameters[0].Declaration
			case "contractIdentity":
				identity := f.selected.TypeParameters[0].Declaration
				f.implementation.TypeParameters[0].Declaration = identity
				f.implementation.TypeParameterBindings["U"] = identity
				for _, p := range f.implementation.Parameters {
					p.TypeParameterBindings["U"] = identity
				}
			}
			if f.proof() != nil {
				t.Fatal("unowned or incoherent anonymous declaration acquired body authority")
			}
		})
	}
}

func TestGenericMethodClassAnonymousCompleteOrigins(t *testing.T) {
	for _, name := range []string{"nominalCollision", "wildcardNominalCollision", "sourceClassShadow", "primitiveResult", "genericTarget", "classCapture", "methodCapture", "identityBody", "synchronizedBody", "statefulBody"} {
		t.Run(name, func(t *testing.T) {
			contract := `package p;public interface Poly{<T>T echo(Class<T> token,T value);}`
			source := `package q;import p.Poly;class Owner{Poly make(){return new Poly(){public<U>U echo(Class<U> token,U value){return value;}};}}`
			switch name {
			case "nominalCollision":
				contract = `package p;class T2{}public interface Poly{<T>T echo(Class<T> token,T2 nominal,T value);}`
				source = `package q;import p.Poly;class T2{}class Owner{Poly make(){return new Poly(){public<U>U echo(Class<U> token,T2 nominal,U value){return value;}};}}`
			case "wildcardNominalCollision":
				contract = `package p;class T2{}public interface Poly{<T>T echo(Class<T> token,Class<? super T2> bound,T value);}`
				source = `package q;import p.Poly;class T2{}class Owner{Poly make(){return new Poly(){public<U>U echo(Class<U> token,Class<? super T2> bound,U value){return value;}};}}`
			case "sourceClassShadow":
				source = `package q;import p.Poly;class Class<V>{}class Owner{Poly make(){return new Poly(){public<U>U echo(Class<U> token,U value){return value;}};}}`
			case "primitiveResult":
				source = strings.Replace(source, "public<U>U echo", "public<U>int echo", 1)
				source = strings.Replace(source, "return value;", "return 1;", 1)
			case "genericTarget":
				contract = `package p;public interface Poly<S>{<T>T echo(Class<T> token,T value);}`
				source = strings.ReplaceAll(source, "Poly make()", "Poly<String> make()")
				source = strings.ReplaceAll(source, "new Poly()", "new Poly<String>()")
			case "classCapture":
				source = strings.Replace(source, "class Owner{", "class Owner<S>{", 1)
			case "methodCapture":
				source = strings.Replace(source, "Poly make()", "<S>Poly make()", 1)
			case "identityBody":
				source = strings.Replace(source, "return value;", "this.hashCode();return value;", 1)
			case "synchronizedBody":
				source = strings.Replace(source, "public<U>", "public synchronized<U>", 1)
			case "statefulBody":
				source = strings.Replace(source, "new Poly(){", "new Poly(){int state;", 1)
			}
			f := methodClassAnonymousSource(t, map[string]string{"p/Poly.java": contract, "q/Owner.java": source}, "q.Owner")
			if f.proof() != nil {
				t.Fatal("missing declaration-origin or allocation proof acquired anonymous body authority")
			}
		})
	}
}

func TestGenericMethodClassAnonymousProofIsPure(t *testing.T) {
	f := methodClassAnonymousSource(t, map[string]string{"p/Owner.java": `package p;interface Poly{<T>T echo(Class<T> token,T value);}class Owner{Poly make(){return new Poly(){public<U>U echo(Class<U> token,U value){return value;}};}}`}, "p.Owner")
	f.ctx.importAliases = map[string]string{"keep": "alias"}
	f.ctx.usedImports = map[string]bool{"keep": true}
	counter := 13
	f.ctx.anonClassCounter = &counter
	graph := symbol.GlobalScope
	name := f.ctx.localScope.Name
	decl := f.implementation.TypeParameters[0].Declaration
	for i := 0; i < 25; i++ {
		if f.proof() == nil {
			t.Fatal("repeat proof changed admission")
		}
	}
	if graph != symbol.GlobalScope || counter != 13 || f.ctx.localScope.Name != name || f.implementation.TypeParameters[0].Declaration != decl || !reflect.DeepEqual(f.ctx.importAliases, map[string]string{"keep": "alias"}) || !reflect.DeepEqual(f.ctx.usedImports, map[string]bool{"keep": true}) {
		t.Fatal("anonymous proof mutated allocator/import/name/declaration/registry state")
	}
}

func TestGenericMethodClassAnonymousDistinctDeclarationRoles(t *testing.T) {
	const source = `package p;interface Poly{<A,B>A echo(Class<A> token,A value);}class Owner{Poly make(){return new Poly(){public<B,A>B echo(Class<B> token,B value){return value;}};}}class Other{<B>B other(B value){return value;}}`
	for _, role := range []string{"differentContractSlot", "unrelatedRegisteredMethod"} {
		t.Run(role, func(t *testing.T) {
			f := methodClassAnonymousSource(t, map[string]string{"p/Owner.java": source}, "p.Owner")
			fresh := f.implementation.TypeParameters[0].Declaration
			borrowed := f.selected.TypeParameters[1].Declaration
			if role == "unrelatedRegisteredMethod" {
				other := findQualifiedSourceClass("p.Other")
				if other == nil {
					t.Fatal("actual unrelated owner absent")
				}
				method := other.FindMethodByName("other", nil)
				if method == nil || len(method.TypeParameters) != 1 {
					t.Fatal("actual unrelated method binder absent")
				}
				borrowed = method.TypeParameters[0].Declaration
			}
			if fresh == nil || borrowed == nil || fresh == borrowed || borrowed.SourceName != "B" {
				t.Fatal("distinct same-spelling declaration-role premise absent")
			}
			f.implementation.TypeParameters[0].Declaration = borrowed
			members := append([]*symbol.Definition{f.implementation}, f.implementation.Parameters...)
			for _, member := range members {
				for key, identity := range member.TypeParameterBindings {
					if identity == fresh {
						member.TypeParameterBindings[key] = borrowed
					}
				}
				if member.DirectTypeParameter == fresh {
					member.DirectTypeParameter = borrowed
				}
			}
			if f.proof() != nil {
				t.Error("anonymous method borrowed a registered declaration role: " + role)
			}
		})
	}
}

func TestGenericMethodClassOccurrenceAnnotatedHeaderNames(t *testing.T) {
	ctx, _, method := methodClassOccurrenceOwner(t, `package p;
import java.lang.annotation.ElementType;
import java.lang.annotation.Target;
@Target(ElementType.TYPE_PARAMETER) @interface Mark {}
public interface Owner { <@Mark U,T> T echo(Class<T> token,T value,U other); }
`)
	params := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("type_parameters"))
	if len(params) != 2 || len(method.TypeParameters) != 2 {
		t.Fatal("actual annotated two-binder premise absent")
	}
	t.Logf("actual first syntax child kind=%s content=%s; modeled binders=%s,%s", params[0].NamedChild(0).Type(), params[0].NamedChild(0).Content(ctx.currentFile.Source), method.TypeParameters[0].Name, method.TypeParameters[1].Name)
	// The independently javac/javap-validated declaration names are U and T.
	coherent := method.TypeParameters[0].Name == "U" && method.TypeParameters[1].Name == "T"
	got := canonicalGenericMethodClassSlot(method, 0, ctx)
	if got != coherent {
		t.Fatalf("native Class admission=%t with actual annotated binder header coherent=%t", got, coherent)
	}
}

func TestGenericMethodClassOccurrenceAnnotatedBoundHeader(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/OwnerBound.java": `package p;
import java.lang.annotation.ElementType;
import java.lang.annotation.Target;
@Target(ElementType.TYPE_USE) @interface TypeMark {}
public interface OwnerBound { <T extends @TypeMark Number> T echo(Class<T> token,T value); }
`})
	owner := findQualifiedSourceClass("p.OwnerBound")
	if owner == nil {
		t.Fatal("actual annotated bound owner absent")
	}
	method := owner.FindMethodByName("echo", nil)
	if method == nil || len(method.TypeParameters) != 1 {
		t.Fatal("actual annotated bound method absent")
	}
	ctx = classScopeCtx(owner, ctx)
	ctx.localScope = method
	t.Logf("modeled method binder=%s bounds=%v; actual header=%s", method.TypeParameters[0].Name, method.TypeParameters[0].Bounds, method.DeclarationNode.ChildByFieldName("type_parameters").Content(ctx.currentFile.Source))
	if canonicalGenericMethodClassSlot(method, 0, ctx) || genericMethodHasErasedEntry(method) {
		t.Fatal("native Class admission trusted unsupported annotated bound header")
	}
}

func TestGenericMethodClassAnonymousPreservesCarriedOrigin(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Owner.java": `package p;
class Outer<A>{ class Inner<B>{} }
class Parent extends Outer<String>{}
public class Owner extends Parent { public <T> void observe(Inner<T> value){} }
`})
	owner := findQualifiedSourceClass("p.Owner")
	if owner == nil {
		t.Fatal("actual carried owner absent")
	}
	method := owner.FindMethodByName("observe", nil)
	if method == nil || len(method.Parameters) != 1 {
		t.Fatal("actual carried formal absent")
	}
	ctx = classScopeCtx(owner, ctx)
	ctx.localScope = method
	typ := definitionDeclaredJavaType(method.Parameters[0])
	before := declaredJavaTypeOrigin(typ, ctx)
	after := anonymousClassMethodDeclaredOrigin(typ, ctx)
	t.Logf("source=%s before carried arguments=%d unresolved=%t after arguments=%d", typ.Original, len(before.arguments), before.unresolved, len(after.arguments))
	if !sourceOriginAncestryComplete(before) || len(before.arguments) != 2 || before.arguments[0].externalName != "java.lang.String" || before.arguments[1].parameter != method.TypeParameters[0].Declaration {
		t.Fatal("actual complete String plus methodT carried-origin premise absent")
	}
	if !sameJavaTypeOrigin(before, after) {
		t.Fatal("anonymous signature proof dropped actual carried origin")
	}
}

func TestGenericMethodClassAnonymousDistinctCarriedOrigins(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Owner.java": `package p;class Outer<A>{class Inner<B>{}}class Owner extends Outer<String>{<T>void observe(Inner<T> value){}}class Other extends Outer<Integer>{<T>void observe(Inner<T> value){}}`})
	owner := findQualifiedSourceClass("p.Owner")
	other := findQualifiedSourceClass("p.Other")
	if owner == nil || other == nil {
		t.Fatal("actual distinct concrete owner declarations absent")
	}
	leftMethod := owner.FindMethodByName("observe", nil)
	rightMethod := other.FindMethodByName("observe", nil)
	if leftMethod == nil || rightMethod == nil {
		t.Fatal("actual declaration-owned formals absent")
	}
	leftCtx := classScopeCtx(owner, ctx)
	leftCtx.localScope = leftMethod
	rightCtx := classScopeCtx(other, ctx)
	rightCtx.localScope = rightMethod
	left := anonymousClassMethodDeclaredOrigin(definitionDeclaredJavaType(leftMethod.Parameters[0]), leftCtx)
	right := anonymousClassMethodDeclaredOrigin(definitionDeclaredJavaType(rightMethod.Parameters[0]), rightCtx)
	right = substituteJavaTypeOrigin(right, map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{rightMethod.TypeParameters[0].Declaration: {parameter: leftMethod.TypeParameters[0].Declaration, javaType: "T"}})
	if sameJavaTypeOrigin(left, right) {
		t.Fatal("anonymous signature proof collapsed distinct String/Integer enclosing origins")
	}
}
