package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

const hiddenLexicalReceiverSource = `interface Marker { int mark(); }
class Base implements Marker { public int mark(){return 7;} }
class Child extends Base {}
class T {}
class Carrier<T> {
 class Cell<U> {}
 int implicit(Cell<Integer> value){return 1;} long implicit(Object value){return 2L;}
 int explicit(Carrier<T>.Cell<Integer> value){return 3;} long explicit(Object value){return 4L;}
 <T extends Marker> int shadow(Cell<T> value){return 5;} long shadow(Object value){return 6L;}
 <T extends Marker> int written(Carrier<T>.Cell<T> value){return 7;}
}`

func TestHiddenLexicalReceiverSlotDeclarationIdentity(t *testing.T) {
	helper := setupParseHelper(t, hiddenLexicalReceiverSource)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	if owner == nil || len(owner.Subclasses) != 1 {
		t.Fatal("owner/member declarations missing")
	}
	member := owner.Subclasses[0]
	method := owner.FindMethodByName("shadow", nil)
	written := owner.FindMethodByName("written", nil)
	if method == nil || written == nil {
		t.Fatal("method declarations missing")
	}
	classParameter, methodParameter := owner.TypeParameters[0], method.TypeParameters[0]
	if member.TypeParameters[0].Declaration != classParameter.Declaration {
		t.Fatal("hidden slot lost owning declaration")
	}
	if methodParameter.Declaration == classParameter.Declaration {
		t.Fatal("same-named method/class declarations collapsed")
	}
	if method.Parameters[0].TypeParameterBindings["T"] != methodParameter.Declaration {
		t.Fatal("written Cell<T> did not bind method T")
	}
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, method
	args := sourceClassGoTypeArgumentExprs(member, []string{"T"}, owner, nil, inScopeTypeParameters(ctx), ctx)
	want := []string{classParameter.EmittedName(), methodParameter.EmittedName()}
	got := []string{boundedArgumentTestGoType(t, args[0]), boundedArgumentTestGoType(t, args[1])}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden/written AST slots=%v, want declaration-owned %v", got, want)
	}
	ctx.localScope = written
	args = sourceClassGoTypeArgumentExprs(member, []string{"T", "T"}, owner, nil, inScopeTypeParameters(ctx), ctx)
	wantWritten := written.TypeParameters[0].EmittedName()
	for i, arg := range args {
		if got := boundedArgumentTestGoType(t, arg); got != wantWritten {
			t.Errorf("explicit slot%d=%s, want method-owned %s", i, got, wantWritten)
		}
	}
	// Check the actual helper formal, not only the standalone argument builder.
	file, ok := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx).(*ast.File)
	if !ok {
		t.Fatal("program AST missing")
	}
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Type.Params == nil {
			continue
		}
		for _, field := range fn.Type.Params.List {
			if len(field.Names) != 1 || field.Names[0].Name != "value" {
				continue
			}
			if boundedArgumentTestGoType(t, field.Type) != "*"+member.Class.Name+"["+want[0]+", "+want[1]+"]" {
				continue
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("helper formal preserving class/method slots %v missing", want)
	}
}

