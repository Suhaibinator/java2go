package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/stdjava"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const enumReview184CommentJava = `package review;public class Probe{public static void main(String[]args){String result;switch(args[0]){case "colon-selector":result=ColonSelector.run();break;case "colon-body":result=ColonBody.run();break;case "arrow-selector":result=ArrowSelector.run();break;case "arrow-body":result=ArrowBody.run();break;case "defaultonly-selector":result=DefaultOnlySelector.run();break;case "defaultonly-body":result=DefaultOnlyBody.run();break;case "empty-selector":result=EmptySelector.run();break;case "empty-body":result=EmptyBody.run();break;default:throw new AssertionError(args[0]);}System.out.println(result);}}class Trace{static String value="";static void add(String text){value+=text;}}
class ColonSelector {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) /* selector role comment */ {case A:out=1;break;case B:out=2;break;default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class ColonBody {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) {
 // direct block comment
case A:out=1;break;case B:out=2;break;default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class ArrowSelector {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) /* selector role comment */ {case A -> out=1;case B -> out=2;default -> out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class ArrowBody {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) {
 // direct block comment
case A -> out=1;case B -> out=2;default -> out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class DefaultOnlySelector {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) /* selector role comment */ {default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class DefaultOnlyBody {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) {
 // direct block comment
default:out=3;}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class EmptySelector {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) /* selector role comment */ {}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
class EmptyBody {enum Band{A,B;static{Trace.add("I");}}static Band select(){Trace.add("S");return null;}static String run(){Trace.value="";int out=-9;try{switch(select()) {
 // direct block comment
}}catch(NullPointerException e){Trace.add("N");}return Trace.value+":"+out;}}
`
const enumReview184CombinedJava = `package combined;public class Probe{public static void main(String[]args){String result;switch(args[0]){case "arrow-null":result=Arrow.run(0);break;case "arrow-matching":result=Arrow.run(1);break;case "arrow-unmatched":result=Arrow.run(2);break;case "colon-null":result=Colon.run(0);break;case "colon-matching":result=Colon.run(1);break;case "colon-unmatched":result=Colon.run(2);break;case "value-null":result=Value.run(0);break;case "value-matching":result=Value.run(1);break;case "value-unmatched":result=Value.run(2);break;case "fallthrough-null":result=Fallthrough.run(0);break;case "fallthrough-matching":result=Fallthrough.run(1);break;case "fallthrough-unmatched":result=Fallthrough.run(2);break;default:throw new AssertionError(args[0]);}System.out.println(result);}}class Trace{static String value="";static void add(String text){value+=text;}}
class Arrow {enum Band{A,B;static{Trace.add("I");}}static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:Band.B;}static String run(int mode){Trace.value="";int out=-9;switch(select(mode)){case A -> out=1;case null /* null marker */, /* default marker */ default -> out=4;}return Trace.value+":"+out;}}
class Colon {enum Band{A,B;static{Trace.add("I");}}static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:Band.B;}static String run(int mode){Trace.value="";int out=-9;switch(select(mode)){case A:out=1;break;case null /* null marker */, /* default marker */ default:out=4;break;}return Trace.value+":"+out;}}
class Value {enum Band{A,B;static{Trace.add("I");}}static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:Band.B;}static String run(int mode){Trace.value="";int out=-9;out=switch(select(mode)){case A -> 1;case null /* null marker */, /* default marker */ default -> 4;};return Trace.value+":"+out;}}
class Fallthrough {enum Band{A,B;static{Trace.add("I");}}static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:Band.B;}static String run(int mode){Trace.value="";int out=0;switch(select(mode)){case A:Trace.add("A");out+=1;case null /* null marker */, /* default marker */ default:Trace.add("D");out+=4;break;}return Trace.value+":"+out;}}
`

func enumReview184Context(t *testing.T, source, className string) (Ctx, []*sitter.Node) {
	t.Helper()
	ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": source}, "Probe.java")
	scope := ctx.currentFile.FindClassScope(className)
	if scope == nil {
		t.Fatal("fixture class missing")
	}
	ctx = classScopeCtx(scope, ctx)
	methods := scope.FindMethod().ByOriginalName("run")
	if len(methods) != 1 {
		t.Fatal("fixture method missing")
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
	if len(selections) != 1 {
		t.Fatalf("fixture switch count%d", len(selections))
	}
	return ctx, selections
}
func TestEnumReview184CommentsRetainDeclarationAndNullProof(t *testing.T) {
	for _, name := range []string{"ColonSelector", "ColonBody", "ArrowSelector", "ArrowBody", "DefaultOnlySelector", "DefaultOnlyBody", "EmptySelector", "EmptyBody"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184CommentJava, name)
			selection := selections[0]
			selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
			source := ctx.currentFile.Source
			owner := sourceEnumSwitchOwner(selector, source, ctx)
			if owner == nil {
				t.Fatal("fixture enum owner missing")
			}
			for index := range owner.EnumConstants {
				owner.EnumConstants[index].Field.Name = "ownedComment" + owner.EnumConstants[index].Name
			}
			for _, child := range nodeutil.NamedChildrenOf(selection) {
				t.Logf("selection role type=%s extra=%t", child.Type(), child.IsExtra())
			}
			tag := ast.NewIdent("oneTag")
			got := canonicalSourceEnumSwitchSelector(tag, selector, block, source, ctx)
			call, ok := got.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 || call.Args[0] != tag {
				t.Errorf("recognized switch comments disabled enum null guard: %s", enumSwitch184ExpressionText(t, got))
			}
			for _, arm := range nodeutil.SemanticNamedChildrenOf(block) {
				for _, label := range nodeutil.SemanticNamedChildrenOf(arm) {
					if label.Type() != "switch_label" {
						continue
					}
					for _, expr := range nodeutil.SemanticNamedChildrenOf(label) {
						if expr.Type() != "identifier" {
							continue
						}
						name := expr.Content(source)
						for _, constant := range owner.EnumConstants {
							if constant.Name == name {
								actual := enumSwitch184ExpressionText(t, parseSwitchLabelExpression(expr, source, ctx, nil))
								want := enumSwitch184ExpressionText(t, staticFieldStorageExpr(&staticFieldAccess{resolution: &fieldResolution{def: constant.Field, owner: owner}}, ctx))
								if actual != want {
									t.Errorf("comment selection lost declaration storage: got%s want%s", actual, want)
								}
							}
						}
					}
				}
			}
		})
	}
}
func TestEnumReview184CombinedNullDefaultExistingSplitContract(t *testing.T) {
	for _, name := range []string{"Arrow", "Colon", "Value", "Fallthrough"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184CombinedJava, name)
			source := ctx.currentFile.Source
			found := 0
			for _, arm := range nodeutil.SemanticNamedChildrenOf(selections[0].ChildByFieldName("body")) {
				combined := false
				for _, label := range nodeutil.SemanticNamedChildrenOf(arm) {
					if label.Type() != "switch_label" {
						continue
					}
					if strings.Contains(label.Content(source), "default") && strings.Contains(label.Content(source), "null") {
						combined = true
						for _, child := range nodeutil.NamedChildrenOf(label) {
							t.Logf("direct combined marker type=%s extra=%t text=%q", child.Type(), child.IsExtra(), child.Content(source))
						}
					}
				}
				if !combined {
					continue
				}
				found++
				var cases []ast.Expr
				var isDefault bool
				if arm.Type() == "switch_rule" {
					cases, isDefault, _ = splitSwitchRule(arm, source, ctx, nil)
				} else {
					cases, isDefault, _ = splitSwitchGroup(arm, source, ctx, nil)
				}
				if !isDefault {
					t.Errorf("case null, default lost default marker: cases%v", cases)
				}
				for _, expr := range cases {
					if strings.Contains(enumSwitch184ExpressionText(t, expr), "default_") {
						t.Error("case null, default emitted keyword as value")
					}
				}
			}
			if found != 1 {
				t.Fatalf("fixture combined marker count%d", found)
			}
		})
	}
}
func TestEnumReview184CommentedSwitchJDKGo(t *testing.T) {
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>review</groupId><artifactId>comments</artifactId><version>1</version></project>", "src/main/java/review/Probe.java": enumReview184CommentJava}
	runCampaignCompilerStrictProjectObservations(t, files, "review.Probe", []campaignCompilerProjectObservation{
		{name: "colon-selector", args: []string{"colon-selector"}, stdout: "SN:-9\n"},
		{name: "colon-body", args: []string{"colon-body"}, stdout: "SN:-9\n"},
		{name: "arrow-selector", args: []string{"arrow-selector"}, stdout: "SN:-9\n"},
		{name: "arrow-body", args: []string{"arrow-body"}, stdout: "SN:-9\n"},
		{name: "defaultonly-selector", args: []string{"defaultonly-selector"}, stdout: "SN:-9\n"},
		{name: "defaultonly-body", args: []string{"defaultonly-body"}, stdout: "SN:-9\n"},
		{name: "empty-selector", args: []string{"empty-selector"}, stdout: "SN:-9\n"},
		{name: "empty-body", args: []string{"empty-body"}, stdout: "SN:-9\n"},
	})
}
func TestEnumReview184CombinedSwitchJDKGo(t *testing.T) {
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>combined</groupId><artifactId>control</artifactId><version>1</version></project>", "src/main/java/combined/Probe.java": enumReview184CombinedJava}
	runCampaignCompilerStrictProjectObservations(t, files, "combined.Probe", []campaignCompilerProjectObservation{
		{name: "arrow-null", args: []string{"arrow-null"}, stdout: "S:4\n"},
		{name: "arrow-matching", args: []string{"arrow-matching"}, stdout: "SI:1\n"},
		{name: "arrow-unmatched", args: []string{"arrow-unmatched"}, stdout: "SI:4\n"},
		{name: "colon-null", args: []string{"colon-null"}, stdout: "S:4\n"},
		{name: "colon-matching", args: []string{"colon-matching"}, stdout: "SI:1\n"},
		{name: "colon-unmatched", args: []string{"colon-unmatched"}, stdout: "SI:4\n"},
		{name: "value-null", args: []string{"value-null"}, stdout: "S:4\n"},
		{name: "value-matching", args: []string{"value-matching"}, stdout: "SI:1\n"},
		{name: "value-unmatched", args: []string{"value-unmatched"}, stdout: "SI:4\n"},
		{name: "fallthrough-null", args: []string{"fallthrough-null"}, stdout: "SD:4\n"},
		{name: "fallthrough-matching", args: []string{"fallthrough-matching"}, stdout: "SIAD:5\n"},
		{name: "fallthrough-unmatched", args: []string{"fallthrough-unmatched"}, stdout: "SID:4\n"},
	})
}

const enumReview184MultiJava = `package multi;public class Probe{public static void main(String[]args){String out;switch(args[0]){case "colon-null":out=Colon.run(0);break;case "colon-first":out=Colon.run(1);break;case "colon-second":out=Colon.run(2);break;case "colon-default":out=Colon.run(3);break;case "arrow-null":out=Arrow.run(0);break;case "arrow-first":out=Arrow.run(1);break;case "arrow-second":out=Arrow.run(2);break;case "arrow-default":out=Arrow.run(3);break;default:throw new AssertionError(args[0]);}System.out.println(out);}}class Trace{static String value="";static void add(String text){value+=text;}}
class Colon{enum Band{A,B,C;static{Trace.add("I");}}static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:mode==2?Band.B:Band.C;}static String run(int mode){Trace.value="";int out=-9;try{switch(select(mode)) /* selector role */ {case A, /* comma label comment */ B:out=5;break;default:out=3;}}catch(NullPointerException ex){Trace.add("N");}return Trace.value+":"+out;}}
class Arrow{enum Band{A,B,C;static{Trace.add("I");}}static Band select(int mode){Trace.add("S");return mode==0?null:mode==1?Band.A:mode==2?Band.B:Band.C;}static String run(int mode){Trace.value="";int out=-9;try{switch(select(mode)) /* selector role */ {case A, /* comma label comment */ B -> out=5;default -> out=3;}}catch(NullPointerException ex){Trace.add("N");}return Trace.value+":"+out;}}
`

func TestEnumReview184CommaLabelsRetainAllConstants(t *testing.T) {
	for _, name := range []string{"Colon", "Arrow"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184MultiJava, name)
			selection := selections[0]
			source := ctx.currentFile.Source
			owner := sourceEnumSwitchOwner(selection.ChildByFieldName("condition"), source, ctx)
			if owner == nil {
				t.Fatal("fixture enum missing")
			}
			for index := range owner.EnumConstants {
				owner.EnumConstants[index].Field.Name = "ownedComma" + owner.EnumConstants[index].Name
			}
			checked := 0
			for _, arm := range nodeutil.SemanticNamedChildrenOf(selection.ChildByFieldName("body")) {
				for _, label := range nodeutil.SemanticNamedChildrenOf(arm) {
					if label.Type() != "switch_label" || !strings.Contains(label.Content(source), "comma label comment") {
						continue
					}
					checked++
					var cases []ast.Expr
					var isDefault bool
					if arm.Type() == "switch_rule" {
						cases, isDefault, _ = splitSwitchRule(arm, source, ctx, nil)
					} else {
						cases, isDefault, _ = splitSwitchGroup(arm, source, ctx, nil)
					}
					if isDefault || len(cases) != 2 {
						t.Errorf("ordinary comma label lost exact two constants: count%d default%t", len(cases), isDefault)
						continue
					}
					for index, name := range []string{"A", "B"} {
						var field *staticFieldAccess
						for _, constant := range owner.EnumConstants {
							if constant.Name == name {
								field = &staticFieldAccess{resolution: &fieldResolution{def: constant.Field, owner: owner}}
							}
						}
						if field == nil {
							t.Fatal("fixture field missing")
						}
						got := enumSwitch184ExpressionText(t, cases[index])
						want := enumSwitch184ExpressionText(t, staticFieldStorageExpr(field, ctx))
						if got != want {
							t.Errorf("ordinary comma label lost owned declaration: got%s want%s", got, want)
						}
					}
				}
			}
			if checked != 1 {
				t.Fatalf("fixture comma arm count%d", checked)
			}
		})
	}
}
func TestEnumReview184CommaLabelJDKGo(t *testing.T) {
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>multi</groupId><artifactId>labels</artifactId><version>1</version></project>", "src/main/java/multi/Probe.java": enumReview184MultiJava}
	runCampaignCompilerStrictProjectObservations(t, files, "multi.Probe", []campaignCompilerProjectObservation{
		{name: "colon-null", args: []string{"colon-null"}, stdout: "SN:-9\n"},
		{name: "colon-first", args: []string{"colon-first"}, stdout: "SI:5\n"},
		{name: "colon-second", args: []string{"colon-second"}, stdout: "SI:5\n"},
		{name: "colon-default", args: []string{"colon-default"}, stdout: "SI:3\n"},
		{name: "arrow-null", args: []string{"arrow-null"}, stdout: "SN:-9\n"},
		{name: "arrow-first", args: []string{"arrow-first"}, stdout: "SI:5\n"},
		{name: "arrow-second", args: []string{"arrow-second"}, stdout: "SI:5\n"},
		{name: "arrow-default", args: []string{"arrow-default"}, stdout: "SI:3\n"},
	})
}

func TestEnumReview184CombinedMarkerRejectsUnprovenSource(t *testing.T) {
	for _, name := range []string{"missing_context", "truncated_source", "changed_keyword", "ordinary_comma", "foreign_tree", "source_mismatch", "wrong_syntax_node"} {
		t.Run(name, func(t *testing.T) {
			java, className := enumReview184CombinedJava, "Arrow"
			if name == "ordinary_comma" {
				java, className = enumReview184MultiJava, "Colon"
			}
			ctx, selections := enumReview184Context(t, java, className)
			source := ctx.currentFile.Source
			marker := "default marker"
			if name == "ordinary_comma" {
				marker = "comma label comment"
			}
			var label *sitter.Node
			for _, arm := range nodeutil.SemanticNamedChildrenOf(selections[0].ChildByFieldName("body")) {
				for _, child := range nodeutil.SemanticNamedChildrenOf(arm) {
					if child.Type() == "switch_label" && strings.Contains(child.Content(source), marker) {
						label = child
					}
				}
			}
			if label == nil {
				t.Fatal("fixture marker missing")
			}
			if name != "ordinary_comma" && !combinedNullDefaultSwitchLabel(label, source, ctx) {
				t.Fatal("fixture proven marker missing")
			}
			var keepAlive any
			switch name {
			case "missing_context":
				ctx.currentFile = nil
			case "truncated_source":
				source = append([]byte(nil), source[:label.EndByte()-1]...)
				ctx.currentFile.Source = source
			case "foreign_tree":
				foreign := &parsing.SourceFile{Name: "Foreign.java", Source: append([]byte(nil), source...)}
				if err := foreign.ParseAST(); err != nil {
					t.Fatal(err)
				}
				keepAlive = foreign
				label = nil
				var walk func(*sitter.Node)
				walk = func(node *sitter.Node) {
					if node == nil || label != nil {
						return
					}
					if node.Type() == "switch_label" && strings.Contains(node.Content(source), marker) {
						label = node
						return
					}
					for _, child := range nodeutil.NamedChildrenOf(node) {
						walk(child)
					}
				}
				walk(foreign.Ast)
				if label == nil {
					t.Fatal("fixture foreign marker missing")
				}
			case "source_mismatch":
				source = append([]byte(nil), source...)
				source[0] = 'q'
			case "wrong_syntax_node":
				label = label.Parent()
			case "changed_keyword":
				source = append([]byte(nil), source...)
				children := nodeutil.SemanticNamedChildrenOf(label)
				copy(source[children[1].StartByte():children[1].EndByte()], []byte("defaxlt"))
				ctx.currentFile.Source = source
			}
			before := len(Diagnostics())
			if combinedNullDefaultSwitchLabel(label, source, ctx) {
				t.Fatal("unproven syntax acquired default semantics")
			}
			if len(Diagnostics()) != before {
				t.Fatal("pure marker proof introduced diagnostics")
			}
			runtime.KeepAlive(keepAlive)
		})
	}
}

func enumDispatch184EmittedSwitch(t *testing.T, ctx Ctx, node *sitter.Node, value bool) *ast.SwitchStmt {
	t.Helper()
	var emitted ast.Node
	if value {
		emitted = buildSwitchExpressionIIFE(node, ctx.currentFile.Source, ctx)
	} else {
		emitted = ParseStmt(node, ctx.currentFile.Source, ctx)
	}
	var result *ast.SwitchStmt
	count := 0
	ast.Inspect(emitted, func(node ast.Node) bool {
		if selection, ok := node.(*ast.SwitchStmt); ok {
			result = selection
			count++
		}
		return true
	})
	if count != 1 {
		t.Fatalf("fixture emitted switch count%d", count)
	}
	return result
}
func enumDispatch184SemanticReferenceCondition(t *testing.T, expression ast.Expr, name string, null bool) {
	t.Helper()
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		t.Fatalf("reference condition not semantic: %s", enumSwitch184ExpressionText(t, expression))
	}
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fun.Sel.Name != "JavaReferenceEqual" {
		t.Fatal("condition does not preserve Java null/identity")
	}
	value, ok := call.Args[0].(*ast.Ident)
	if !ok || value.Name != name {
		t.Fatal("condition changed staged selector")
	}
	if null {
		nilValue, ok := call.Args[1].(*ast.Ident)
		if !ok || nilValue.Name != "nil" {
			t.Fatal("null condition changed literalnull")
		}
	}
}
func enumDispatch184AssertSafeNullSelection(t *testing.T, selection *ast.SwitchStmt) string {
	t.Helper()
	stage, ok := selection.Init.(*ast.AssignStmt)
	if !ok || selection.Tag != nil || stage.Tok != token.DEFINE || len(stage.Lhs) != 1 || len(stage.Rhs) != 1 {
		t.Errorf("direct-null enum dispatch compares nullable storage before default: tag=%s init=%T", enumSwitch184ExpressionText(t, selection.Tag), selection.Init)
		return ""
	}
	name, ok := stage.Lhs[0].(*ast.Ident)
	if !ok || name.Name == "" {
		t.Fatal("selector stage missing exact inferred variable")
	}
	compared := 0
	for _, statement := range selection.Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok {
			t.Fatal("fixture noncase")
		}
		for _, expression := range clause.List {
			if condition, ok := expression.(*ast.BinaryExpr); ok && condition.Op == token.LAND {
				guard, ok := condition.X.(*ast.UnaryExpr)
				if !ok || guard.Op != token.NOT {
					t.Fatal("enum case lacks semantic shortcircuit nullguard")
				}
				enumDispatch184SemanticReferenceCondition(t, guard.X, name.Name, true)
				enumDispatch184SemanticReferenceCondition(t, condition.Y, name.Name, false)
				compared++
			} else {
				enumDispatch184SemanticReferenceCondition(t, expression, name.Name, true)
			}
		}
	}
	if compared == 0 {
		t.Fatal("fixture enum comparisons missing")
	}
	return name.Name
}

func TestEnumDispatch184ExistingAPINullDefaultStorageSafety(t *testing.T) {
	for _, name := range []string{"Arrow", "Colon", "Value", "Fallthrough"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184CombinedJava, name)
			selection := enumDispatch184EmittedSwitch(t, ctx, selections[0], name == "Value")
			enumDispatch184AssertSafeNullSelection(t, selection)
		})
	}
}
func TestEnumDispatch184NoDirectNullPreservesPointerSwitch(t *testing.T) {
	ctx, selections := enumNull184Context(t, "NullColon", "run")
	selection := enumDispatch184EmittedSwitch(t, ctx, selections[0], false)
	if selection.Init != nil || selection.Tag == nil {
		t.Fatal("ordinary enum switch no longer retains checked pointer tag")
	}
}
func TestEnumDispatch184ExistingAPITemporaryAvoidsSourceAndLiveNames(t *testing.T) {
	source := `package names;class Probe{enum Band{A,B}static int run(Band value){int __java2goEnumSwitchTag=41;int __java2goEnumSwitchTag_0=42;switch(value){case A:return __java2goEnumSwitchTag;case null,default:return __java2goEnumSwitchTag+1;}}}`
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "source-name", true: "live-emitted-name"}[mutate], func(t *testing.T) {
			ctx, selections := enumReview184Context(t, source, "Probe")
			if mutate {
				found := false
				for _, definition := range ctx.localScope.Children {
					if definition != nil && definition.OriginalName == "__java2goEnumSwitchTag" {
						definition.Name = "__java2goEnumSwitchTag_0"
						found = true
					}
				}
				if !found {
					t.Fatal("fixture local definition missing")
				}
			}
			selection := enumDispatch184EmittedSwitch(t, ctx, selections[0], false)
			name := enumDispatch184AssertSafeNullSelection(t, selection)
			if name == "__java2goEnumSwitchTag" || name == "__java2goEnumSwitchTag_0" {
				t.Fatalf("selector stage captured live source/emitted name%s", name)
			}
		})
	}
}

const enumDispatch184ErasedJava = `package erased;
public class Probe {
 static String trace="";
 enum Band {A,B;static{trace+="I";}}
 static Band select(int mode){trace+="S";return mode==0?null:mode==1?Band.A:Band.B;}
 public static void main(String[]args){int out;switch(args[0]){
 case "field-null":out=FieldCase.run(new Cell<Band>(select(0)));break;
 case "field-matching":out=FieldCase.run(new Cell<Band>(select(1)));break;
 case "field-unmatched":out=FieldCase.run(new Cell<Band>(select(2)));break;
 case "method-null":out=MethodCase.run(new Cell<Band>(select(0)));break;
 case "method-matching":out=MethodCase.run(new Cell<Band>(select(1)));break;
 case "method-unmatched":out=MethodCase.run(new Cell<Band>(select(2)));break;
 case "sole-null":out=OnlyDefault.run(new Cell<Band>(select(0)));break;
 case "sole-nonnull":out=OnlyDefault.run(new Cell<Band>(select(1)));break;
 default:throw new AssertionError(args[0]);}
 System.out.println(trace+":"+out);
 }
}
class Cell<T>{T value;Cell(T value){this.value=value;}T get(){return value;}}
class FieldCase{static int run(Cell<Probe.Band>cell){switch(cell.value){case A:return 1;case null,default:return 4;}}}
class MethodCase{static int run(Cell<Probe.Band>cell){switch(cell.get()){case A:return 1;case null,default:return 4;}}}
class OnlyDefault{static int run(Cell<Probe.Band>cell){int out=-9;switch(cell.get()){case null,default -> out=4;}return out;}}
`

func TestEnumDispatch184ErasedSelectorsExistingAPI(t *testing.T) {
	for _, name := range []string{"FieldCase", "MethodCase"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumDispatch184ErasedJava, name)
			selector := selections[0].ChildByFieldName("condition")
			if sourceEnumSwitchOwner(selector, ctx.currentFile.Source, ctx) == nil {
				t.Error("erased concrete source enum selector lost declaration provenance")
			}
			selection := enumDispatch184EmittedSwitch(t, ctx, selections[0], false)
			enumDispatch184AssertSafeNullSelection(t, selection)
		})
	}
}
func TestEnumDispatch184SemanticTypedNilAndSoleDefaultControls(t *testing.T) {
	var pointer *struct{ value int }
	enumDispatch191AssertTypedNil(t, pointer)
	ctx, selections := enumReview184Context(t, enumDispatch184ErasedJava, "OnlyDefault")
	selection := enumDispatch184EmittedSwitch(t, ctx, selections[0], false)
	if selection.Init != nil || selection.Tag == nil || len(selection.Body.List) != 1 || len(selection.Body.List[0].(*ast.CaseClause).List) != 0 {
		t.Fatal("sole null/default introduced an unused stage or changed existing default")
	}
}
func TestEnumDispatch184ErasedSelectorsJDKGo(t *testing.T) {
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>erased</groupId><artifactId>nullable</artifactId><version>1</version></project>", "src/main/java/erased/Probe.java": enumDispatch184ErasedJava}
	runCampaignCompilerStrictProjectObservations(t, files, "erased.Probe", []campaignCompilerProjectObservation{
		{name: "field-null", args: []string{"field-null"}, stdout: "S:4\n"},
		{name: "field-matching", args: []string{"field-matching"}, stdout: "SI:1\n"},
		{name: "field-unmatched", args: []string{"field-unmatched"}, stdout: "SI:4\n"},
		{name: "method-null", args: []string{"method-null"}, stdout: "S:4\n"},
		{name: "method-matching", args: []string{"method-matching"}, stdout: "SI:1\n"},
		{name: "method-unmatched", args: []string{"method-unmatched"}, stdout: "SI:4\n"},
		{name: "sole-null", args: []string{"sole-null"}, stdout: "S:4\n"},
		{name: "sole-nonnull", args: []string{"sole-nonnull"}, stdout: "SI:4\n"},
	})
}

func TestEnumDispatch184PlanProofPrecedesMutation(t *testing.T) {
	for _, name := range []string{"positive-body-identity", "preexisting-init", "missing-context", "clause-count", "duplicate-clause", "nil-clause", "null-slot-mismatch", "missing-constant-field", "changed-keyword"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184CombinedJava, "Arrow")
			selection := selections[0]
			selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
			evidence := newEnumSwitchLoweringEvidence(selector, block, ctx)
			source := ctx.currentFile.Source
			originalTag := ast.NewIdent("originalLoweredTag")
			owner := sourceEnumSwitchOwner(selector, source, ctx)
			if owner == nil || len(owner.EnumConstants) == 0 || owner.EnumConstants[0].Field == nil {
				t.Fatal("fixture declaration missing")
			}
			firstSourceArm := nodeutil.SemanticNamedChildrenOf(block)[0]
			firstSourceLabel := nodeutil.SemanticNamedChildrenOf(firstSourceArm)[0]
			originalCase := parseSwitchLabelExpression(nodeutil.SemanticNamedChildrenOf(firstSourceLabel)[0], source, ctx, evidence)
			firstBody := &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("firstBody")}}
			defaultBody := &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("defaultBody")}}
			first := &ast.CaseClause{List: []ast.Expr{originalCase}, Body: []ast.Stmt{firstBody}}
			fallback := &ast.CaseClause{Body: []ast.Stmt{defaultBody}}
			body := &ast.BlockStmt{List: []ast.Stmt{first, fallback}}
			statement := &ast.SwitchStmt{Tag: originalTag, Body: body}
			var originalInit ast.Stmt
			switch name {
			case "preexisting-init":
				originalInit = &ast.EmptyStmt{}
				statement.Init = originalInit
			case "missing-context":
				ctx.currentFile = nil
			case "clause-count":
				body.List = body.List[:1]
			case "duplicate-clause":
				body.List[1] = first
			case "nil-clause":
				body.List[0] = (*ast.CaseClause)(nil)
			case "null-slot-mismatch":
				first.List[0] = ast.NewIdent("nil")
				originalCase = first.List[0]
			case "missing-constant-field":
				owner := sourceEnumSwitchOwner(selector, source, ctx)
				if owner == nil {
					t.Fatal("fixture enum missing")
				}
				owner.EnumConstants[0].Field = nil
			case "changed-keyword":
				source = append([]byte(nil), source...)
				position := strings.Index(string(source), "/* default marker */ default")
				if position < 0 {
					t.Fatal("fixture keyword missing")
				}
				position += len("/* default marker */ ")
				copy(source[position:position+7], []byte("defaxlt"))
				ctx.currentFile.Source = source
			}
			diagnosticsBefore := len(Diagnostics())
			importsBefore := len(ctx.usedImports)
			aliasesBefore := len(ctx.importAliases)
			got := canonicalSourceEnumNullSwitch(statement, selector, block, source, ctx, evidence)
			if got != statement || got.Body != body || first.Body[0] != firstBody || fallback.Body[0] != defaultBody {
				t.Fatal("dispatch changed clause/body identity")
			}
			if name == "positive-body-identity" {
				stage, ok := got.Init.(*ast.AssignStmt)
				if !ok || len(stage.Rhs) != 1 || stage.Rhs[0] != originalTag || got.Tag != nil {
					t.Fatal("selector was not staged exactly once by identity")
				}
				enumDispatch184AssertSafeNullSelection(t, got)
			} else {
				if got.Init != originalInit || got.Tag != originalTag || first.List[0] != originalCase {
					t.Fatal("unproven plan partially mutated AST")
				}
				if len(ctx.usedImports) != importsBefore || len(ctx.importAliases) != aliasesBefore {
					t.Fatal("unproven plan changed imports")
				}
			}
			if len(Diagnostics()) != diagnosticsBefore {
				t.Fatal("plan proof introduced diagnostics")
			}
		})
	}
}

func TestEnumDispatch184StorageCorrespondenceBeforeMutation(t *testing.T) {
	for _, name := range []string{"exact-storage", "wrong-constant", "arbitrary-call", "stale-field-name", "stale-package-alias", "permuted-labels"} {
		t.Run(name, func(t *testing.T) {
			java := `package consumer;import owned.Band;class Probe{static int run(Band band){switch(band){case A,B:return 1;case null,default:return 4;}}}`
			ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": java, "Band.java": `package owned;public enum Band{A,B}`}, "Probe.java")
			scope := ctx.currentFile.FindClassScope("Probe")
			if scope == nil {
				t.Fatal("fixture class missing")
			}
			ctx = classScopeCtx(scope, ctx)
			methods := scope.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("fixture method missing")
			}
			ctx.localScope = methods[0]
			ctx.usedImports = map[string]bool{}
			ctx.importAliases = map[string]string{}
			var selection *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n == nil {
					return
				}
				if n.Type() == "switch_expression" || n.Type() == "switch_statement" {
					selection = n
					return
				}
				for _, child := range nodeutil.SemanticNamedChildrenOf(n) {
					walk(child)
				}
			}
			walk(methods[0].DeclarationNode)
			if selection == nil {
				t.Fatal("fixture switch missing")
			}
			selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
			evidence := newEnumSwitchLoweringEvidence(selector, block, ctx)
			source := ctx.currentFile.Source
			owner := sourceEnumSwitchOwner(selector, source, ctx)
			if owner == nil || len(owner.EnumConstants) != 2 {
				t.Fatal("fixture source enum missing")
			}
			arms := nodeutil.SemanticNamedChildrenOf(block)
			label := nodeutil.SemanticNamedChildrenOf(arms[0])[0]
			labels := nodeutil.SemanticNamedChildrenOf(label)
			if len(labels) != 2 {
				t.Fatal("fixture labels missing")
			}
			first := &ast.CaseClause{List: []ast.Expr{parseSwitchLabelExpression(labels[0], source, ctx, evidence), parseSwitchLabelExpression(labels[1], source, ctx, evidence)}, Body: []ast.Stmt{&ast.ReturnStmt{}}}
			fallback := &ast.CaseClause{Body: []ast.Stmt{&ast.ReturnStmt{}}}
			statement := &ast.SwitchStmt{Tag: ast.NewIdent("originalTag"), Body: &ast.BlockStmt{List: []ast.Stmt{first, fallback}}}
			switch name {
			case "wrong-constant":
				first.List[0] = first.List[1]
			case "arbitrary-call":
				first.List[0] = &ast.CallExpr{Fun: ast.NewIdent("unrelatedSideEffect")}
			case "stale-field-name":
				owner.EnumConstants[0].Field.Name += "Changed"
			case "stale-package-alias":
				ctx.importAliases["owned"] = "changedAlias"
			case "permuted-labels":
				first.List[0], first.List[1] = first.List[1], first.List[0]
			}
			originalTag, originalBody := statement.Tag, statement.Body
			originalExpressions := append([]ast.Expr(nil), first.List...)
			imports, aliases := map[string]bool{}, map[string]string{}
			for k, v := range ctx.usedImports {
				imports[k] = v
			}
			for k, v := range ctx.importAliases {
				aliases[k] = v
			}
			diagnostics := len(Diagnostics())
			got := canonicalSourceEnumNullSwitch(statement, selector, block, source, ctx, evidence)
			if name == "exact-storage" {
				enumDispatch184AssertSafeNullSelection(t, got)
				return
			}
			unchanged := got == statement && got.Init == nil && got.Tag == originalTag && got.Body == originalBody && len(first.List) == len(originalExpressions)
			if unchanged {
				for i, x := range originalExpressions {
					if first.List[i] != x {
						unchanged = false
					}
				}
			}
			if !unchanged || !reflect.DeepEqual(imports, ctx.usedImports) || !reflect.DeepEqual(aliases, ctx.importAliases) || len(Diagnostics()) != diagnostics {
				t.Error("uncertified storage accepted or refusal produced AST/import/diagnostic side effects")
			}
		})
	}
}

func TestEnumDispatch184CertificateLifetimeAndStorageShapes(t *testing.T) {
	for _, name := range []string{"plain", "shadowed-helper", "moved-root", "changed-name", "changed-helper-arguments", "format-inspect"} {
		t.Run(name, func(t *testing.T) {
			ctx, selections := enumReview184Context(t, enumReview184CombinedJava, "Arrow")
			selection := selections[0]
			selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
			evidence := newEnumSwitchLoweringEvidence(selector, block, ctx)
			source := ctx.currentFile.Source
			owner := sourceEnumSwitchOwner(selector, source, ctx)
			if owner == nil || len(owner.EnumConstants) == 0 || len(ctx.localScope.Parameters) == 0 {
				t.Fatal("fixture enum/parameter missing")
			}
			field := owner.EnumConstants[0].Field
			if name == "shadowed-helper" || name == "changed-helper-arguments" {
				field.Name = ctx.localScope.Parameters[0].Name
			}
			label := nodeutil.SemanticNamedChildrenOf(nodeutil.SemanticNamedChildrenOf(nodeutil.SemanticNamedChildrenOf(block)[0])[0])[0]
			expression := parseSwitchLabelExpression(label, source, ctx, evidence)
			if !enumSwitchStorageCorresponds(expression, label, field, owner, ctx, evidence) {
				t.Fatal("fresh actual compiler storage lacks correspondence")
			}
			beforeImports, beforeAliases := map[string]bool{}, map[string]string{}
			for k, v := range ctx.usedImports {
				beforeImports[k] = v
			}
			for k, v := range ctx.importAliases {
				beforeAliases[k] = v
			}
			diagnostics := len(Diagnostics())
			var carrier *ast.Ident
			switch x := expression.(type) {
			case *ast.Ident:
				carrier = x
			case *ast.StarExpr:
				carrier = x.X.(*ast.CallExpr).Fun.(*ast.Ident)
			default:
				t.Fatalf("unexpected fixture storage %T", expression)
			}
			switch name {
			case "shadowed-helper":
				if _, ok := expression.(*ast.StarExpr); !ok {
					t.Fatal("actual shadowed storage helper missing")
				}
			case "moved-root":
				original := expression
				expression = &ast.Ident{Name: carrier.Name}
				evidence.storage[expression] = evidence.storage[original]
			case "changed-name":
				carrier.Name += "Changed"
			case "changed-helper-arguments":
				expression.(*ast.StarExpr).X.(*ast.CallExpr).Args = []ast.Expr{ast.NewIdent("injected")}
			case "format-inspect":
				var annotated, ordinary bytes.Buffer
				if err := format.Node(&annotated, token.NewFileSet(), expression); err != nil {
					t.Fatal(err)
				}
				annotatedCount := 0
				ast.Inspect(expression, func(n ast.Node) bool {
					if n != nil {
						annotatedCount++
					}
					return true
				})
				certificate := evidence.storage[expression]
				delete(evidence.storage, expression)
				if err := format.Node(&ordinary, token.NewFileSet(), expression); err != nil {
					t.Fatal(err)
				}
				ordinaryCount := 0
				ast.Inspect(expression, func(n ast.Node) bool {
					if n != nil {
						ordinaryCount++
					}
					return true
				})
				evidence.storage[expression] = certificate
				if !bytes.Equal(annotated.Bytes(), ordinary.Bytes()) || annotatedCount != ordinaryCount {
					t.Fatal("certificate changed printing or AST traversal")
				}
			}
			got := enumSwitchStorageCorresponds(expression, label, field, owner, ctx, evidence)
			want := name == "plain" || name == "shadowed-helper" || name == "format-inspect"
			if got != want {
				t.Fatalf("storage correspondence=%t want %t", got, want)
			}
			if !reflect.DeepEqual(beforeImports, ctx.usedImports) || !reflect.DeepEqual(beforeAliases, ctx.importAliases) || len(Diagnostics()) != diagnostics {
				t.Fatal("pure certificate check changed imports/diagnostics")
			}
		})
	}
	t.Run("existing-object-preserved", func(t *testing.T) {
		ctx, selections := enumReview184Context(t, enumReview184CombinedJava, "Arrow")
		selection := selections[0]
		selector, block := selection.ChildByFieldName("condition"), selection.ChildByFieldName("body")
		evidence := newEnumSwitchLoweringEvidence(selector, block, ctx)
		source := ctx.currentFile.Source
		owner := sourceEnumSwitchOwner(selector, source, ctx)
		label := nodeutil.SemanticNamedChildrenOf(nodeutil.SemanticNamedChildrenOf(nodeutil.SemanticNamedChildrenOf(block)[0])[0])[0]
		parsed, err := parser.ParseFile(token.NewFileSet(), "existing.go", "package existing; var existing int", 0)
		if err != nil {
			t.Fatal(err)
		}
		expression := parsed.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
		binding := expression.Obj
		if binding == nil {
			t.Fatal("fixture lacks genuine Go binding metadata")
		}
		if certifyEnumSwitchStorage(expression, label, owner.EnumConstants[0].Field, owner, ctx, evidence) != expression || expression.Obj != binding {
			t.Fatal("existing Go object metadata overwritten")
		}
		if enumSwitchStorageCorresponds(expression, label, owner.EnumConstants[0].Field, owner, ctx, evidence) {
			t.Fatal("ordinary Go binding treated as private certificate")
		}
	})
}

func enumDispatch191AssertTypedNil(t *testing.T, erased any) {
	t.Helper()
	if erased == nil || !stdjava.JavaReferenceEqual(erased, nil) {
		t.Fatal("existing semantic null API did not distinguish typednil")
	}
}
