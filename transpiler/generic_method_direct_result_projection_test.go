package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

const directBoundResultExpandedSource = `class Base { int baseCode(){return 9;} }
class Child extends Base {int childCode(){return 17;}}
class Owner<T> {
 class Cell<U>{U value;Cell(U value){this.value=value;}U read(){Main.reads++;return value;}}
 <T extends Base>T echo(Cell<T> value){T result=value.read();Main.bodies++;return result;}
}
public class Main {
 static int reads,bodies;
 public static void main(String[]args){
  Owner<String> owner=new Owner<String>();Child child=new Child();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(child);
  Child declared=owner.echo(cell);System.out.println("declared="+(declared==child));
  Base base=owner.echo(cell);System.out.println("base="+base.baseCode());
  Object object=owner.echo(cell);System.out.println("object="+(object==child));
  var inferred=owner.echo(cell);System.out.println("var="+inferred.childCode());
  System.out.println("chain="+owner.echo(cell).childCode());
  owner.echo(cell);System.out.println("discard="+reads+":"+bodies);
  Owner<String>.Cell<Child> empty=owner.new Cell<Child>(null);
  Child nullable=owner.echo(empty);System.out.println("null="+(nullable==null)+":"+reads+":"+bodies);
  Owner<String>.Cell<Base> wide=owner.new Cell<Base>(child);
  Base wider=owner.<Base>echo(wide);System.out.println("explicit="+wider.baseCode()+":"+reads+":"+bodies);
 }
}
`
const directBoundResultExpandedOracle = "declared=true\nbase=9\nobject=true\nvar=17\nchain=17\ndiscard=6:6\nnull=true:7:7\nexplicit=9:8:8\n"

func TestGenericDirectBoundResultPreservesJavaInference(t *testing.T) {
	helper := setupParseHelper(t, directBoundResultExpandedSource)
	owner := helper.File.Symbols.FindClassScope("Owner")
	main := helper.File.Symbols.FindClassScope("Main")
	def := owner.FindMethodByName("echo", nil)
	if def.TypeParameters[0].Declaration == owner.TypeParameters[0].Declaration || def.TypeParameters[0].EmittedName() == owner.TypeParameters[0].EmittedName() {
		t.Fatal("method/class T identities collapsed")
	}
	ctx := helper.Ctx.Clone()
	ctx.currentClass = main
	ctx.localScope = main.FindMethodByName("main", nil)
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(helper.File.Source) == "echo" {
			seen++
			want := "Child"
			if n.ChildByFieldName("type_arguments") != nil {
				want = "Base"
			}
			got, ok := inferredGenericInvocationResult(def, n, ctx, helper.File.Source)
			if !ok || stripJavaQualifier(got) != want {
				t.Errorf("%s Java result=%q known=%v,want %s", n.Content(helper.File.Source), got, ok, want)
			}
			if n.ChildByFieldName("type_arguments") == nil {
				physical := genericArrayInvocationTypeBindings(def, n, ctx, helper.File.Source)
				if stripJavaQualifier(physical["T"]) != "Base" {
					t.Errorf("physical helper binding changed: %v", physical)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 8 {
		t.Fatalf("calls=%d,want8", seen)
	}
}

func TestGenericDirectBoundResultUsesDeclaredConsumer(t *testing.T) {
	helper := setupParseHelper(t, directBoundResultExpandedSource)
	main := helper.File.Symbols.FindClassScope("Main")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = main
	ctx.localScope = main.FindMethodByName("main", nil)
	targets := map[string]string{"declared": "Child", "base": "Base", "object": "Object", "nullable": "Child", "wider": "Base"}
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "variable_declarator" {
			name := n.ChildByFieldName("name").Content(helper.File.Source)
			if target, ok := targets[name]; ok {
				seen++
				invocation := n.ChildByFieldName("value")
				consuming := ctx.Clone()
				consuming.expectedType = target
				consuming.expectedTypeRoot = invocation
				expr := ParseExpr(invocation, helper.File.Source, consuming)
				var out bytes.Buffer
				if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), "ObjectView[") {
					t.Errorf("%s has no post-call consumer projection: %s", name, out.String())
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != len(targets) {
		t.Fatalf("consumers=%d,want%d", seen, len(targets))
	}
}

func TestGenericDirectBoundResultExpandedJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultExpandedSource, directBoundResultExpandedOracle)
}

// Byte-identical copy of the original full physical-owner matrix. Its cast
// timing is mandatory: Base/Object consumers succeed, Child fails after the
// body, and an Alien fails the first-bound read before the body counter.
func TestGenericDirectBoundResultOriginalTimingJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `class Base{} class Child extends Base{} class Alien{}
class Owner<T> {
 class Cell<U>{U value;Cell(U seed){value=seed;}U read(){Main.reads++;return value;}void put(U next){value=next;}}
 <T extends Base>T echo(Cell<T> value){T got=value.read();Main.bodies++;return got;}
}
public class Main {
 static int reads,bodies;
 public static void main(String[]args) {
  Owner<String> owner=new Owner<String>();Child child=new Child();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(child);
  Child returned=owner.echo(cell);System.out.println("same="+(returned==child));
  Owner.Cell raw=cell;raw.put(new Base());
  Base broad=owner.echo(cell);System.out.println("broad="+(broad!=null)+":"+reads+":"+bodies);
  try{Child narrow=owner.echo(cell);System.out.println("wrong="+narrow);}catch(ClassCastException expected){System.out.println("resultcast="+reads+":"+bodies);}
  raw.put(new Alien());try{Base bad=owner.echo(cell);System.out.println("wrong="+bad);}catch(ClassCastException expected){System.out.println("readcast="+reads+":"+bodies);}
 }
}`, "same=true\nbroad=true:2:2\nresultcast=3:3\nreadcast=4:3\n")
}

