package transpiler

import (
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// These controls use parsed owner/method declarations and completed family
// plans. The original full-project compilation supplies the pre-fix RED.
func TestPrivateGenericHelperOwnerSlotsUseCompletedPhysicalFamily(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"object", `class Foo{} class Box<T>{private<R>R identity(R value){return value;}public Foo call(Foo value){return identity(value);}static<X>void demand(Box<X> box){}}`, "any"},
		{"number", `class Foo{} class Box<T extends Number>{private<R>R identity(R value){return value;}public Foo call(Foo value){return identity(value);}static<X extends Number>void demand(Box<X> box){}}`, "JavaNumber"},
		{"enum", `class Foo{} class Box<T extends java.lang.Enum>{private<R>R identity(R value){return value;}public Foo call(Foo value){return identity(value);}static<X extends java.lang.Enum>void demand(Box<X> box){}}`, "JavaEnum"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			owner := helper.File.Symbols.FindClassScope("Box")
			method := owner.FindMethodByName("identity", nil)
			caller := owner.FindMethodByName("call", nil)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = owner
			ctx.localScope = caller
			ctx.expectedType, ctx.expectedTypeRoot = "", nil
			ctx.importAliases = map[string]string{stdjavaImportPath: "physicalRuntime"}
			family := canonicalGenericFamily(owner, ctx)
			if family == nil {
				t.Fatal("source family must complete before physical helper owner arguments")
			}
			if _, member := family.members[owner]; !member {
				t.Fatal("original owner absent from completed family")
			}
			declaration := owner.TypeParameters[0].Declaration
			methodDeclaration := method.TypeParameters[0].Declaration
			if declaration == nil || methodDeclaration == nil || declaration == methodDeclaration {
				t.Fatal("owner and method binders collapsed")
			}
			invocation := findNode(caller.DeclarationNode, "method_invocation")
			logicalMethod := inferMethodTypeArguments(method, invocation, ctx, helper.File.Source)
			if len(logicalMethod) != 1 {
				t.Fatal("actual no-target private method R inference lost")
			}
			foo, ok := logicalMethod[0].(*ast.StarExpr)
			if !ok {
				t.Fatalf("R lost reference pointer: %T", logicalMethod[0])
			}
			fooDeclaration := helper.File.Symbols.FindClassScope("Foo")
			logicalBindings := genericJavaInvocationTypeBindings(method, invocation, ctx, helper.File.Source)
			if fooDeclaration == nil || resolveClassScopeByQualifiedName(ctx, logicalBindings["R"]) != fooDeclaration || method.Parameters[0].TypeParameterBindings["R"] != methodDeclaration || definitionDeclaredJavaType(method).TypeParameterBindings["R"] != methodDeclaration {
				t.Fatal("actual declaration-owned R inference lost source Foo identity")
			}
			if name, ok := foo.X.(*ast.Ident); !ok || name.Name != fooDeclaration.Class.Name {
				t.Fatalf("R no longer inferred Foo: %#v", foo.X)
			}
			logicalOwner := []ast.Expr{ast.NewIdent(owner.TypeParameters[0].EmittedName())}
			got := genericPrivateHelperOwnerTypeArguments(owner, method, logicalOwner, ctx)
			if len(got) != 1 || got[0] == logicalOwner[0] {
				t.Fatal("certified physical owner slot not selected")
			}
			if tc.want == "any" {
				id, ok := got[0].(*ast.Ident)
				if !ok || id.Name != "any" {
					t.Fatalf("Object physical slot=%#v", got[0])
				}
			} else {
				selector, ok := got[0].(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != tc.want {
					t.Fatalf("bounded physical slot=%#v", got[0])
				}
				alias, ok := selector.X.(*ast.Ident)
				if !ok || alias.Name != "physicalRuntime" {
					t.Fatal("caller runtime alias lost")
				}
			}
			if owner.TypeParameters[0].Declaration != declaration || method.TypeParameters[0].Declaration != methodDeclaration || owner.TypeParameters[0].Name != "T" || method.OriginalType != "R" || method.Parameters[0].OriginalType != "R" {
				t.Fatal("logical declarations/signatures changed")
			}
			if id, ok := logicalOwner[0].(*ast.Ident); !ok || id.Name != owner.TypeParameters[0].EmittedName() {
				t.Fatal("logical owner arguments mutated")
			}
		})
	}
}

