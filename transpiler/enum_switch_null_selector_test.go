package transpiler

import (
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

const enumNull184Java = `package enumnull;
public class Probe {
 public static void main(String[] args) {
  String result;
  switch(args[0]) {
  case "colon": result=NullColon.run();break;
  case "arrow": result=NullArrow.run();break;
  case "expression": result=NullExpression.run();break;
  case "explicit-colon": result=ExplicitColon.run(0);break;
  case "explicit-arrow": result=ExplicitArrow.run(0);break;
  case "combined-null-default": result=CombinedNullDefault.run(false);break;
  case "nested": result=NestedNull.run();break;
  case "abrupt": result=Abrupt.run();break;
  case "nonnull": result=NonNull.run();break;
  case "repeated": result=Repeated.run();break;
  case "default-only": result=DefaultOnly.run();break;
  case "empty": result=EmptySwitch.run();break;
  case "explicit-colon-nonnull": result=ExplicitColon.run(1);break;
  case "explicit-colon-default": result=ExplicitColon.run(2);break;
  case "explicit-arrow-nonnull": result=ExplicitArrow.run(1);break;
  case "explicit-arrow-default": result=ExplicitArrow.run(2);break;
  case "combined-unmatched": result=CombinedNullDefault.run(true);break;
  case "combined-colon-null": result=CombinedNullDefaultColon.run(false);break;
  case "combined-colon-unmatched": result=CombinedNullDefaultColon.run(true);break;
  default: throw new AssertionError(args[0]);
  }
  System.out.println(result);
 }
}
class Trace {static String value="";static void add(String text){value+=text;}}
class NullColon {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return null;}
 static String run(){Trace.value="";int out=-9;try{switch(select()){case A:out=1;break;case B:out=2;break;default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class NullArrow {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return null;}
 static String run(){Trace.value="";int out=-9;try{switch(select()){case A -> out=1;case B -> out=2;default -> out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class NullExpression {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return null;}
 static String run(){Trace.value="";int out=-9;try{out=switch(select()){case A -> 1;case B -> 2;};}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class ExplicitColon {
 enum Band {A,B,C;static {Trace.add("I");}}
 static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:Band.C;}
 static String run(int mode){Trace.value="";int out=-9;switch(select(mode)){case null:out=4;break;case A:out=1;break;case B:out=2;break;default:out=3;}return Trace.value+":"+out;}
}
class ExplicitArrow {
 enum Band {A,B,C;static {Trace.add("I");}}
 static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:Band.C;}
 static String run(int mode){Trace.value="";int out=-9;switch(select(mode)){case null -> out=4;case A,B -> out=1;default -> out=3;}return Trace.value+":"+out;}
}
class CombinedNullDefault {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(boolean nonnull){Trace.add("S");return nonnull?Band.B:null;}
 static String run(boolean nonnull){Trace.value="";int out=-9;switch(select(nonnull)){case A -> out=1;case null,default -> out=4;}return Trace.value+":"+out;}
}
class NestedNull {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return null;}
 static String run(){Trace.value="";int out=-9;try{switch(select()){case A: switch(select()){case null:out=4;break;default:out=5;}break;case B:out=2;break;default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class Abrupt {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");throw new IllegalStateException("stop");}
 static String run(){Trace.value="";int out=-9;try{switch(select()){case A:out=1;break;case B:out=2;break;default:out=3;}}catch(IllegalStateException e){Trace.add("X");}return Trace.value+":"+out;}
}
class NonNull {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return Band.A;}
 static String run(){Trace.value="";int out=-9;switch(select()){case A:out=1;break;case B:out=2;break;default:out=3;}return Trace.value+":"+out;}
}
class Repeated {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(boolean value){Trace.add("S");return value?Band.A:null;}
 static int choose(boolean value){switch(select(value)){case A:return 1;case B:return 2;default:return 3;}}
 static String run(){Trace.value="";int out=choose(true);try{out+=choose(false);}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class DefaultOnly {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return null;}
 static String run(){Trace.value="";int out=-9;try{switch(select()){default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class EmptySwitch {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(){Trace.add("S");return null;}
 static String run(){Trace.value="";int out=-9;try{switch(select()){}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}
}
class CombinedNullDefaultColon {
 enum Band {A,B;static {Trace.add("I");}}
 static Band select(boolean nonnull){Trace.add("S");return nonnull?Band.B:null;}
 static String run(boolean nonnull){Trace.value="";int out=-9;switch(select(nonnull)){case A:out=1;break;case null,default:out=4;break;}return Trace.value+":"+out;}
}
`

func enumNull184Context(t *testing.T, className, method string) (Ctx, []*sitter.Node) {
	t.Helper()
	const caller = "enumnull/Probe.java"
	ctx := ordinaryDemandTestContext(t, map[string]string{caller: enumNull184Java}, caller)
	scope := ctx.currentFile.FindClassScope(className)
	if scope == nil {
		t.Fatal("fixture source class missing")
	}
	ctx = classScopeCtx(scope, ctx)
	methods := scope.FindMethod().ByOriginalName(method)
	if len(methods) != 1 {
		t.Fatal("fixture unique source method missing")
	}
	ctx.localScope = methods[0]
	ctx.importAliases = map[string]string{}
	ctx.usedImports = map[string]bool{}
	var selections []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "switch_expression" || node.Type() == "switch_statement" {
			selections = append(selections, node)
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(methods[0].DeclarationNode)
	if len(selections) == 0 {
		t.Fatal("fixture switch missing")
	}
	return ctx, selections
}

func enumNull184CheckTag(t *testing.T, ctx Ctx, selection *sitter.Node, guarded bool) {
	t.Helper()
	selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
	if selector == nil || block == nil {
		t.Fatal("fixture selector/block missing")
	}
	tag := ast.NewIdent("alreadyLoweredTag")
	before := len(Diagnostics())
	got := canonicalSourceEnumSwitchSelector(tag, selector, block, ctx.currentFile.Source, ctx)
	if len(Diagnostics()) != before {
		t.Fatal("tag proof introduced diagnostics")
	}
	if !guarded {
		if got != tag {
			t.Fatalf("old tag identity changed: %s", enumSwitch184ExpressionText(t, got))
		}
		return
	}
	call, ok := got.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Args[0] != tag {
		t.Fatalf("tag not preserved once: %s", enumSwitch184ExpressionText(t, got))
	}
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fun.Sel.Name != "ReferenceRequireNonNull" {
		t.Fatalf("wrong null contract: %s", enumSwitch184ExpressionText(t, got))
	}
	count := 0
	ast.Inspect(got, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name == tag.Name {
			count++
		}
		return true
	})
	if count != 1 {
		t.Fatalf("selector occurs %d times", count)
	}
}

func TestEnumNull184DirectOwnedLabelsAndDefaultOnly(t *testing.T) {
	rows := []struct {
		class, method string
		guards        []bool
	}{
		{"NullColon", "run", []bool{true}}, {"NullArrow", "run", []bool{true}}, {"NullExpression", "run", []bool{true}},
		{"ExplicitColon", "run", []bool{false}}, {"ExplicitArrow", "run", []bool{false}}, {"CombinedNullDefault", "run", []bool{false}},
		{"CombinedNullDefaultColon", "run", []bool{false}},
		{"NestedNull", "run", []bool{true, false}}, {"Abrupt", "run", []bool{true}}, {"NonNull", "run", []bool{true}}, {"Repeated", "choose", []bool{true}},
		{"DefaultOnly", "run", []bool{true}}, {"EmptySwitch", "run", []bool{true}},
	}
	for _, row := range rows {
		t.Run(row.class, func(t *testing.T) {
			ctx, selections := enumNull184Context(t, row.class, row.method)
			if len(selections) != len(row.guards) {
				t.Fatalf("switch count=%d want%d", len(selections), len(row.guards))
			}
			for i, selection := range selections {
				enumNull184CheckTag(t, ctx, selection, row.guards[i])
			}
		})
	}
}

func TestEnumNull184LiveOwnerAndUncertainProof(t *testing.T) {
	for _, name := range []string{"live_enum_toggle", "missing_context", "missing_owner_declaration", "source_mismatch", "wrong_block"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumNull184Context(t, "NullColon", "run")
			selection := selections[0]
			selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
			source := ctx.currentFile.Source
			owner := sourceEnumSwitchOwner(selector, source, ctx)
			if owner == nil {
				t.Fatal("fixture enum proof missing")
			}
			enumNull184CheckTag(t, ctx, selection, true)
			switch name {
			case "live_enum_toggle":
				owner.IsEnum = false
			case "missing_context":
				ctx.currentFile = nil
			case "missing_owner_declaration":
				owner.Class.DeclarationNode = nil
			case "source_mismatch":
				source = append([]byte(nil), source...)
				source[0] = 'q'
			case "wrong_block":
				block = selector
			}
			tag := ast.NewIdent("unchangedTag")
			before := len(Diagnostics())
			if got := canonicalSourceEnumSwitchSelector(tag, selector, block, source, ctx); got != tag {
				t.Fatalf("uncertain proof changed tag: %s", enumSwitch184ExpressionText(t, got))
			}
			if len(Diagnostics()) != before {
				t.Fatal("uncertain proof introduced diagnostics")
			}
			if name == "live_enum_toggle" {
				owner.IsEnum = true
				enumNull184CheckTag(t, ctx, selection, true)
			}
		})
	}
}