func TestGenericDirectBoundResultVarAndSuperclassProjection(t *testing.T) {
	helper := setupParseHelper(t, directBoundResultExpandedSource)
	main := helper.File.Symbols.FindClassScope("Main")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = main
	ctx.localScope = main.FindMethodByName("main", nil)
	targets := map[string]string{"inferred": "var", "base": "Base"}
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "variable_declarator" {
			if target, ok := targets[n.ChildByFieldName("name").Content(helper.File.Source)]; ok {
				seen++
				invocation := n.ChildByFieldName("value")
				consuming := ctx.Clone()
				consuming.expectedType = target
				consuming.expectedTypeRoot = invocation
				expr := ParseExpr(invocation, helper.File.Source, consuming)
				if target != "var" {
					expr = coerceArgumentToExpectedType(expr, invocation, target, consuming, helper.File.Source)
				}
				ast.Inspect(expr, func(node ast.Node) bool {
					if id, ok := node.(*ast.Ident); ok && id.Name == "var" {
						t.Error("Java var became an emitted Go type instead of the argument-inferred Child")
					}
					if selector, ok := node.(*ast.SelectorExpr); ok {
						if call, ok := selector.X.(*ast.CallExpr); ok {
							if index, ok := call.Fun.(*ast.IndexExpr); ok {
								if function, ok := index.X.(*ast.SelectorExpr); ok && function.Sel.Name == "ObjectView" {
									t.Errorf("already projected consumer receives a second superclass selector %s", selector.Sel.Name)
								}
							}
						}
					}
					return true
				})
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != len(targets) {
		t.Fatalf("controls=%d want%d", seen, len(targets))
	}
}

// A Java class and method can resemble the runtime service. Its ordinary
// Child result still needs the Base superclass view at this assignment.
const directBoundResultSourceLookalike = `class Base{} class Child extends Base{}
class stdjava {static Child ObjectView(Child value){return value;}}
public class Main{public static void main(String[]args){Child child=new Child();Base result=stdjava.ObjectView(child);System.out.println(result==child);}}
`

func TestGenericDirectBoundResultSourceLookalikeJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultSourceLookalike, "true\n")
}