func TestPrivateGenericHelperOwnerSlotsRefuseUncertifiedOrForeignOrigins(t *testing.T) {
	helper := setupParseHelper(t, `class Foo{} class Box<T>{private<R>R identity(R value){return value;}private static<R>R staticIdentity(R value){return value;}public<R>R publicIdentity(R value){return value;}static<X>void demand(Box<X> box){}}`)
	owner := helper.File.Symbols.FindClassScope("Box")
	method := owner.FindMethodByName("identity", nil)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = owner
	family := canonicalGenericFamily(owner, ctx)
	if family == nil {
		t.Fatal("control family did not complete")
	}
	logical := []ast.Expr{ast.NewIdent(owner.TypeParameters[0].EmittedName())}
	preserved := func(scope *symbol.ClassScope, definition *symbol.Definition, c Ctx) {
		t.Helper()
		got := genericPrivateHelperOwnerTypeArguments(scope, definition, logical, c)
		if len(got) != 1 || got[0] != logical[0] {
			t.Fatal("uncertified origin acquired physical arguments")
		}
	}
	copiedMethod := *method
	preserved(owner, &copiedMethod, ctx)
	copiedOwner := *owner
	preserved(&copiedOwner, method, ctx)
	preserved(owner, owner.FindMethodByName("staticIdentity", nil), ctx)
	preserved(owner, owner.FindMethodByName("publicIdentity", nil), ctx)
	preserved(nil, method, ctx)
	partial := *family
	partial.representations = make(map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation, len(family.representations))
	for declaration, representation := range family.representations {
		partial.representations[declaration] = representation
	}
	delete(partial.representations, owner.TypeParameters[0].Declaration)
	withoutRepresentation := ctx.Clone()
	withoutRepresentation.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{&partial}}
	preserved(owner, method, withoutRepresentation)
	short := genericPrivateHelperOwnerTypeArguments(owner, method, nil, ctx)
	if short != nil {
		t.Fatal("missing owner arity fabricated a physical slot")
	}
	// Incomplete declaration and inventory proofs must not be repaired by names.
	cached := ctx.Clone()
	cached.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{family}}
	oldOwnerNode := owner.Class.DeclarationNode
	owner.Class.DeclarationNode = nil
	preserved(owner, method, cached)
	owner.Class.DeclarationNode = oldOwnerNode
	oldMethodNode := method.DeclarationNode
	method.DeclarationNode = nil
	preserved(owner, method, cached)
	method.DeclarationNode = oldMethodNode
	oldDeclaration := owner.TypeParameters[0].Declaration
	owner.TypeParameters[0].Declaration = nil
	preserved(owner, method, cached)
	owner.TypeParameters[0].Declaration = oldDeclaration
	missingBinder := *family
	missingBinder.binders = make(map[*symbol.TypeParamDeclaration]struct{}, len(family.binders))
	for declaration := range family.binders {
		missingBinder.binders[declaration] = struct{}{}
	}
	delete(missingBinder.binders, oldDeclaration)
	withoutBinder := ctx.Clone()
	withoutBinder.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{&missingBinder}}
	preserved(owner, method, withoutBinder)
	oldInterface := owner.IsInterface
	owner.IsInterface = true
	preserved(owner, method, cached)
	owner.IsInterface = oldInterface
	t.Run("recorded-local-and-anonymous-copied-owner", func(t *testing.T) {
		parsed := setupParseHelper(t, `class Box<T>{private<R>R identity(R value){return value;}static<X>void demand(Box<X> box){}}interface Left<T>{T read();}interface Right<T>{T read();}class Use{<X>void make(){class Local implements Left<X>,Right<X>{public X read(){return null;}}}Left<String> anonymous(){return new Left<String>(){public String read(){return "ok";}};}}`)
		actual := parsed.File.Symbols.FindClassScope("Box")
		private := actual.FindMethodByName("identity", nil)
		syntheticCtx := parsed.Ctx.Clone()
		syntheticCtx.genericFamilies = &genericFamilyAnalysis{}
		plan, err := planGenericFamily(parsed.File.Symbols.FindClassScope("Left"), syntheticCtx)
		if err != nil {
			t.Fatal(err)
		}
		if plan == nil || len(plan.locals) != 1 || len(plan.anonymous) != 1 {
			t.Fatal("actual local/anonymous source events missing")
		}
		for kind, events := range map[string]map[genericFamilySourceKey]genericFamilyAnonymous{"local": plan.locals, "anonymous": plan.anonymous} {
			for _, event := range events {
				copied := *actual
				class := *actual.Class
				class.DeclarationNode = event.node
				copied.Class = &class
				copied.Enclosing = event.owner
				c := syntheticCtx.Clone()
				c.currentClass = &copied
				c.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{plan}}
				if kind == "anonymous" {
					// This source event carries no Box<T> owner binder. Its actual
					// emitted scope must retain the audited parent and carried list.
					if len(event.carried) != 0 || len(copied.TypeParameters) != 1 || !event.parent.IsInterface {
						t.Fatal("anonymous source event no longer distinguishes foreign owner metadata")
					}
					if canonicalGenericFamily(&copied, c) != nil {
						t.Fatal("anonymous event accepted foreign Box<T> owner metadata")
					}
					copied.TypeParameters = append([]symbol.TypeParam(nil), event.carried...)
					copied.Superclass = ""
					copied.ImplementedInterfaces = []string{event.javaType}
				} else if len(copied.TypeParameters) != len(actual.TypeParameters) {
					t.Fatal("local copied-owner control lost its full phantom owner arity")
				}
				// This source key really synthesizes membership. That membership
				// cannot certify this copied scope as the method's actual owner.
				synthesized := canonicalGenericFamily(&copied, c)
				if synthesized == nil {
					t.Fatalf("%s copy did not exercise source-key synthesis", kind)
				}
				if _, member := synthesized.members[&copied]; !member {
					t.Fatal("copy not in synthesized membership")
				}
				declaring, known := genericMethodOwnerContext(private)
				if !known || declaring.currentClass != actual || declaring.currentClass == &copied {
					t.Fatal("actual source graph owner identity lost")
				}
				arguments := []ast.Expr{ast.NewIdent(actual.TypeParameters[0].EmittedName())}
				got := genericPrivateHelperOwnerTypeArguments(&copied, private, arguments, c)
				if len(got) != 1 || got[0] != arguments[0] {
					t.Fatalf("%s copied scope acquired phantom physical owner slots", kind)
				}
			}
		}
	})

}

