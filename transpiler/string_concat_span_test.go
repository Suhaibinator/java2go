package transpiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func concatSpanFixture(t *testing.T, source, expression string) (Ctx, *sitter.Node) {
	t.Helper()
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx
	for _, owner := range helper.File.Symbols.TopLevelClasses {
		for _, method := range owner.Methods {
			if method.OriginalName != "join" || method.DeclarationNode == nil {
				continue
			}
			ctx = classScopeCtx(owner, ctx)
			ctx.localScope = method
			ctx.executionContextName = "sameExecutionToken"
			var selected *sitter.Node
			var visit func(*sitter.Node)
			visit = func(node *sitter.Node) {
				if node == nil {
					return
				}
				if node.Type() == "binary_expression" && node.Content(helper.File.Source) == expression {
					selected = node
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					visit(child)
				}
			}
			visit(method.DeclarationNode)
			if selected != nil {
				return ctx, selected
			}
		}
	}
	t.Fatalf("missing actual declaration/expression %q", expression)
	return Ctx{}, nil
}

func concatSpanCallName(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.Ident:
		return fun.Name
	}
	return ""
}

func TestCanonicalStringConcatSpanPlanner(t *testing.T) {
	t.Run("legacy-nullable-return-assertion", checkCanonicalNullableConcatAssertionRows)
	positive := []struct{ name, parameters, before, expression string }{
		{"params", "String s,int n,long wide", "", `s+"|"+n+wide`},
		{"qualified", "java.lang.String s,int n", "", `s+"|"+n`},
		{"local", "String input,int n", `String s=input;`, `s+"|"+n`},
		{"multiple-locals", "String input,int n", `String first=input, s=first;`, `s+"|"+n`},
		{"primitive-leading", "int n,String s", "", `n+s+"!"`},
		{"primitive-widths", "String s,byte a,short b,char c,boolean d,int e,long f,float g,double h", "", `s+a+b+c+d+e+f+g+h`},
		{"literals", "String s", "", `s+17+18L+'x'+true+1.25f+2.5d`},
		{"temp-collision", "String _javaConcatValue,int _javaConcatText", "", `_javaConcatValue+"|"+_javaConcatText`},
	}
	for _, row := range positive {
		t.Run("accept/"+row.name, func(t *testing.T) {
			source := "public class SpanProbe {static String join(" + row.parameters + "){" + row.before + "return " + row.expression + ";}}"
			ctx, node := concatSpanFixture(t, source, row.expression)
			before := make(map[string]string)
			for key, value := range ctx.importAliases {
				before[key] = value
			}
			leaves, ok := planCanonicalStringConcatSpan(node, ctx.currentFile.Source, ctx)
			if !ok || len(leaves) < 3 {
				t.Fatalf("eligible scalar span rejected: %q", row.expression)
			}
			after := make(map[string]string)
			for key, value := range ctx.importAliases {
				after[key] = value
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("planning mutated import state")
			}
			lowered, ok := lowerCanonicalStringConcatSpan(node, ctx.currentFile.Source, ctx)
			if !ok {
				t.Fatal("approved plan was not lowered")
			}
			iife, ok := lowered.(*ast.CallExpr)
			if !ok {
				t.Fatalf("not IIFE: %T", lowered)
			}
			function, ok := iife.Fun.(*ast.FuncLit)
			if !ok || len(function.Body.List) != 2*len(leaves)+1 {
				t.Fatal("not one read followed by one conversion per leaf")
			}
			var texts []string
			for index, leaf := range leaves {
				value := function.Body.List[2*index].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
				text := function.Body.List[2*index+1].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
				if len(value.Values) != 1 || len(text.Values) != 1 || value.Names[0].Name == text.Names[0].Name {
					t.Fatal("lost once-only typed staging")
				}
				if row.name == "temp-collision" && (value.Names[0].Name == "_javaConcatValue" || text.Names[0].Name == "_javaConcatText") {
					t.Fatal("temporary shadows source input")
				}
				want := map[string]string{"byte": "JavaStringValueOfInt", "short": "JavaStringValueOfInt", "int": "JavaStringValueOfInt", "char": "JavaStringValueOfChar", "long": "JavaStringValueOfLong", "float": "JavaStringValueOfFloat", "double": "JavaStringValueOfDouble", "boolean": "JavaStringValueOfBoolean"}[leaf.javaType]
				if leaf.text {
					want = "JavaStringTextOperandExecution"
					call := text.Values[0].(*ast.CallExpr)
					execution, ok := call.Args[0].(*ast.Ident)
					if !ok || execution.Name != "sameExecutionToken" {
						t.Fatal("conversion changed execution identity")
					}
				}
				if concatSpanCallName(text.Values[0]) != want {
					t.Fatalf("wrong static conversion for %s", leaf.javaType)
				}
				texts = append(texts, text.Names[0].Name)
			}
			result := function.Body.List[len(function.Body.List)-1].(*ast.ReturnStmt).Results[0]
			if concatSpanCallName(result) != "ConcatJavaStringSpan" {
				t.Fatal("missing final span join")
			}
			for index, arg := range result.(*ast.CallExpr).Args {
				if arg.(*ast.Ident).Name != texts[index] {
					t.Fatal("converted order changed")
				}
			}
		})
	}
	negative := []struct{ name, prefix, parameters, before, expression string }{
		{"two-leaves", "", "String s,int n", "", `s+n`},
		{"constant", "", "String s", "", `"a"+"b"+"c"`},
		{"constant-prefix", "", "String s", "", `"a"+"b"+s`},
		{"final-constant-prefix", "", "String s", `final String a="a",b="b";`, `a+b+s`},
		{"numeric-prefix", "", "int n,String s", "", `n+1+s`},
		{"right-group", "", "String s,String q,int n", "", `s+(q+"|"+n)`},
		{"parenthesized-leaf", "", "String s,int n", "", `(s)+"|"+n`},
		{"field", "static int cell;", "String s,int n", "", `s+"|"+n+cell`},
		{"array", "", "String s,int[] a,int n", "", `s+"|"+n+a[0]`},
		{"call", "static int effect(){return 1;}", "String s,int n", "", `s+"|"+n+effect()`},
		{"cast", "", "String s,int n", "", `s+"|"+n+(long)n`},
		{"update", "", "String s,int n", "", `s+"|"+n+n++`},
		{"assignment", "", "String s,int n", "", `s+"|"+n+(n=3)`},
		{"conditional", "", "String s,int n", "", `s+"|"+n+(true?1:2)`},
		{"boxed", "", "String s,Integer n", "", `s+"|"+n`},
		{"object", "", "String s,Object n", "", `s+"|"+n`},
		{"CharSequence", "", "String s,CharSequence n", "", `s+"|"+n`},
		{"type-array", "", "String[] s,int n", "", `s+"|"+n`},
		{"declarator-array", "", "String s[],int n", "", `s+"|"+n`},
		{"varargs", "", "String... s", "", `s+"|"+1`},
		{"var-local", "", "String input,int n", `var s=input;`, `s+"|"+n`},
		{"later-local", "String s;", "int n", "", `s+"|"+n`},
		{"sibling-local", "String s;", "int n", `{String s="local";}`, `s+"|"+n`},
		{"nearest-unsupported", "", "String input,int n", `String s=input; {Object s=input;`, `s+"|"+n`},
	}
	for _, row := range negative {
		t.Run("refuse/"+row.name, func(t *testing.T) {
			end := ""
			if row.name == "nearest-unsupported" {
				end = "}"
			}
			source := "public class SpanProbe {" + row.prefix + "static String join(" + row.parameters + "){" + row.before + "return " + row.expression + ";" + end + "}}"
			ctx, node := concatSpanFixture(t, source, row.expression)
			var visit func(*sitter.Node)
			visit = func(part *sitter.Node) {
				if part == nil {
					return
				}
				if part.Type() == "binary_expression" {
					if _, ok := planCanonicalStringConcatSpan(part, ctx.currentFile.Source, ctx); ok {
						t.Fatalf("unsupported maximal expression leaked optimized prefix %q", part.Content(ctx.currentFile.Source))
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(part) {
					visit(child)
				}
			}
			visit(node)
		})
	}
	for _, row := range []struct {
		name, source, expression string
		want                     bool
	}{
		{"source-String", `class String {} class SpanProbe {static String join(String s,int n){return s+"|"+n;}}`, `s+"|"+n`, false},
		{"method-binder", `class SpanProbe {static <String> java.lang.String join(String s,int n){return s+"|"+n;}}`, `s+"|"+n`, false},
		{"qualified-through-binder", `class SpanProbe {static <String> java.lang.String join(java.lang.String s,int n){return s+"|"+n;}}`, `s+"|"+n`, true},
		{"foreign-import", `import foreign.String; class SpanProbe {static String join(String s,int n){return s+"|"+n;}}`, `s+"|"+n`, false},
		{"unknown-wildcard", `import foreign.*; class SpanProbe {static String join(String s,int n){return s+"|"+n;}}`, `s+"|"+n`, false},
		{"unknown-static", `import static foreign.Owner.String; class SpanProbe {static String join(String s,int n){return s+"|"+n;}}`, `s+"|"+n`, false},
		{"source-java-prefix", `class java {static class lang {static class String {}}} class SpanProbe {static java.lang.String join(java.lang.String s,int n){return s+"|"+n;}}`, `s+"|"+n`, false},
	} {
		t.Run("owner/"+row.name, func(t *testing.T) {
			ctx, node := concatSpanFixture(t, row.source, row.expression)
			_, ok := planCanonicalStringConcatSpan(node, ctx.currentFile.Source, ctx)
			if ok != row.want {
				t.Fatalf("owner eligibility=%t want %t", ok, row.want)
			}
		})
	}
	for _, row := range []struct{ name, source, expression string }{
		{"lambda", `class SpanProbe {static String join(String outer,int n){java.util.function.Function<Object,String> fn=(s)->outer+"|"+n+"|"+s;return "";}}`, `outer+"|"+n+"|"+s`},
		{"enhanced-for", `class SpanProbe {static String join(String outer,int n,Object[] values){for(Object s:values){return outer+"|"+n+"|"+s;}return "";}}`, `outer+"|"+n+"|"+s`},
		{"ordinary-for-binding", `class SpanProbe {static String join(String outer,int n){for(String s="x";n>0;n--){return outer+"|"+n+"|"+s;}return "";}}`, `outer+"|"+n+"|"+s`},
		{"catch", `class SpanProbe {static String join(String outer,int n){try{throw new RuntimeException();}catch(Exception s){return outer+"|"+n+"|"+s;}}}`, `outer+"|"+n+"|"+s`},
		{"resource", `class SpanProbe {static String join(String outer,int n) throws Exception {try(java.io.StringReader s=null){return outer+"|"+n+"|"+s;}}}`, `outer+"|"+n+"|"+s`},
	} {
		t.Run("binding/"+row.name, func(t *testing.T) {
			ctx, node := concatSpanFixture(t, row.source, row.expression)
			if _, ok := planCanonicalStringConcatSpan(node, ctx.currentFile.Source, ctx); ok {
				t.Fatal("unsupported lexical binder admitted")
			}
		})
	}
	t.Run("physical-context", func(t *testing.T) {
		source := `class SpanProbe {static String join(String s,int n){return s+"|"+n;}}`
		ctx, node := concatSpanFixture(t, source, `s+"|"+n`)
		twin, _ := concatSpanFixture(t, source, `s+"|"+n`)
		if _, ok := planCanonicalStringConcatSpan(node, []byte(source), twin); ok {
			t.Fatal("byte-identical foreign tree donated authority")
		}
		// A current225 anonymous-origin fact cannot be cleared by donating the
		// exact physical named class/method/root pointers into a copied scope.
		markedOwner := *ctx.currentClass
		markedOwner.MarkAnonymousSourceOrigin()
		marked := ctx.Clone()
		marked.currentClass = &markedOwner
		if _, ok := planCanonicalStringConcatSpan(node, []byte(source), marked); ok {
			t.Fatal("marked anonymous origin borrowed named declaration authority")
		}
		erased := ctx.Clone()
		erased.erasedGenericMethodBody = ctx.localScope
		if _, ok := planCanonicalStringConcatSpan(node, []byte(source), erased); ok {
			t.Fatal("erased ABI context accepted as plain scalar storage")
		}
		missing := ctx.Clone()
		missing.localScope = nil
		if _, ok := planCanonicalStringConcatSpan(node, []byte(source), missing); ok {
			t.Fatal("missing method declaration accepted")
		}
		if _, ok := planCanonicalStringConcatSpan(node, []byte(source+" "), ctx); ok {
			t.Fatal("different source accepted")
		}
	})
	t.Run("hook-and-renaming", func(t *testing.T) {
		for _, name := range []string{"RenamedAlpha", "OtherBeta"} {
			source := "class " + name + ` {static String join(String value,int number){return value+"|"+number;}}`
			generated := renderGoFileFromJava(t, source)
			if !strings.Contains(generated, "ConcatJavaStringSpan") {
				t.Fatal("real ParseExpr hook missing after renaming")
			}
		}
	})
}

// verifyCanonicalNullableConcatReturn checks this test's actual value flow,
// accepting the old direct fold and the equivalent typed staging form.
func verifyCanonicalNullableConcatReturn(generated, ownerName, nullableName string) error {
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if err != nil {
		return err
	}
	// The predeclared nil must retain its language binding. A local or file
	// declaration named nil could otherwise make a typed cast appear null.
	nilShadowed := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.ValueSpec:
			for _, name := range node.Names {
				nilShadowed = nilShadowed || name.Name == "nil"
			}
		case *ast.TypeSpec:
			nilShadowed = nilShadowed || node.Name.Name == "nil"
		case *ast.Field:
			for _, name := range node.Names {
				nilShadowed = nilShadowed || name.Name == "nil"
			}
		case *ast.FuncDecl:
			nilShadowed = nilShadowed || node.Name.Name == "nil"
		case *ast.ImportSpec:
			nilShadowed = nilShadowed || node.Name != nil && node.Name.Name == "nil"
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, left := range node.Lhs {
					if name, ok := left.(*ast.Ident); ok {
						nilShadowed = nilShadowed || name.Name == "nil"
					}
				}
			}
		case *ast.RangeStmt:
			if node.Tok == token.DEFINE {
				for _, left := range []ast.Expr{node.Key, node.Value} {
					if name, ok := left.(*ast.Ident); ok {
						nilShadowed = nilShadowed || name.Name == "nil"
					}
				}
			}
		}
		return true
	})
	if nilShadowed {
		return fmt.Errorf("shadowed predeclared nil")
	}
	canonicalImports := 0
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err == nil && path == "github.com/NickyBoy89/java2go/stdjava" && (imported.Name == nil || imported.Name.Name == "stdjava") {
			canonicalImports++
		}
	}
	if canonicalImports != 1 {
		return fmt.Errorf("missing canonical stdjava import")
	}
	isRef := func(expr ast.Expr, name string) bool {
		star, ok := expr.(*ast.StarExpr)
		if !ok {
			return false
		}
		selector, ok := star.X.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return false
		}
		qualifier, ok := selector.X.(*ast.Ident)
		return ok && qualifier.Name == "stdjava"
	}
	identIs := func(expr ast.Expr, name string) bool {
		ident, ok := expr.(*ast.Ident)
		return ok && ident.Name == name
	}
	callIs := func(expr ast.Expr, name string, count int) (*ast.CallExpr, bool) {
		call, ok := expr.(*ast.CallExpr)
		if !ok || len(call.Args) != count || call.Ellipsis.IsValid() {
			return nil, false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name || !identIs(selector.X, "stdjava") {
			return nil, false
		}
		return call, true
	}
	var owner *ast.FuncDecl
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == ownerName {
			if owner != nil || function.Recv != nil {
				return fmt.Errorf("ambiguous owner")
			}
			owner = function
		}
	}
	if owner == nil || owner.Body == nil || owner.Type.Params == nil || len(owner.Type.Params.List) != 1 || owner.Type.Results == nil || len(owner.Type.Results.List) != 1 || owner.Type.TypeParams != nil || !isRef(owner.Type.Results.List[0].Type, "JavaString") {
		return fmt.Errorf("missing typed owner")
	}
	parameter := owner.Type.Params.List[0]
	if len(parameter.Names) != 1 || !isRef(parameter.Type, "Execution") {
		return fmt.Errorf("missing typed execution parameter")
	}
	execution := parameter.Names[0].Name
	if execution == "stdjava" || execution == nullableName || execution == "result" || execution == "_" {
		return fmt.Errorf("execution binding collision")
	}
	reserved := func(name string) bool { return name == execution || name == "stdjava" }
	// Parentheses preserve an assignment target's binding and addressability.
	bindingIdent := func(expr ast.Expr) (*ast.Ident, bool) {
		for {
			parenthesized, ok := expr.(*ast.ParenExpr)
			if !ok {
				break
			}
			expr = parenthesized.X
		}
		ident, ok := expr.(*ast.Ident)
		return ident, ok
	}
	bindingChanged := false
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.ValueSpec:
			for _, name := range node.Names {
				bindingChanged = bindingChanged || reserved(name.Name)
			}
		case *ast.TypeSpec:
			bindingChanged = bindingChanged || reserved(node.Name.Name)
		case *ast.Field:
			for _, name := range node.Names {
				bindingChanged = bindingChanged || reserved(name.Name)
			}
		case *ast.AssignStmt:
			for _, left := range node.Lhs {
				if name, ok := bindingIdent(left); ok {
					bindingChanged = bindingChanged || reserved(name.Name)
				}
			}
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				value := node.X
				for {
					parenthesized, ok := value.(*ast.ParenExpr)
					if !ok {
						break
					}
					value = parenthesized.X
				}
				if name, ok := value.(*ast.Ident); ok && name.Name == execution {
					bindingChanged = true
				}
			}
		case *ast.IncDecStmt:
			if name, ok := bindingIdent(node.X); ok {
				bindingChanged = bindingChanged || reserved(name.Name)
			}
		case *ast.RangeStmt:
			for _, left := range []ast.Expr{node.Key, node.Value} {
				if name, ok := bindingIdent(left); ok {
					bindingChanged = bindingChanged || reserved(name.Name)
				}
			}
		}
		return true
	})
	if bindingChanged || len(owner.Body.List) < 2 {
		return fmt.Errorf("execution or runtime alias rebound")
	}
	ownerReturns, nullableDeclarations := 0, 0
	ast.Inspect(owner.Body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		if _, ok := node.(*ast.ReturnStmt); ok {
			ownerReturns++
		}
		if declaration, ok := node.(*ast.ValueSpec); ok {
			for _, name := range declaration.Names {
				if name.Name == nullableName {
					nullableDeclarations++
				}
			}
		}
		return true
	})
	terminal, ok := owner.Body.List[len(owner.Body.List)-1].(*ast.ReturnStmt)
	if !ok || ownerReturns != 1 || nullableDeclarations != 1 || len(terminal.Results) != 1 {
		return fmt.Errorf("missing sole owner terminal return")
	}
	declaration, ok := owner.Body.List[len(owner.Body.List)-2].(*ast.DeclStmt)
	if !ok {
		return fmt.Errorf("missing nullable declaration")
	}
	general, ok := declaration.Decl.(*ast.GenDecl)
	if !ok || general.Tok != token.VAR || len(general.Specs) != 1 {
		return fmt.Errorf("missing nullable scalar var")
	}
	nullable, ok := general.Specs[0].(*ast.ValueSpec)
	if !ok || len(nullable.Names) != 1 || nullable.Names[0].Name != nullableName || !isRef(nullable.Type, "JavaString") || len(nullable.Values) != 1 {
		return fmt.Errorf("wrong nullable declaration")
	}
	nullValue := identIs(nullable.Values[0], "nil")
	if cast, ok := nullable.Values[0].(*ast.CallExpr); ok && len(cast.Args) == 1 && !cast.Ellipsis.IsValid() && identIs(cast.Args[0], "nil") {
		if parenthesized, ok := cast.Fun.(*ast.ParenExpr); ok && isRef(parenthesized.X, "JavaString") {
			nullValue = true
		}
	}
	if !nullValue {
		return fmt.Errorf("nullable not initialized null")
	}
	expression := terminal.Results[0]
	reads := 0
	ast.Inspect(expression, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name == nullableName {
			reads++
		}
		return true
	})
	if reads != 1 {
		return fmt.Errorf("nullable return reads=%d, want1", reads)
	}
	leafIs := func(expr ast.Expr, index int) bool {
		if index != 1 {
			name := "result"
			if index == 2 {
				name = nullableName
			}
			return identIs(expr, name)
		}
		call, ok := callIs(expr, "JavaStringLiteralUTF16", 1)
		if !ok {
			return false
		}
		literal, ok := call.Args[0].(*ast.CompositeLit)
		if !ok || len(literal.Elts) != 1 {
			return false
		}
		array, ok := literal.Type.(*ast.ArrayType)
		if !ok || array.Len != nil || !identIs(array.Elt, "uint16") {
			return false
		}
		unit, ok := literal.Elts[0].(*ast.BasicLit)
		if !ok || unit.Kind != token.INT {
			return false
		}
		value, err := strconv.ParseUint(unit.Value, 0, 16)
		return err == nil && value == 58
	}
	textIs := func(expr ast.Expr, value string) bool {
		call, ok := callIs(expr, "JavaStringTextOperandExecution", 2)
		return ok && identIs(call.Args[0], execution) && identIs(call.Args[1], value)
	}
	if call, ok := expression.(*ast.CallExpr); ok && len(call.Args) == 0 && !call.Ellipsis.IsValid() {
		if function, ok := call.Fun.(*ast.FuncLit); ok {
			if function.Type.Params == nil || len(function.Type.Params.List) != 0 || function.Type.Results == nil || len(function.Type.Results.List) != 1 || function.Type.TypeParams != nil || !isRef(function.Type.Results.List[0].Type, "JavaString") || len(function.Body.List) != 7 {
				return fmt.Errorf("wrong typed staging function")
			}
			seen := map[string]bool{nullableName: true, "result": true, execution: true, "stdjava": true, "_": true}
			var texts []string
			for index := 0; index < 3; index++ {
				var pair []*ast.ValueSpec
				for offset := 0; offset < 2; offset++ {
					statement, ok := function.Body.List[2*index+offset].(*ast.DeclStmt)
					if !ok {
						return fmt.Errorf("nondeclaration in staging")
					}
					declaration, ok := statement.Decl.(*ast.GenDecl)
					if !ok || declaration.Tok != token.VAR || len(declaration.Specs) != 1 {
						return fmt.Errorf("nonvar staging")
					}
					value, ok := declaration.Specs[0].(*ast.ValueSpec)
					if !ok || len(value.Names) != 1 || len(value.Values) != 1 || !isRef(value.Type, "JavaString") || seen[value.Names[0].Name] {
						return fmt.Errorf("untyped, reused or shadowing stage")
					}
					seen[value.Names[0].Name] = true
					pair = append(pair, value)
				}
				if !leafIs(pair[0].Values[0], index) || !textIs(pair[1].Values[0], pair[0].Names[0].Name) {
					return fmt.Errorf("wrong ordered value or same-execution conversion")
				}
				texts = append(texts, pair[1].Names[0].Name)
			}
			result, ok := function.Body.List[6].(*ast.ReturnStmt)
			if !ok || len(result.Results) != 1 {
				return fmt.Errorf("missing staged return")
			}
			join, ok := callIs(result.Results[0], "ConcatJavaStringSpan", 3)
			if !ok {
				return fmt.Errorf("missing exact span join")
			}
			for index, argument := range join.Args {
				if !identIs(argument, texts[index]) {
					return fmt.Errorf("reordered or unbound converted operand")
				}
			}
			return nil
		}
	}
	var operands []ast.Expr
	var join func(ast.Expr) bool
	operand := func(expr ast.Expr) bool {
		call, ok := callIs(expr, "JavaStringTextOperandExecution", 2)
		if !ok || !identIs(call.Args[0], execution) {
			return false
		}
		if _, nested := callIs(call.Args[1], "ConcatJavaStrings", 2); nested {
			return join(call.Args[1])
		}
		operands = append(operands, call.Args[1])
		return true
	}
	join = func(expr ast.Expr) bool {
		call, ok := callIs(expr, "ConcatJavaStrings", 2)
		return ok && operand(call.Args[0]) && operand(call.Args[1])
	}
	if !join(expression) || len(operands) != 3 {
		return fmt.Errorf("unsupported direct fold")
	}
	for index, leaf := range operands {
		if !leafIs(leaf, index) {
			return fmt.Errorf("wrong direct operand order")
		}
	}
	return nil
}