func TestHiddenLexicalReceiverSlotControls(t *testing.T) {
	classParameter := symbol.NewTypeParam("T", nil)
	methodParameter := symbol.NewTypeParam("T", nil)
	localParameter := symbol.NewTypeParam("T", nil)
	symbol.DisambiguateTypeParamGoNames([]symbol.TypeParam{classParameter, methodParameter, localParameter})
	ownParameter := symbol.NewTypeParam("U", nil)
	owner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{classParameter}}
	member := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{classParameter, ownParameter}, DeclaredTypeParameters: []symbol.TypeParam{ownParameter}, IsInner: true, Enclosing: owner}
	method := &symbol.Definition{TypeParameters: []symbol.TypeParam{methodParameter}}
	ctx := Ctx{currentClass: owner, localScope: method}
	localCtx := ctx.Clone()
	localCtx.syntheticTypeParameters = []symbol.TypeParam{localParameter}
	staticCtx := ctx.Clone()
	staticCtx.localScope = &symbol.Definition{TypeParameters: []symbol.TypeParam{methodParameter}, IsStatic: true}
	foreign := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{symbol.NewTypeParam("T", nil)}}
	for _, test := range []struct {
		name                         string
		arguments, receiverArguments []string
		receiver                     *symbol.ClassScope
		ctx                          Ctx
		want                         []string
	}{
		{"method shadow", []string{"T"}, nil, owner, ctx, []string{classParameter.EmittedName(), methodParameter.EmittedName()}},
		{"local shadow", []string{"T"}, nil, owner, localCtx, []string{classParameter.EmittedName(), localParameter.EmittedName()}},
		{"explicit owner", []string{"T", "T"}, nil, owner, ctx, []string{methodParameter.EmittedName(), methodParameter.EmittedName()}},
		{"explicit receiver view", []string{"T"}, []string{"T"}, owner, ctx, []string{methodParameter.EmittedName(), methodParameter.EmittedName()}},
		{"raw own argument", nil, nil, owner, ctx, []string{classParameter.EmittedName(), "any"}},
		{"missing receiver", []string{"T"}, nil, nil, ctx, []string{"any", methodParameter.EmittedName()}},
		{"static fallback", []string{"T"}, nil, owner, staticCtx, []string{methodParameter.EmittedName(), methodParameter.EmittedName()}},
		{"foreign declaration same spelling", []string{"T"}, nil, foreign, ctx, []string{methodParameter.EmittedName(), methodParameter.EmittedName()}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := sourceClassGoTypeArgumentExprs(member, test.arguments, test.receiver, test.receiverArguments, inScopeTypeParameters(test.ctx), test.ctx)
			got := make([]string, len(args))
			for i, arg := range args {
				got[i] = boundedArgumentTestGoType(t, arg)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("slots=%v,want%v", got, test.want)
			}
		})
	}
}

func TestHiddenLexicalReceiverSlotClassShadowAndPartialOwner(t *testing.T) {
	outer := symbol.NewTypeParam("T", nil)
	inner := symbol.NewTypeParam("T", nil)
	methodParameter := symbol.NewTypeParam("T", nil)
	own := symbol.NewTypeParam("U", nil)
	symbol.DisambiguateTypeParamGoNames([]symbol.TypeParam{outer, inner, methodParameter})
	receiver := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{outer, inner}}
	member := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{outer, inner, own}, DeclaredTypeParameters: []symbol.TypeParam{own}}
	ctx := Ctx{currentClass: receiver, localScope: &symbol.Definition{TypeParameters: []symbol.TypeParam{methodParameter}}}
	for _, test := range []struct {
		name          string
		written, want []string
	}{
		{"both hidden", []string{"T"}, []string{outer.EmittedName(), inner.EmittedName(), methodParameter.EmittedName()}},
		{"partly written owner", []string{"T", "T"}, []string{methodParameter.EmittedName(), inner.EmittedName(), methodParameter.EmittedName()}},
		{"fully written owner", []string{"T", "T", "T"}, []string{methodParameter.EmittedName(), methodParameter.EmittedName(), methodParameter.EmittedName()}},
	} {
		t.Run(test.name, func(t *testing.T) {
			arguments := sourceClassGoTypeArgumentExprs(member, test.written, receiver, nil, inScopeTypeParameters(ctx), ctx)
			got := make([]string, len(arguments))
			for i, arg := range arguments {
				got[i] = boundedArgumentTestGoType(t, arg)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("nested class slots=%v,want%v", got, test.want)
			}
		})
	}
}

