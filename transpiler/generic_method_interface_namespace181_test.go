package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"sort"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func physicalInterface181Graph(t *testing.T, sources map[string]string) map[string]parsing.SourceFile {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	var names []string
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	files := map[string]parsing.SourceFile{}
	for _, name := range names {
		file := parsing.SourceFile{Name: name, Source: []byte(sources[name])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files[name] = file
	}
	for _, name := range names {
		ResolveFile(files[name])
	}
	return files
}

func physicalInterface181String(t *testing.T, expression ast.Expr) string {
	t.Helper()
	var result bytes.Buffer
	if err := printer.Fprint(&result, token.NewFileSet(), expression); err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func physicalInterface181Observe(t *testing.T, file parsing.SourceFile, methodName, invokedName string, wantInterface bool) {
	t.Helper()
	class := file.Symbols.FindClassScope("Main")
	if class == nil {
		t.Fatal("actual caller Main missing")
	}
	methods := class.FindMethod().ByOriginalName(methodName)
	if len(methods) != 1 {
		t.Fatal("actual caller method missing")
	}
	ctx := Ctx{currentFile: file.Symbols, currentClass: class, localScope: methods[0], executionContextName: "actualExecution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	var invocations []*sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" && n.ChildByFieldName("object") != nil && n.ChildByFieldName("name").Content(file.Source) == invokedName {
			invocations = append(invocations, n)
		}
		for _, child := range nodeutil.NamedChildrenOf(n) {
			visit(child)
		}
	}
	visit(methods[0].DeclarationNode)
	if len(invocations) != 1 {
		t.Fatalf("actual invocation count=%d", len(invocations))
	}
	invocation := invocations[0]
	owner := resolveClassScopeByIdentifier(ctx, file.Source, invocation.ChildByFieldName("object"))
	selected := findBestMethodInHierarchy(owner, invokedName, invocation.ChildByFieldName("arguments"), false, true, ctx, file.Source)
	if selected == nil || selected.def == nil || selected.owner != owner {
		t.Fatal("actual static declaration not selected")
	}
	slots, declaring, known := declaredGenericMethodPhysicalSlots(selected.def)
	if !known || len(slots) != 1 || slots[0].implicitObject || slots[0].parameter.Declaration != selected.def.TypeParameters[0].Declaration {
		t.Fatal("actual first-bound declaration slot lost")
	}
	bound := resolveClassScopeByQualifiedName(declaring, slots[0].javaType)
	if bound == nil || bound.Class == nil || bound.IsInterface != wantInterface {
		t.Fatalf("actual bound identity unavailable: slot=%+v scope=%p", slots[0], bound)
	}
	physical, javaTypes, ok := genericMethodPhysicalTypeArguments(selected.def, ctx)
	if !ok || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != slots[0].javaType {
		t.Fatal("physical slot changed Java declaration spelling")
	}
	expected := qualifiedNameExpr(bound.Class.Name, findJavaPackageForClassScope(bound), ctx)
	if !wantInterface {
		expected = &ast.StarExpr{X: expected}
	}
	t.Logf("PHYSICAL181 caller=%q/%p method=%p callee=%q/%p declaration=%p binder=%p slot=%q declaringClass=%p bound=%p/%q interface=%t callerWeight=%p actualGoAST=%T:%s expectedGoAST=%T:%s aliases=%v", file.Symbols.Package, class, methods[0], findJavaPackageForClassScope(owner), owner, selected.def, slots[0].parameter.Declaration, slots[0].javaType, declaring.currentClass, bound, bound.Class.Name, bound.IsInterface, resolveClassScopeByQualifiedName(ctx, "Weight"), physical[0], physicalInterface181String(t, physical[0]), expected, physicalInterface181String(t, expected), ctx.importAliases)
	if physicalInterface181String(t, physical[0]) != physicalInterface181String(t, expected) {
		t.Fatalf("declaration-owned physical bound must use caller Go namespace: got %s want %s", physicalInterface181String(t, physical[0]), physicalInterface181String(t, expected))
	}
	var args []ast.Expr
	for _, arg := range nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments")) {
		args = append(args, ParseExpr(arg, file.Source, ctx))
	}
	call := staticGenericPhysicalInvocation(selected, invocation, args, ctx, file.Source, nil)
	if call == nil {
		t.Fatal("actual static physical invocation plan absent")
	}
	callee := executionMethodCallName(selected.def, selected.owner, ctx)
	found := false
	ast.Inspect(call, func(n ast.Node) bool {
		indexed, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}
		name := ""
		switch head := indexed.X.(type) {
		case *ast.Ident:
			name = head.Name
		case *ast.SelectorExpr:
			name = head.Sel.Name
		}
		if name == callee {
			found = true
			if physicalInterface181String(t, indexed.Index) != physicalInterface181String(t, expected) {
				t.Fatal("actual call Go AST lost caller-qualified owned interface")
			}
		}
		return true
	})
	if !found {
		t.Fatal("actual selected function index absent from Go AST")
	}
}