func TestEnumNull184OtherSelectorTypesRetainOldTag(t *testing.T) {
	source := `package other;class Probe {enum Band{A}static int integer(int value){switch(value){default:return 0;}}static int text(String value){switch(value){default:return 0;}}static int object(Object value){switch(value){default:return 0;}}static <T extends Enum<T>> int bounded(T value){switch(value){default:return 0;}}static int array(Band[] value){switch(value){default:return 0;}}}`
	for _, name := range []string{"integer", "text", "object", "bounded", "array"} {
		t.Run(name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{"other/Probe.java": source}, "other/Probe.java")
			scope := ctx.currentFile.FindClassScope("Probe")
			ctx = classScopeCtx(scope, ctx)
			methods := scope.FindMethod().ByOriginalName(name)
			if len(methods) != 1 {
				t.Fatal("fixture method missing")
			}
			ctx.localScope = methods[0]
			ctx.usedImports = map[string]bool{}
			ctx.importAliases = map[string]string{}
			var selection *sitter.Node
			var walk func(*sitter.Node)
			walk = func(node *sitter.Node) {
				if node == nil {
					return
				}
				if node.Type() == "switch_expression" || node.Type() == "switch_statement" {
					selection = node
					return
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					walk(child)
				}
			}
			walk(methods[0].DeclarationNode)
			if selection == nil {
				t.Fatal("fixture switch missing")
			}
			enumNull184CheckTag(t, ctx, selection, false)
		})
	}
}