func TestGenericDirectBoundResultRuntimeProjectionIdentity(t *testing.T) {
	helper := setupParseHelper(t, directBoundResultSourceLookalike)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("main", nil)
	// Resolving the source class reserves its name before choosing the runtime
	// import alias, exactly as the ordinary source caller does.
	ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	typ := javaTypeStringToGoTypeExpr("Base", inScopeTypeParameters(ctx), ctx)
	projected := stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{typ}, []ast.Expr{ast.NewIdent("value"), ast.NewIdent("descriptor")})
	if !runtimeObjectViewHasExpectedType(projected, "Base", ctx) {
		t.Fatal("matching generated runtime view was not recognized")
	}
	if runtimeObjectViewHasExpectedType(projected, "Child", ctx) {
		t.Fatal("different Java consumer was treated as already projected")
	}
	lookalike := &ast.CallExpr{Fun: &ast.IndexExpr{X: &ast.SelectorExpr{X: ast.NewIdent("source"), Sel: ast.NewIdent("ObjectView")}, Index: typ}, Args: []ast.Expr{ast.NewIdent("value"), ast.NewIdent("descriptor")}}
	if runtimeObjectViewHasExpectedType(lookalike, "Base", ctx) {
		t.Fatal("source ObjectView lookalike suppressed a required conversion")
	}
	local := &ast.CallExpr{Fun: &ast.IndexExpr{X: ast.NewIdent("ObjectView"), Index: typ}, Args: []ast.Expr{ast.NewIdent("value")}}
	if runtimeObjectViewHasExpectedType(local, "Base", ctx) {
		t.Fatal("lexical ObjectView lookalike suppressed a required conversion")
	}
}