func checkCanonicalNullableConcatAssertionRows(t *testing.T) {
	t.Helper()
	const header = `package main
import stdjava "github.com/NickyBoy89/java2go/stdjava"
func RunJava2goExecution(exec *stdjava.Execution) *stdjava.JavaString {
 result := stdjava.JavaStringLiteralUTF16([]uint16{118})
 func(dst **stdjava.JavaString) func(rhs int32) *stdjava.JavaString {
  old := *dst
  return func(rhs int32) *stdjava.JavaString {
   value := stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(exec, old), stdjava.JavaStringValueOfInt(rhs))
   *dst = value
   return value
  }
 }(&result)(7)
 var nullable *stdjava.JavaString = (*stdjava.JavaString)(nil)
 return `
	const direct = `stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(exec, stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(exec, result), stdjava.JavaStringTextOperandExecution(exec, stdjava.JavaStringLiteralUTF16([]uint16{58})))), stdjava.JavaStringTextOperandExecution(exec, nullable))`
	const staged = `func() *stdjava.JavaString {
 var v0 *stdjava.JavaString = result
 var t0 *stdjava.JavaString = stdjava.JavaStringTextOperandExecution(exec, v0)
 var v1 *stdjava.JavaString = stdjava.JavaStringLiteralUTF16([]uint16{58})
 var t1 *stdjava.JavaString = stdjava.JavaStringTextOperandExecution(exec, v1)
 var v2 *stdjava.JavaString = nullable
 var t2 *stdjava.JavaString = stdjava.JavaStringTextOperandExecution(exec, v2)
 return stdjava.ConcatJavaStringSpan(t0, t1, t2)
}()`
	old := header + direct + "\n}"
	current := header + staged + "\n}"
	mutate := func(source, before, after string) string {
		t.Helper()
		if strings.Count(source, before) != 1 {
			t.Fatalf("mutation target not unique: %q", before)
		}
		return strings.Replace(source, before, after, 1)
	}
	// Rename Go identifiers, never token substrings such as t1 in uint16 or
	// occurrences in comments and string literals.
	alphaSource := mutate(current, " var nullable", " // exec v0 t0 v1 t1 v2 t2 uint16\n _ = \"exec v0 t0 v1 t1 v2 t2 uint16\"\n var nullable")
	alphaSet := token.NewFileSet()
	alphaTree, err := parser.ParseFile(alphaSet, "alpha.go", alphaSource, parser.ParseComments)
	if err != nil {
		t.Fatalf("alpha fixture parse: %v", err)
	}
	renames := map[string]string{"exec": "sameScope", "v0": "leftValue", "t0": "leftText", "v1": "middleValue", "t1": "middleText", "v2": "lastValue", "t2": "lastText"}
	tokens := func(tree *ast.File) (names map[string]int, literals, comments []string) {
		names = make(map[string]int)
		ast.Inspect(tree, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.Ident:
				names[node.Name]++
			case *ast.BasicLit:
				literals = append(literals, node.Value)
			}
			return true
		})
		for _, group := range tree.Comments {
			for _, comment := range group.List {
				comments = append(comments, comment.Text)
			}
		}
		return
	}
	beforeNames, beforeLiterals, beforeComments := tokens(alphaTree)
	ast.Inspect(alphaTree, func(node ast.Node) bool {
		if name, ok := node.(*ast.Ident); ok {
			if replacement, found := renames[name.Name]; found {
				name.Name = replacement
			}
		}
		return true
	})
	var printedAlpha strings.Builder
	if err := printer.Fprint(&printedAlpha, alphaSet, alphaTree); err != nil {
		t.Fatalf("alpha fixture print: %v", err)
	}
	alpha := printedAlpha.String()
	reparsedAlpha, err := parser.ParseFile(token.NewFileSet(), "renamed-alpha.go", alpha, parser.ParseComments)
	if err != nil {
		t.Fatalf("renamed alpha fixture parse: %v", err)
	}
	afterNames, afterLiterals, afterComments := tokens(reparsedAlpha)
	if !reflect.DeepEqual(beforeLiterals, afterLiterals) || !reflect.DeepEqual(beforeComments, afterComments) {
		t.Fatal("alpha rename changed literal or comment tokens")
	}
	expectedNames := make(map[string]int)
	for name, count := range beforeNames {
		if replacement, renamed := renames[name]; renamed {
			expectedNames[replacement] = count
			if count == 0 || beforeNames[replacement] != 0 || afterNames[name] != 0 || afterNames[replacement] != count {
				t.Fatalf("wrong alpha binding rename %s", name)
			}
		} else {
			expectedNames[name] = count
			if afterNames[name] != count {
				t.Fatalf("alpha rename changed unrelated identifier %s", name)
			}
		}
	}
	if !reflect.DeepEqual(expectedNames, afterNames) {
		t.Fatal("alpha rename changed unexpected identifier tokens")
	}
	for name := range renames {
		if beforeNames[name] == 0 {
			t.Fatalf("alpha fixture lacks intended identifier %s", name)
		}
	}
	rows := []struct {
		name, source string
		accept       bool
	}{
		{"old-direct-with-compound-IIFE-returns", old, true},
		{"new-staged-with-compound-IIFE-returns", current, true},
		{"alpha-renamed-execution-and-stages", alpha, true},
		{"duplicate-nullable-read", mutate(current, "var v2 *stdjava.JavaString = nullable", "var v2 *stdjava.JavaString = stdjava.ConcatJavaStrings(nullable, nullable)"), false},
		{"missing-nullable-read", mutate(current, "var v2 *stdjava.JavaString = nullable", "var v2 *stdjava.JavaString = result"), false},
		{"wrong-nullable-initializer", mutate(current, "var nullable *stdjava.JavaString = (*stdjava.JavaString)(nil)", "var nullable *stdjava.JavaString = result"), false},
		{"wrong-nullable-declaration-type", mutate(current, "var nullable *stdjava.JavaString", "var nullable any"), false},
		{"wrong-stage-type", mutate(current, "var v2 *stdjava.JavaString", "var v2 any"), false},
		{"wrong-source-value", mutate(current, "var v0 *stdjava.JavaString = result", "var v0 *stdjava.JavaString = nullable"), false},
		{"fresh-execution", mutate(current, "JavaStringTextOperandExecution(exec, v2)", "JavaStringTextOperandExecution(stdjava.NewExecution(), v2)"), false},
		{"other-execution", mutate(current, "JavaStringTextOperandExecution(exec, v2)", "JavaStringTextOperandExecution(other, v2)"), false},
		{"shadowed-execution", mutate(current, "var v2 *stdjava.JavaString", "var exec *stdjava.JavaString"), false},
		{"reassigned-execution", mutate(current, " var nullable", " exec = stdjava.NewExecution()\n var nullable"), false},
		{"shadowed-runtime-alias", mutate(current, " var nullable", " stdjava := 0\n var nullable"), false},
		{"shadowed-nullable", mutate(current, "var v2 *stdjava.JavaString", "var nullable *stdjava.JavaString"), false},
		{"missing-conversion", mutate(current, "var t2 *stdjava.JavaString = stdjava.JavaStringTextOperandExecution(exec, v2)", "var t2 *stdjava.JavaString = v2"), false},
		{"wrong-conversion-helper", mutate(current, "JavaStringTextOperandExecution(exec, v2)", "JavaStringValueOfExecution(exec, v2)"), false},
		{"conversion-before-value", mutate(current, "var v2 *stdjava.JavaString = nullable\n var t2 *stdjava.JavaString = stdjava.JavaStringTextOperandExecution(exec, v2)", "var t2 *stdjava.JavaString = stdjava.JavaStringTextOperandExecution(exec, v2)\n var v2 *stdjava.JavaString = nullable"), false},
		{"conversion-of-sibling-stage", mutate(current, "JavaStringTextOperandExecution(exec, v2)", "JavaStringTextOperandExecution(exec, v1)"), false},
		{"reused-text-name", mutate(current, "var t2 *stdjava.JavaString", "var t1 *stdjava.JavaString"), false},
		{"reordered-span-arguments", mutate(current, "ConcatJavaStringSpan(t0, t1, t2)", "ConcatJavaStringSpan(t0, t2, t1)"), false},
		{"omitted-span-argument", mutate(current, "ConcatJavaStringSpan(t0, t1, t2)", "ConcatJavaStringSpan(t0, t1)"), false},
		{"extra-effect-statement", mutate(current, " return stdjava.ConcatJavaStringSpan", " touch()\n return stdjava.ConcatJavaStringSpan"), false},
		{"direct-missing-nullable-conversion", mutate(old, "stdjava.JavaStringTextOperandExecution(exec, nullable)", "nullable"), false},
		{"direct-reordered-operands", header + `stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(exec, stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(exec, nullable), stdjava.JavaStringTextOperandExecution(exec, stdjava.JavaStringLiteralUTF16([]uint16{58})))), stdjava.JavaStringTextOperandExecution(exec, result))` + "\n}", false},
		{"wrong-colon-unit", mutate(current, "[]uint16{58}", "[]uint16{59}"), false},
		{"unbound-stage-alias", mutate(current, "ConcatJavaStringSpan(t0, t1, t2)", "ConcatJavaStringSpan(t0, t1, missing)"), false},
		{"shadowed-predeclared-nil-valid-Go", mutate(current, " var nullable", " nil := stdjava.JavaStringLiteralUTF16([]uint16{110,117,108,108})\n var nullable"), false},
		{"indirect-execution-replacement-valid-Go", mutate(current, " var nullable", " func(p **stdjava.Execution) { *p = stdjava.NewExecution() }(&(exec))\n var nullable"), false},
		{"parenthesized-execution-rebind-valid-Go", mutate(current, " var nullable", " ((exec)) = stdjava.NewExecution()\n var nullable"), false},
		{"wrong-owner", mutate(current, "func RunJava2goExecution", "func OtherOwner"), false},
	}
	if len(rows) != 31 {
		t.Fatal("expected exactly3 positive and28 refusal rows")
	}
	for _, row := range rows {
		err := verifyCanonicalNullableConcatReturn(row.source, "RunJava2goExecution", "nullable")
		if (err == nil) != row.accept {
			t.Fatalf("%s: accepted=%t want=%t error=%v", row.name, err == nil, row.accept, err)
		}
	}
}