func TestEnumNull184JDK21InitializationAndNullContracts(t *testing.T) {
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>enumnull</groupId><artifactId>control</artifactId><version>1</version></project>", "src/main/java/enumnull/Probe.java": enumNull184Java}
	runCampaignCompilerStrictProjectObservations(t, files, "enumnull.Probe", []campaignCompilerProjectObservation{
		{name: "colon", args: []string{"colon"}, stdout: "SN:-9\n"},
		{name: "arrow", args: []string{"arrow"}, stdout: "SN:-9\n"},
		{name: "expression", args: []string{"expression"}, stdout: "SN:-9\n"},
		{name: "explicit-colon", args: []string{"explicit-colon"}, stdout: "S:4\n"},
		{name: "explicit-arrow", args: []string{"explicit-arrow"}, stdout: "S:4\n"},
		{name: "combined-null-default", args: []string{"combined-null-default"}, stdout: "S:4\n"},
		{name: "nested", args: []string{"nested"}, stdout: "SN:-9\n"},
		{name: "abrupt", args: []string{"abrupt"}, stdout: "SX:-9\n"},
		{name: "nonnull", args: []string{"nonnull"}, stdout: "SI:1\n"},
		{name: "repeated", args: []string{"repeated"}, stdout: "SISN:1\n"},
		{name: "default-only", args: []string{"default-only"}, stdout: "SN:-9\n"},
		{name: "empty", args: []string{"empty"}, stdout: "SN:-9\n"},
		{name: "combined-colon-null", args: []string{"combined-colon-null"}, stdout: "S:4\n"},
		{name: "combined-colon-unmatched", args: []string{"combined-colon-unmatched"}, stdout: "SI:4\n"},
		{name: "combined-unmatched", args: []string{"combined-unmatched"}, stdout: "SI:4\n"},
		{name: "explicit-arrow-default", args: []string{"explicit-arrow-default"}, stdout: "SI:3\n"},
		{name: "explicit-arrow-nonnull", args: []string{"explicit-arrow-nonnull"}, stdout: "SI:1\n"},
		{name: "explicit-colon-default", args: []string{"explicit-colon-default"}, stdout: "SI:3\n"},
		{name: "explicit-colon-nonnull", args: []string{"explicit-colon-nonnull"}, stdout: "SI:1\n"},
	})
}