func TestHiddenLexicalReceiverSlotExplicitOwnerJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner=new Carrier<String>(); Carrier<Child> methodOwner=new Carrier<Child>();
  Carrier<Child>.Cell<Child> methodCell=methodOwner.new Cell<Child>();
  System.out.print(owner.written(methodCell));
 }
}`, "7", map[string]string{"Carrier.java": hiddenLexicalReceiverSource})
}

const hiddenLexicalDependentSource = `interface Marker { int mark(); }
class Carrier<T> {
 class Cell<U extends java.util.List<T>> {}
 <T extends Marker> int wildcard(Cell<?> value){return 11;}
 <T extends Marker> int raw(Cell value){return 12;}
}`

func TestHiddenLexicalReceiverSlotDependentBoundAST(t *testing.T) {
	helper := setupParseHelper(t, hiddenLexicalDependentSource)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	if owner == nil || len(owner.Subclasses) != 1 {
		t.Fatal("dependent owner/member missing")
	}
	member := owner.Subclasses[0]
	if member.TypeParameters[0].Declaration != owner.TypeParameters[0].Declaration {
		t.Fatal("dependent hidden declaration lost")
	}
	bound := member.OwnTypeParameters()[0].Bounds[0]
	if bound.TypeParameterBindings["T"] != owner.TypeParameters[0].Declaration {
		t.Fatal("dependent bound rebound outside declaration")
	}
	for _, name := range []string{"wildcard", "raw"} {
		t.Run(name, func(t *testing.T) {
			method := owner.FindMethodByName(name, nil)
			if method == nil {
				t.Fatal("dependent method missing")
			}
			ctx := helper.Ctx.Clone()
			ctx.currentClass, ctx.localScope = owner, method
			var written []string
			if name == "wildcard" {
				written = []string{"?"}
			}
			args := sourceClassGoTypeArgumentExprs(member, written, owner, nil, inScopeTypeParameters(ctx), ctx)
			className := owner.TypeParameters[0].EmittedName()
			want := []string{className, "*stdjava.List[" + className + "]"}
			got := make([]string, len(args))
			for i, arg := range args {
				got[i] = boundedArgumentTestGoType(t, arg)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("dependent slots=%v,want%v", got, want)
			}
			if member.OwnTypeParameters()[0].Bounds[0].Original != "java.util.List<T>" {
				t.Fatal("dependent declaration bound changed")
			}
		})
	}
}

func TestHiddenLexicalReceiverSlotWrapperNilEquivalence(t *testing.T) {
	helper := setupParseHelper(t, hiddenLexicalDependentSource)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	member := owner.Subclasses[0]
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, owner.FindMethodByName("wildcard", nil)
	for _, written := range [][]string{nil, {"?"}, {"T", "?"}, {"T", "java.util.List<T>"}} {
		arguments := normalizeClassTypeArguments(member, written, owner, nil)
		raw := sourceClassRawArgumentSlots(member, written, owner, nil)
		old := sourceClassGoArguments(member, arguments, raw, inScopeTypeParameters(ctx), ctx)
		methodName := ctx.localScope.TypeParameters[0].EmittedName()
		want := []string{methodName, "*stdjava.List[" + methodName + "]"}
		for i, argument := range old {
			if got := boundedArgumentTestGoType(t, argument); got != want[i] {
				t.Errorf("existing unseeded slot%d=%s, want unchanged %s", i, got, want[i])
			}
		}
		worker := sourceClassGoArgumentsWithLexicalSlots(member, arguments, raw, inScopeTypeParameters(ctx), ctx, nil)
		for i := range old {
			if a, b := boundedArgumentTestGoType(t, old[i]), boundedArgumentTestGoType(t, worker[i]); a != b {
				t.Errorf("nil seed changed slot%d:%s vs%s", i, a, b)
			}
		}
	}
}

func TestHiddenLexicalReceiverSlotDependentExplicitControls(t *testing.T) {
	helper := setupParseHelper(t, hiddenLexicalDependentSource)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	member := owner.Subclasses[0]
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, owner.FindMethodByName("wildcard", nil)
	name := ctx.localScope.TypeParameters[0].EmittedName()
	for _, test := range []struct {
		name                       string
		written, receiverArguments []string
	}{
		{"explicit owner wildcard", []string{"T", "?"}, nil},
		{"explicit receiver raw", nil, []string{"T"}},
		{"explicit receiver wildcard", []string{"?"}, []string{"T"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := sourceClassGoTypeArgumentExprs(member, test.written, owner, test.receiverArguments, inScopeTypeParameters(ctx), ctx)
			want := []string{name, "*stdjava.List[" + name + "]"}
			for i, arg := range args {
				if got := boundedArgumentTestGoType(t, arg); got != want[i] {
					t.Errorf("written dependent slot%d=%s,want%s", i, got, want[i])
				}
			}
		})
	}
}

func TestHiddenLexicalReceiverSlotInterfaceExplicitInvocationJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner=new Carrier<String>(); Carrier<String>.Cell<Child> same=owner.new Cell<Child>();
  System.out.print(owner.<Child>shadow(same));
 }
}`, "5", map[string]string{"Carrier.java": hiddenLexicalReceiverSource})
}

func TestHiddenLexicalReceiverSlotDependentExplicitInvocationJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner=new Carrier<String>();
  System.out.print(owner.<Marker>wildcard((Carrier<String>.Cell<?>)null)+":"+owner.<Marker>raw((Carrier<String>.Cell<?>)null));
 }
}`, "11:12", map[string]string{"Carrier.java": hiddenLexicalDependentSource})
}
