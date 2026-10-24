package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func originalInheritedGenericFieldSource(t *testing.T) string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "local_generic_edge_audit_tdd_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "TestLocalGenericEdgeAudit_InheritedGenericFieldSubstitutionPaths" {
			continue
		}
		for _, statement := range function.Body.List {
			expression, ok := statement.(*ast.ExprStmt)
			if !ok {
				continue
			}
			call, ok := expression.X.(*ast.CallExpr)
			if !ok || len(call.Args) != 3 {
				continue
			}
			literal, ok := call.Args[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			source, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			return source
		}
	}
	t.Fatal("unchanged original inherited-field Java fixture missing")
	return ""
}

func TestInnerCarrierOriginalNumericInvocationTypes(t *testing.T) {
	source := originalInheritedGenericFieldSource(t)
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("InheritedGenericFieldProgram")
	if ctx.currentClass == nil {
		t.Fatal("original program class missing")
	}
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "run" {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil {
		t.Fatal("original run method missing")
	}
	observed := map[string]int{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" {
			name := node.ChildByFieldName("name").Content(helper.File.Source)
			if name == "read" || name == "readWildcard" || name == "readRaw" {
				observed[name]++
				actual, known := inferExprJavaType(node, ctx, helper.File.Source)
				if !known || actual != "int" {
					t.Errorf("original %s infers %q/%t, want int/true", node.Content(helper.File.Source), actual, known)
				}
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	for _, name := range []string{"read", "readWildcard", "readRaw"} {
		if observed[name] != 1 {
			t.Fatalf("original numeric invocation %s count=%d, want1", name, observed[name])
		}
	}
}

func TestInnerCarrierOriginalJDK21(t *testing.T) {
	source := originalInheritedGenericFieldSource(t)
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main { public static void main(String[] args){System.out.print(InheritedGenericFieldProgram.run());} }`, "6:7:8", map[string]string{"InheritedGenericFieldProgram.java": source})
}

const innerCarrierIdentitySource = `class T {}
class Base {} class Child extends Base {}
class Carrier<T> {
 class Cell<U> {}
 int implicit(Cell<Integer> value){return 1;} long implicit(Object value){return 2L;}
 int explicit(Carrier<T>.Cell<Integer> value){return 3;} long explicit(Object value){return 4L;}
 <T extends Base> int shadow(Cell<T> value){return 5;} long shadow(Object value){return 6L;}
}
class Probe {
 static String run(Carrier<String> owner,Carrier<String>.Cell<Integer> same,Carrier<Integer>.Cell<Integer> other,Carrier<String>.Cell<Child> sameBound,Carrier<Integer>.Cell<Child> otherBound){
  return owner.implicit(same)+":"+owner.implicit(other)+":"+owner.explicit(same)+":"+owner.explicit(other)+":"+owner.shadow(sameBound)+":"+owner.shadow(otherBound);
 }
}`

func TestInnerCarrierApplicabilityKeepsReceiverAndBinderIdentity(t *testing.T) {
	source := innerCarrierIdentitySource
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Probe")
	if ctx.currentClass == nil {
		t.Fatal("probe class missing")
	}
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "run" {
			ctx.localScope = method
		}
	}
	want := map[string]string{
		"owner.implicit(same)": "int", "owner.implicit(other)": "long",
		"owner.explicit(same)": "int", "owner.explicit(other)": "long",
		"owner.shadow(sameBound)": "int", "owner.shadow(otherBound)": "long",
	}
	observed := map[string]bool{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" {
			text := node.Content(helper.File.Source)
			if expected, found := want[text]; found {
				observed[text] = true
				actual, known := inferExprJavaType(node, ctx, helper.File.Source)
				if !known || actual != expected {
					t.Errorf("%s infers %q/%t, want %s/true", text, actual, known, expected)
				}
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if len(observed) != len(want) {
		t.Fatalf("observed %d invocation controls, want%d", len(observed), len(want))
	}
}

func TestInnerCarrierCallerBinderIsNotCalleeInferenceVariable(t *testing.T) {
	helper := setupParseHelper(t, `class Cargo<T> { class Slot<U> {} int pick(Slot<Integer> value){return 1;} long pick(Object value){return 2L;} }
class Use { static <T> long run(Cargo<String> owner,Cargo<T>.Slot<Integer> other){return owner.pick(other);} }`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Use")
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "run" {
			ctx.localScope = method
		}
	}
	var walk func(*sitter.Node)
	observed := false
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" && node.ChildByFieldName("name").Content(helper.File.Source) == "pick" {
			observed = true
			actual, known := inferExprJavaType(node, ctx, helper.File.Source)
			if !known || actual != "long" {
				t.Errorf("unrelated caller T infers %q/%t, want long/true", actual, known)
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if !observed {
		t.Fatal("caller binder invocation missing")
	}
}

// This independent four-call gate covers receiver-owned invariant formals.
// The six-call shadowed-helper reproducer is retained as a pending campaign gate.
func TestInnerCarrierReceiverIdentityJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner = new Carrier<String>();
  Carrier<Integer> otherOwner = new Carrier<Integer>();
  Carrier<String>.Cell<Integer> same = owner.new Cell<Integer>();
  Carrier<Integer>.Cell<Integer> other = otherOwner.new Cell<Integer>();
  System.out.print(owner.implicit(same)+":"+owner.implicit(other)+":"+owner.explicit(same)+":"+owner.explicit(other));
 }
}`, "1:2:3:4", map[string]string{"Carrier.java": `class Carrier<T> {
 class Cell<U> {}
 int implicit(Cell<Integer> value){return 1;} long implicit(Object value){return 2L;}
 int explicit(Carrier<T>.Cell<Integer> value){return 3;} long explicit(Object value){return 4L;}
}`})
}

func TestInnerCarrierRawFormalKeepsUncheckedApplicability(t *testing.T) {
	const source = `class Storage<T> {
 class Part<U> {}
 int choose(Part value){return 1;} long choose(Object value){return 2L;}
}
public class Main {
 public static void main(String[]args){Storage<String> owner=new Storage<String>();Storage<Integer> otherOwner=new Storage<Integer>();Storage<String>.Part<Integer> same=owner.new Part<Integer>();Storage<Integer>.Part<Integer> other=otherOwner.new Part<Integer>();System.out.print(owner.choose(same)+":"+owner.choose(other));}
}`
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "main" {
			ctx.localScope = method
		}
	}
	observed := map[string]bool{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" && node.ChildByFieldName("name").Content(helper.File.Source) == "choose" {
			text := node.Content(helper.File.Source)
			observed[text] = true
			actual, known := inferExprJavaType(node, ctx, helper.File.Source)
			if !known || actual != "int" {
				t.Errorf("raw formal %s infers %q/%t, want int/true", text, actual, known)
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	for _, call := range []string{"owner.choose(same)", "owner.choose(other)"} {
		if !observed[call] {
			t.Errorf("raw formal call missing: %s", call)
		}
	}
}