func TestPrivateGenericHelperOwnerSlotsKeepOrdinaryAndSourceBoundOwners(t *testing.T) {
	for _, source := range []string{
		`class Foo{} class Box<T>{java.util.List<T> values;private<R>R identity(R value){return value;}Foo call(Foo value){return identity(value);}}`,
		`class Base{} class Foo{} class Box<T extends Base>{private<R>R identity(R value){return value;}Foo call(Foo value){return identity(value);}static<X extends Base>void demand(Box<X> box){}}`,
	} {
		helper := setupParseHelper(t, source)
		owner := helper.File.Symbols.FindClassScope("Box")
		method := owner.FindMethodByName("identity", nil)
		ctx := helper.Ctx.Clone()
		ctx.currentClass = owner
		if canonicalGenericFamily(owner, ctx) != nil {
			t.Fatal("unsupported/ordinary source acquired completed family")
		}
		logical := []ast.Expr{ast.NewIdent(owner.TypeParameters[0].EmittedName())}
		got := genericPrivateHelperOwnerTypeArguments(owner, method, logical, ctx)
		if len(got) != 1 || got[0] != logical[0] {
			t.Fatal("ordinary owner type argument changed")
		}
		explicit := []ast.Expr{&ast.StarExpr{X: ast.NewIdent("Foo")}}
		actual := genericPrivateHelperOwnerTypeArguments(owner, method, explicit, ctx)
		if len(actual) != 1 || actual[0] != explicit[0] {
			t.Fatal("explicit nonfamily owner argument changed")
		}
	}
}

func TestPrivateGenericHelperOwnerSlotsKeepMethodShadowDeclaration(t *testing.T) {
	helper := setupParseHelper(t, `class Foo{} class Box<T>{private<T>T identity(T value){return value;}Foo call(Foo value){return identity(value);}static<X>void demand(Box<X> box){}}`)
	owner := helper.File.Symbols.FindClassScope("Box")
	method := owner.FindMethodByName("identity", nil)
	caller := owner.FindMethodByName("call", nil)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = owner
	ctx.localScope = caller
	ctx.expectedType, ctx.expectedTypeRoot = "", nil
	if canonicalGenericFamily(owner, ctx) == nil {
		t.Fatal("shadow control family did not complete")
	}
	declaration := method.TypeParameters[0].Declaration
	if declaration == nil || declaration == owner.TypeParameters[0].Declaration || method.Parameters[0].TypeParameterBindings["T"] != declaration || definitionDeclaredJavaType(method).TypeParameterBindings["T"] != declaration {
		t.Fatal("shadow method T declaration lost")
	}
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	bindings := genericJavaInvocationTypeBindings(method, invocation, ctx, helper.File.Source)
	logical := inferMethodTypeArguments(method, invocation, ctx, helper.File.Source)
	if bindings["T"] == "" || len(logical) != 1 {
		t.Fatal("actual shadow method inference lost")
	}
	foo, ok := logical[0].(*ast.StarExpr)
	if !ok {
		t.Fatal("shadow method T lost Foo pointer")
	}
	fooDeclaration := helper.File.Symbols.FindClassScope("Foo")
	if fooDeclaration == nil || resolveClassScopeByQualifiedName(ctx, bindings["T"]) != fooDeclaration {
		t.Fatal("shadow method T actual binding lost source Foo declaration")
	}
	name, ok := foo.X.(*ast.Ident)
	if !ok || name.Name != fooDeclaration.Class.Name {
		t.Fatal("shadow method T no longer Foo")
	}
	ownerArgs := []ast.Expr{ast.NewIdent(owner.TypeParameters[0].EmittedName())}
	physical := genericPrivateHelperOwnerTypeArguments(owner, method, ownerArgs, ctx)
	if len(physical) != 1 {
		t.Fatal("owner physical arity lost")
	}
	id, ok := physical[0].(*ast.Ident)
	if !ok || id.Name != "any" {
		t.Fatal("owner physical Object slot lost")
	}
	if method.TypeParameters[0].Declaration != declaration || method.Parameters[0].TypeParameterBindings["T"] != declaration || ownerArgs[0].(*ast.Ident).Name != owner.TypeParameters[0].EmittedName() {
		t.Fatal("logical metadata or caller arguments changed")
	}
}
