package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/printer"
	"go/token"
	"maps"
	"reflect"
	"strings"
	"testing"
)

func helpfulNPEContext(t *testing.T, raw, owner, method, called string) (Ctx, *sitter.Node, []byte) {
	t.Helper()
	h := setupParseHelper(t, raw)
	class := h.File.Symbols.FindClassScope(owner)
	if class == nil {
		t.Fatal("missing source owner")
	}
	ctx := classScopeCtx(class, h.Ctx)
	ctx.localScope = class.FindMethodByName(method, nil)
	if ctx.localScope == nil {
		t.Fatal("missing source method")
	}
	ctx.executionContextName = "__npeExecution"
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || call != nil {
			return
		}
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(h.File.Source) == called {
			call = n
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if call == nil {
		t.Fatal("missing invocation")
	}
	return ctx, call, h.File.Source
}
func helpfulNPEText(t *testing.T, x ast.Expr) string {
	t.Helper()
	var out bytes.Buffer
	if err := printer.Fprint(&out, token.NewFileSet(), x); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func TestHelpfulNPEExistingDirectInvocation(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;class Target<T>{boolean isDirect(){return true;}}class Main{static boolean run(Target<String> lane){return lane.isDirect();}}`, "Main", "run", "isDirect")
	out := helpfulNPEText(t, ParseExpr(n, s, ctx))
	if !strings.Contains(out, "RequireInvocationReceiver(") || !strings.Contains(out, `Cannot invoke \"proof.Target.isDirect()\" because \"<parameter1>\" is null`) {
		t.Fatalf("existing API loses source-owned invocation diagnostic: %s", out)
	}
	if !strings.Contains(out, "__npeExecution") {
		t.Fatal("successful call lost original execution")
	}
}
func TestHelpfulNPEExistingInterfaceInvocation(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;interface Face{int read(int value);}class Main{static int run(Face lane){return lane.read(7);}}`, "Main", "run", "read")
	out := helpfulNPEText(t, ParseExpr(n, s, ctx))
	if !strings.Contains(out, "RequireInvocationReceiver(") || !strings.Contains(out, `proof.Face.read(int)`) {
		t.Fatalf("interface path has no canonical null boundary: %s", out)
	}
}

func helpfulNPEResolution(t *testing.T, ctx Ctx, n *sitter.Node, s []byte) *methodResolution {
	t.Helper()
	target := resolveInvocationTarget(n.ChildByFieldName("object"), ctx, s)
	if target == nil {
		t.Fatal("missing actual target")
	}
	resolved, _ := findBestMethodForInvocationTarget(target, n.ChildByFieldName("name").Content(s), n.ChildByFieldName("arguments"), true, true, ctx, s)
	return resolved
}
func TestHelpfulNPESourceDeclarationAndPurity(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;class Target<T>{void accept(T value,String[] strings){}}class Main{static void run(long first,double second,Target<String> map){map.accept(null,null);}}`, "Main", "run", "accept")
	resolved := helpfulNPEResolution(t, ctx, n, s)
	formal := ctx.localScope.Parameters[2]
	// Generated aliases cannot alter Java diagnostic metadata.
	resolved.def.Name = "__renamedMethod"
	resolved.owner.Class.Name = "__renamedClass"
	formal.Name = "__renamedFormal"
	imports, used, aliases := maps.Clone(ctx.currentFile.Imports), maps.Clone(ctx.usedImports), maps.Clone(ctx.importAliases)
	expected := `Cannot invoke "proof.Target.accept(Object, String[])" because "<parameter3>" is null`
	for i := 0; i < 4; i++ {
		plan := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s)
		if plan == nil || plan.message != expected || plan.formal != formal || plan.owner != resolved.owner {
			t.Fatalf("source declaration lost: %+v", plan)
		}
	}
	if !reflect.DeepEqual(imports, ctx.currentFile.Imports) || !reflect.DeepEqual(used, ctx.usedImports) || !reflect.DeepEqual(aliases, ctx.importAliases) {
		t.Fatal("pure diagnostic proof changed imports")
	}
}
func TestHelpfulNPESourceTypeHomonymAndNestedSignature(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`package proof;class Object{}class Target{void accept(Object value){}}class Main{static void run(Target lane){lane.accept(null);}}`, `proof.Target.accept(proof.Object)`},
		{`package proof;class Outer{static class Member{}}class Target{void accept(Outer.Member value){}}class Main{static void run(Target lane){lane.accept(null);}}`, `proof.Target.accept(proof.Outer$Member)`},
		{`package proof;class Target{void accept(Integer a,Boolean b,Class<?> c,Thread d){}}class Main{static void run(Target lane){lane.accept(null,null,null,null);}}`, `proof.Target.accept(java.lang.Integer, java.lang.Boolean, java.lang.Class, java.lang.Thread)`},
	} {
		ctx, n, s := helpfulNPEContext(t, tc.raw, "Main", "run", "accept")
		plan := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), helpfulNPEResolution(t, ctx, n, s), ctx, s)
		if plan == nil || !strings.Contains(plan.message, tc.want) {
			t.Fatalf("canonical signature lost %s: %+v", tc.want, plan)
		}
	}
}
func TestHelpfulNPEArgumentStagingAndVirtualBoundary(t *testing.T) {
	for _, raw := range []string{
		`package proof;class Target{int call(int a,int b){return a+b;}}class Main{static int arg(){return 1;}static int run(Target lane){return lane.call(arg(),arg());}}`,
		`package proof;class Target{int call(int a,int b){return a+b;}}class Child extends Target{int call(int a,int b){return 9;}}class Main{static int arg(){return 1;}static int run(Target lane){return lane.call(arg(),arg());}}`,
		`package proof;interface Target{int call(int a,int b);}class Main{static int arg(){return 1;}static int run(Target lane){return lane.call(arg(),arg());}}`,
	} {
		ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
		out := helpfulNPEText(t, ParseExpr(n, s, ctx))
		guard := strings.Index(out, "RequireInvocationReceiver(")
		last := strings.LastIndex(out, "__java2goInvocationArg1 :=")
		if last < 0 {
			last = strings.LastIndex(out, "__java2goInvocationArg1 ")
		}
		if guard < 0 || last < 0 || guard < last || strings.Count(out, "argJava2goExecution(__npeExecution)") != 2 {
			t.Fatalf("argument evaluation not before single receiver check: %s", out)
		}
		if strings.Count(out, "RequireInvocationReceiver(") != 1 {
			t.Fatal("duplicate null boundary")
		}
	}
}
func TestHelpfulNPEFormalAndExecutableShields(t *testing.T) {
	for _, raw := range []string{
		`class Target{int call(){return 1;}}class Main{static int run(Target lane){lane=null;return lane.call();}}`,
		`class Target{int call(){return 1;}}class Main{static int run(){Target lane=null;return lane.call();}}`,
		`class Target{static int call(){return 1;}}class Main{static int run(Target lane){return lane.call();}}`,
		`class Target{<T>T call(){return null;}}class Main{static Object run(Target lane){return lane.call();}}`,
		`class Target{int call(){return 1;}}class Child extends Target{}class Main{static int run(Child lane){return lane.call();}}`,
		`class Target{int call(){return 1;}}class Main{static<T extends Target>int run(T lane){return lane.call();}}`,
		`class Target{int call(){return 1;}}interface Work{int get();}class Main{static Work run(Target lane){return ()->lane.call();}}`,
		`class Target{int call(){return 1;}}class Main{Target field;int run(Target lane){return field.call();}}`,
	} {
		ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
		resolved := helpfulNPEResolution(t, ctx, n, s)
		if helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s) != nil {
			t.Fatalf("unproved formal/source boundary admitted: %s", raw)
		}
	}
}
func TestHelpfulNPEActualOwnerAndSyntheticShields(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `class Target{int call(){return 1;}}class Main{static int run(Target lane){return lane.call();}}`, "Main", "run", "call")
	r := helpfulNPEResolution(t, ctx, n, s)
	receiver := n.ChildByFieldName("object")
	if helpfulSourceInvocationPlan(n, receiver, r, ctx, s) == nil {
		t.Fatal("control source premise")
	}
	original := r.def
	if helpfulSourceInvocationPlan(n, ctx.localScope.DeclarationNode, r, ctx, s) != nil {
		t.Fatal("unrelated receiver node admitted")
	}
	originalName := original.OriginalName
	original.OriginalName = "poisonedSourceName"
	if helpfulSourceInvocationPlan(n, receiver, r, ctx, s) != nil {
		t.Fatal("mutated source-name metadata admitted")
	}
	original.OriginalName = originalName
	copyDef := *original
	r.def = &copyDef
	copyDef.DeclarationNode = nil
	if helpfulSourceInvocationPlan(n, receiver, r, ctx, s) != nil {
		t.Fatal("AST-less method admitted")
	}
	copyDef = *original
	copyDef.RuntimeDefault = true
	if helpfulSourceInvocationPlan(n, receiver, r, ctx, s) != nil {
		t.Fatal("synthetic platform body admitted")
	}
	r.def = original
	owner := r.owner
	fake := *owner
	fake.Class = &symbol.Definition{OriginalName: owner.Class.OriginalName, DeclarationNode: owner.Class.DeclarationNode}
	r.owner = &fake
	if helpfulSourceInvocationPlan(n, receiver, r, ctx, s) != nil {
		t.Fatal("unregistered same-named owner admitted")
	}
	r.owner = owner
	copyDef = *original
	copyDef.DeclarationNode = ctx.localScope.DeclarationNode
	r.def = &copyDef
	if helpfulSourceInvocationPlan(n, receiver, r, ctx, s) != nil {
		t.Fatal("other actual source method origin admitted")
	}
}

func TestHelpfulNPEDeclarationExecutableBoundaries(t *testing.T) {
	h := setupParseHelper(t, `class Target{} class Main{static class Member{} void run(){class Local{} Object anonymous=new Object(){};}}`)
	seen := map[string]bool{}
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "class_declaration" {
			name := n.ChildByFieldName("name").Content(h.File.Source)
			want := name != "Local"
			if got := helpfulNamedDeclarationBoundary(n); got != want {
				t.Fatalf("declaration boundary %s = %v", name, got)
			}
			seen[name] = true
		}
		if n.Type() == "object_creation_expression" && helpfulNamedDeclarationBoundary(n) {
			t.Fatal("anonymous allocation admitted")
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(h.File.Symbols.ParsedRoot())
	for _, name := range []string{"Target", "Main", "Member", "Local"} {
		if !seen[name] {
			t.Fatalf("missing actual parse node %s", name)
		}
	}
}