func TestGenericDirectBoundResultChainedJavaOrigin(t *testing.T) {
	helper := setupParseHelper(t, directBoundResultExpandedSource)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("main", nil)
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(helper.File.Source) == "childCode" {
			object := n.ChildByFieldName("object")
			if object != nil && object.Type() == "method_invocation" {
				seen++
				var origin inferredJavaTypeOrigin
				typ, known := inferExprJavaType(object, ctx, helper.File.Source, &origin)
				child := helper.File.Symbols.FindClassScope("Child")
				if !known || stripJavaQualifier(typ) != "Child" || origin.nominalScope != child || origin.parameter != nil {
					t.Errorf("chained generic result type=%s known=%v origin=%+v want nominal Child", typ, known, origin)
				}
				result, resultKnown := inferExprJavaType(n, ctx, helper.File.Source)
				if !resultKnown || result != "int" {
					t.Errorf("actual chained childCode returns %s known=%v want int", result, resultKnown)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 1 {
		t.Fatalf("chains=%d want1", seen)
	}
}

// The actual nominal class T and the class/method binders T are three distinct
// declarations. The chained caller must use the nominal class's primitive
// method result; the method's Base helper view remains a separate ABI choice.
const directBoundResultNominalBinderSource = `class Base{int baseCode(){return 9;}}class T extends Base{int childCode(){return 17;}}
class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}U read(){return value;}}<T extends Base>T echo(Cell<T> value){return value.read();}}
public class Main{public static void main(String[]args){Owner<String> owner=new Owner<String>();T child=new T();Owner<String>.Cell<T> cell=owner.new Cell<T>(child);System.out.println("chain="+owner.echo(cell).childCode());Base broad=owner.echo(cell);System.out.println("base="+broad.baseCode());}}
`

func TestGenericDirectBoundResultNominalBinderJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultNominalBinderSource, "chain=17\nbase=9\n")
}

func TestGenericDirectBoundResultSeparatesOwnerAndSiblingOrigins(t *testing.T) {
	helper := setupParseHelper(t, `class Base{}class Child extends Base{int childCode(){return 17;}}class Sibling extends Base{int siblingCode(){return 23;}}
class Owner<T extends Base>{T own;Owner(T own){this.own=own;}<T extends Base>T echo(T value){return value;}<U extends Base>T classResult(U value){return own;}}
class Main{static void check(){Owner<Sibling> owner=new Owner<Sibling>(new Sibling());Child child=new Child();owner.echo(child).childCode();owner.classResult(child).siblingCode();}}`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("check", nil)
	targets := map[string]string{"echo": "Child", "classResult": "Sibling"}
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			if want, ok := targets[n.ChildByFieldName("name").Content(helper.File.Source)]; ok {
				seen++
				var origin inferredJavaTypeOrigin
				typ, known := inferExprJavaType(n, ctx, helper.File.Source, &origin)
				if !known || stripJavaQualifier(typ) != want || origin.nominalScope != helper.File.Symbols.FindClassScope(want) {
					t.Errorf("%s result=%s known=%v origin=%+v want %s", n.Content(helper.File.Source), typ, known, origin, want)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != len(targets) {
		t.Fatalf("calls=%d want%d", seen, len(targets))
	}
}

const directBoundResultAliasNominalSource = `import aliasguard.Base;import aliasguard.T2;import aliasguard.T3;
class Child extends Base{int childCode(){return 17;}}
class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}U read(){return value;}}<T extends Base>T pair(Cell<T> value,Cell<T2> second,Cell<T3> third){return value.read();}}
public class Main{public static void main(String[]args){Owner<String> owner=new Owner<String>();Owner<String>.Cell<Child> child=owner.new Cell<Child>(new Child());Owner<String>.Cell<T2> second=owner.new Cell<T2>(new T2());Owner<String>.Cell<T3> third=owner.new Cell<T3>(new T3());System.out.println("aliases="+owner.pair(child,second,third).childCode());}}
`

func TestGenericDirectBoundResultAliasNominalJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                            `<project><groupId>probe</groupId><artifactId>generic-alias-origin</artifactId><version>1</version></project>`,
		"src/main/java/Main.java":            directBoundResultAliasNominalSource,
		"src/main/java/aliasguard/Base.java": `package aliasguard;public class Base{}`,
		"src/main/java/aliasguard/T2.java":   `package aliasguard;public class T2 extends Base{}`,
		"src/main/java/aliasguard/T3.java":   `package aliasguard;public class T3 extends Base{}`,
	}, "Main", "aliases=17\n")
}

func TestGenericDirectBoundResultAliasNominalOrigins(t *testing.T) {
	helper := setupParseHelper(t, `package aliasguard;
class Base{}class Child extends Base{int childCode(){return 17;}}class T2 extends Base{}class T3 extends Base{}
class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}U read(){return value;}}<T extends Base>T pair(Cell<T> value,Cell<T2> second,Cell<T3> third){return value.read();}}
class Main{static void check(){Owner<String> owner=new Owner<String>();Owner<String>.Cell<Child> child=owner.new Cell<Child>(new Child());Owner<String>.Cell<T2> second=owner.new Cell<T2>(new T2());Owner<String>.Cell<T3> third=owner.new Cell<T3>(new T3());owner.pair(child,second,third).childCode();}}`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("check", nil)
	owner := helper.File.Symbols.FindClassScope("Owner")
	def := owner.FindMethodByName("pair", nil)
	if def.TypeParameters[0].Declaration == owner.TypeParameters[0].Declaration {
		t.Fatal("class/method T collapsed")
	}
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(helper.File.Source) == "pair" {
			seen++
			var origin inferredJavaTypeOrigin
			typ, known := inferExprJavaType(n, ctx, helper.File.Source, &origin)
			if !known || typ != "aliasguard.Child" || origin.nominalScope != helper.File.Symbols.FindClassScope("Child") || origin.parameter != nil {
				t.Errorf("nominal T2/T3 sibling arguments changed pair result=%s known=%v origin=%+v", typ, known, origin)
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 1 {
		t.Fatalf("calls=%d want1", seen)
	}
}

func TestGenericDirectBoundResultQualifiedFormalCollectorCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, formal string
		declared     bool
		want         bool
	}{
		{"declared alias", "T2", true, true},
		{"declared nominal", "aliasguard.T2", true, false},
		{"declared array nominal", "aliasguard.T2[]", true, false},
		{"legacy ownerless alias", "aliasguard.T2", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bounds := map[string][]string{}
			collect := collectGenericMethodInferenceBounds
			if tc.declared {
				collect = collectDeclaredGenericMethodInferenceBounds
			}
			actual := "Child"
			if strings.HasSuffix(tc.formal, "[]") {
				actual += "[]"
			}
			collect(tc.formal, actual, map[string]struct{}{"T2": {}}, bounds, Ctx{})
			if (len(bounds["T2"]) > 0) != tc.want {
				t.Fatalf("qualified formal %s bounds=%v want binding=%v", tc.formal, bounds, tc.want)
			}
		})
	}
}

const directBoundResultNominalReturnSource = `import aliasreturn.Base;import aliasreturn.T2;import aliasreturn.T3;
class Child extends Base{}
class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}}<T extends Base>T2 named(Cell<T> value){return new T2();}<T extends Base>T3 third(Cell<T> value){return new T3();}}
public class Main{public static void main(String[]args){Owner<String> owner=new Owner<String>();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(new Child());T2 second=owner.named(cell);T3 third=owner.third(cell);System.out.println("returns="+second.code()+":"+third.code());}}
`

func TestGenericDirectBoundResultNominalReturnJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                             `<project><groupId>probe</groupId><artifactId>generic-nominal-return</artifactId><version>1</version></project>`,
		"src/main/java/Main.java":             directBoundResultNominalReturnSource,
		"src/main/java/aliasreturn/Base.java": `package aliasreturn;public class Base{}`,
		"src/main/java/aliasreturn/T2.java":   `package aliasreturn;public class T2 extends Base{public int code(){return 22;}}`,
		"src/main/java/aliasreturn/T3.java":   `package aliasreturn;public class T3 extends Base{public int code(){return 33;}}`,
	}, "Main", "returns=22:33\n")
}
func TestGenericDirectBoundResultNominalReturnProjection(t *testing.T) {
	helper := setupParseHelper(t, `package aliasreturn;class Base{}class Child extends Base{}class T2 extends Base{}class T3 extends Base{}
class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}}<T extends Base>T2 named(Cell<T> value){return new T2();}<T extends Base>T3 third(Cell<T> value){return new T3();}}
class Main{static void check(){Owner<String> owner=new Owner<String>();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(new Child());owner.named(cell);owner.third(cell);}}`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("check", nil)
	owner := helper.File.Symbols.FindClassScope("Owner")
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name").Content(helper.File.Source)
			if name == "named" || name == "third" {
				seen++
				def := owner.FindMethodByName(name, nil)
				typ, known := inferredGenericInvocationResult(def, n, ctx, helper.File.Source)
				want := "aliasreturn.T2"
				if name == "third" {
					want = "aliasreturn.T3"
				}
				if !known || typ != want {
					t.Errorf("nominal result=%s known=%v want%s", typ, known, want)
				}
				call := &ast.CallExpr{Fun: ast.NewIdent("invoke")}
				result := genericMethodProjectedResult(call, def, []ast.Expr{javaTypeStringToGoTypeExpr("Base", nil, ctx)}, genericJavaInvocationTypeBindings(def, n, ctx, helper.File.Source), "", ctx)
				if result != call {
					t.Errorf("nominal %s return was projected through an unrelated method binder", want)
				}
				copied := *def
				if genericMethodProjectedResult(call, &copied, []ast.Expr{ast.NewIdent("any")}, map[string]string{"T": "Child"}, "", ctx) != call {
					t.Errorf("copied source nominal %s return used an ownerless synthetic fallback", want)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 2 {
		t.Fatalf("calls=%d want2", seen)
	}
}

const directBoundResultNoArgVarSource = `class Base{int code(){return 9;}}class Owner<T>{<T extends Base>T make(){return null;}}
public class Main{public static void main(String[]args){Owner<String> owner=new Owner<String>();var value=owner.make();System.out.println("varnull="+(value==null));}}
`

func TestGenericDirectBoundResultNoArgVarJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultNoArgVarSource, "varnull=true\n")
}
func TestGenericDirectBoundResultNoArgVarInference(t *testing.T) {
	helper := setupParseHelper(t, directBoundResultNoArgVarSource)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("main", nil)
	def := helper.File.Symbols.FindClassScope("Owner").FindMethodByName("make", nil)
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(helper.File.Source) == "make" {
			seen++
			target := ctx.Clone()
			target.expectedType = "var"
			target.expectedTypeRoot = n
			typ, known := inferredGenericInvocationResult(def, n, target, helper.File.Source)
			untargeted, untargetedKnown := inferredGenericInvocationResult(def, n, ctx, helper.File.Source)
			if typ == "var" || typ != untargeted || known != untargetedKnown {
				t.Errorf("var invented a target: result=%s known=%v; untargeted=%s known=%v", typ, known, untargeted, untargetedKnown)
			}
			expr := ParseExpr(n, helper.File.Source, target)
			if !runtimeObjectViewHasExpectedType(expr, "Base", target) {
				t.Error("actual noarg var emission lost the declared Base view")
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 1 {
		t.Fatalf("calls=%d want1", seen)
	}
}

func TestGenericDirectBoundResultOwnerlessProjectionCompatibility(t *testing.T) {
	for _, spelling := range []string{"U", "U2"} {
		def := &symbol.Definition{OriginalType: spelling, TypeParameters: []symbol.TypeParam{{Name: "U", Declaration: &symbol.TypeParamDeclaration{SourceName: "U", GoName: "U2"}}}}
		call := &ast.CallExpr{Fun: ast.NewIdent("invoke")}
		projected := genericMethodProjectedResult(call, def, []ast.Expr{ast.NewIdent("any")}, map[string]string{"U": "java.lang.Object"}, "", Ctx{})
		if projected == call {
			t.Errorf("ownerless synthetic %s projection lost historical alias behavior", spelling)
		}
	}
}

const directBoundResultUnboundedClassOwnedSource = `class Base{int code(){return 9;}}class Child extends Base{}class Sibling extends Base{}
class Owner<T>{T own;Owner(T own){this.own=own;}<U extends Base>T classResult(U other){return own;}}
public class Main{public static void main(String[]args){Child child=new Child();Owner<Child> owner=new Owner<Child>(child);Child result=owner.classResult(new Sibling());System.out.println("class="+(result==child)+":"+result.code());}}
`

func TestGenericDirectBoundResultUnboundedClassOwnedJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultUnboundedClassOwnedSource, "class=true:9\n")
}

const directBoundResultNoArgChainSource = `class Base{int code(){return 9;}}class Owner<T>{@SuppressWarnings("unchecked")<T extends Base>T make(){Main.bodies++;return (T)(Main.empty?null:new Base());}}
public class Main{static boolean empty=true;static int bodies,finals;public static void main(String[]args){Owner<String> owner=new Owner<String>();try{owner.make().code();System.out.println("wrong");}catch(NullPointerException expected){System.out.print("null="+bodies+":");}finally{finals++;System.out.println(finals);}empty=false;System.out.println("base="+owner.make().code()+":"+bodies);}}
`

func TestGenericDirectBoundResultNoArgBoundChainJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultNoArgChainSource, "null=1:1\nbase=9:2\n")
}

const directBoundResultNoArgExplicitSource = `class Base{}class Child extends Base{}class Owner<T>{@SuppressWarnings("unchecked")<T extends Base>T make(){Main.bodies++;return (T)(Main.empty?null:new Base());}}
public class Main{static boolean empty=true;static int bodies;public static void main(String[]args){Owner<String> owner=new Owner<String>();Child missing=owner.<Child>make();System.out.println("null="+(missing==null)+":"+bodies);empty=false;try{Child value=owner.<Child>make();System.out.println("wrong="+value);}catch(ClassCastException expected){System.out.println("child="+bodies);}}}
`

func TestGenericDirectBoundResultNoArgExplicitChildJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, directBoundResultNoArgExplicitSource, "null=true:1\nchild=2\n")
}