func TestGenericInterfacePhysical181CallerNamespaces(t *testing.T) {
	for _, tc := range []struct {
		name, packageName, declaration, bound, callerDeclarations, callerParameters, implementation string
		iface                                                                                       bool
	}{
		{"nested", "app", "public interface Weight {long value();}", "Policy.Weight", "", "", "implements Policy.Weight", true},
		{"caller-interface", "app", "public interface Weight {long value();}", "Policy.Weight", "interface Weight {int other();}", "", "implements Policy.Weight", true},
		{"caller-binder", "app", "public interface Weight {long value();}", "Policy.Weight", "", "<Weight>", "implements Policy.Weight", true},
		{"import-alias-collision", "app", "public interface Weight {long value();}", "Policy.Weight", "class lib {}", "", "implements Policy.Weight", true},
		{"same-package", "lib", "public interface Weight {long value();}", "Policy.Weight", "", "", "implements Policy.Weight", true},
		{"concrete-class", "app", "public static class Weight {public long value(){return 17;}}", "Policy.Weight", "", "", "extends Policy.Weight", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := physicalInterface181Graph(t, map[string]string{
				"lib/Policy.java": "package lib; public class Policy<ClassBound extends " + tc.bound + "> {" + tc.declaration + " public static <MethodBound extends " + tc.bound + "> long observe(MethodBound value){return value.value();}}",
				"app/Main.java":   "package " + tc.packageName + "; import lib.Policy; " + tc.callerDeclarations + " public class Main" + tc.callerParameters + " {static class Ticket " + tc.implementation + " {public long value(){return 41;}} public static void main(String[] args){long result=Policy.observe(new Ticket());}}",
			})
			physicalInterface181Observe(t, files["app/Main.java"], "main", "observe", tc.iface)
		})
	}
}

func TestGenericInterfacePhysical181TopLevel(t *testing.T) {
	files := physicalInterface181Graph(t, map[string]string{
		"lib/Weight.java": "package lib; public interface Weight {long value();}",
		"lib/Policy.java": "package lib; public class Policy<ClassBound extends Weight> {public static <MethodBound extends Weight> long observe(MethodBound value){return value.value();}}",
		"app/Main.java":   "package app; import lib.Policy; interface Weight {int other();} public class Main {static class Ticket implements lib.Weight {public long value(){return 97;}} public static void main(String[] args){long result=Policy.observe(new Ticket());}}",
	})
	physicalInterface181Observe(t, files["app/Main.java"], "main", "observe", true)
}

func TestGenericInterfacePhysical181RetainsRawArguments(t *testing.T) {
	for _, parameters := range []string{"A", "A, B"} {
		t.Run(parameters, func(t *testing.T) {
			files := physicalInterface181Graph(t, map[string]string{
				"lib/Policy.java": "package lib; public class Policy {public interface Weight<" + parameters + "> {long value();} public static <T extends Weight> long observe(T value){return value.value();}}",
				"app/Main.java":   "package app; public class Main {}",
			})
			caller := files["app/Main.java"]
			ctx := Ctx{currentFile: caller.Symbols, currentClass: caller.Symbols.BaseClass, importAliases: map[string]string{}, usedImports: map[string]bool{}}
			owner := files["lib/Policy.java"].Symbols.BaseClass
			method := owner.FindMethod().ByOriginalName("observe")[0]
			slots, declaring, known := declaredGenericMethodPhysicalSlots(method)
			if !known || len(slots) != 1 {
				t.Fatal("actual raw-interface slot absent")
			}
			before := javaTypeStringToGoTypeExpr(slots[0].javaType, inScopeTypeParameters(declaring), declaring)
			after, _, ok := genericMethodPhysicalTypeArguments(method, ctx)
			if !ok || len(after) != 1 {
				t.Fatal("actual raw-interface physical argument absent")
			}
			var head ast.Expr
			switch original := before.(type) {
			case *ast.IndexExpr:
				projected, same := after[0].(*ast.IndexExpr)
				if !same || physicalInterface181String(t, projected.Index) != physicalInterface181String(t, original.Index) {
					t.Fatal("owned raw interface single physical argument changed")
				}
				head = projected.X
			case *ast.IndexListExpr:
				projected, same := after[0].(*ast.IndexListExpr)
				if !same || len(projected.Indices) != len(original.Indices) {
					t.Fatal("owned raw interface physical argument list changed")
				}
				for i := range original.Indices {
					if physicalInterface181String(t, projected.Indices[i]) != physicalInterface181String(t, original.Indices[i]) {
						t.Fatal("owned raw interface physical argument changed")
					}
				}
				head = projected.X
			default:
				t.Fatalf("actual generic interface representation=%T, expected indexed type", before)
			}
			bound := resolveClassScopeByQualifiedName(declaring, slots[0].javaType)
			expected := qualifiedNameExpr(bound.Class.Name, findJavaPackageForClassScope(bound), ctx)
			if physicalInterface181String(t, head) != physicalInterface181String(t, expected) {
				t.Fatal("raw interface head lost caller qualification")
			}
		})
	}
}
