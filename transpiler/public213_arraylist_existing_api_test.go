package transpiler

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Pure existing-API reducers for the public baseline. No proposed helper or
// runtime API is referenced. Original application inputs and oracles stay frozen.
func public213ArrayListText(value ast.Node) string {
	if value == nil {
		return "<nil>"
	}
	var out bytes.Buffer
	if err := printer.Fprint(&out, token.NewFileSet(), value); err != nil {
		return "<print-error>"
	}
	return out.String()
}
func public213ArrayListOccurrence(t *testing.T, declarations, result, expression string) (*ParseHelper, Ctx, *sitter.Node) {
	t.Helper()
	source := `import java.util.ArrayList;import java.util.List;import java.util.function.Supplier;import java.util.stream.Stream;import java.util.stream.Collectors;` + declarations + `class Use{static ` + result + ` run(String first,String last){return ` + expression + `;}static ` + result + ` other(String first,String last){return null;}}`
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	if method == nil || method.DeclarationNode == nil {
		t.Fatal("actual attached run method missing")
	}
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	ctx.executionContextName = executionParameterName(method.DeclarationNode, h.File.Source, ctx)
	statement := findNode(method.DeclarationNode.ChildByFieldName("body"), "return_statement")
	if statement == nil || statement.NamedChildCount() != 1 {
		t.Fatal("actual attached return expression missing")
	}
	return h, ctx, statement.NamedChild(0)
}
func public213ArrayListFactoryPremise(t *testing.T, h *ParseHelper, ctx Ctx, collect *sitter.Node, want string) ast.Expr {
	t.Helper()
	receiver := collect.ChildByFieldName("object")
	name, known := intrinsicStaticClassName(receiver.ChildByFieldName("object"), ctx, h.File.Source)
	if !known || name != "Stream" || receiver.ChildByFieldName("name").Content(h.File.Source) != "of" {
		t.Fatal("actual canonical static Stream factory not selected")
	}
	elements := receiverElementJavaTypes(receiver, ctx, h.File.Source)
	if len(elements) != 1 || public213ArrayListText(javaTypeStringToGoTypeExpr(elements[0], inScopeTypeParameters(ctx), ctx)) != want {
		t.Fatal("receiver physical element failed before collector query")
	}
	parsed := ParseExpr(receiver, h.File.Source, ctx)
	if !strings.Contains(public213ArrayListText(parsed), "NewStream["+want+"]") {
		t.Fatal("existing Stream factory emitter failed before collector query")
	}
	collector := invocationArgumentNode(collect, 0)
	factory, canonical := collectorFactoryName(collector, h.File.Source)
	if !canonical || factory != "toCollection" || invocationArgumentCount(collector) != 1 {
		t.Fatal("actual collector factory failed before capability query")
	}
	return parsed
}
func TestPublic213ArrayListLegacyToListControl(t *testing.T) {
	h, ctx, node := public213ArrayListOccurrence(t, "", "List<String>", `Stream.<String>of(first,null,last).collect(Collectors.toList())`)
	receiver := ParseExpr(node.ChildByFieldName("object"), h.File.Source, ctx)
	value := lowerCollectCall(receiver, node, ctx, h.File.Source)
	got := public213ArrayListText(value)
	if !strings.Contains(got, "NewStream[*stdjava.JavaString]") || !strings.HasSuffix(got, ".ToList()") || strings.Contains(got, "StreamCollectPlanExecution") || strings.Contains(got, "StreamFromValues[") {
		t.Fatalf("unchanged public legacy toList control failed: %s", got)
	}
}
func TestPublic213ArrayListGeneralTreeMapGateControl(t *testing.T) {
	h := setupParseHelper(t, `final class Held<E> extends java.util.TreeMap<E,E>{E seen;E echo(E value){return value;}}`)
	owner := h.Ctx.currentFile.FindClassScope("Held")
	if owner == nil || owner.Superclass != "java.util.TreeMap<E,E>" {
		t.Fatal("actual generic TreeMap parent premise missing")
	}
	if plan, err := planGenericFamily(owner, h.Ctx); plan != nil || err == nil {
		t.Fatal("public general family gate unexpectedly admits native TreeMap layout")
	}
}
func TestPublic213ArrayListWrittenObjectFactoryPhysicalType(t *testing.T) {
	h, ctx, node := public213ArrayListOccurrence(t, "", "Stream<Object>", `Stream.<Object>of(first,null,last)`)
	arguments := node.ChildByFieldName("type_arguments")
	selected, known := intrinsicStaticClassName(node.ChildByFieldName("object"), ctx, h.File.Source)
	if arguments == nil || arguments.NamedChildCount() != 1 || arguments.NamedChild(0).Content(h.File.Source) != "Object" || !known || selected != "Stream" {
		t.Fatal("actual written Object/canonical Stream premise missing")
	}
	value := ParseExpr(node, h.File.Source, ctx)
	call, ok := value.(*ast.CallExpr)
	if !ok {
		t.Fatal("supported factory did not render a call")
	}
	typed, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		t.Fatal("supported factory did not render its actual physical type")
	}
	physical := public213ArrayListText(typed.Index)
	elements := receiverElementJavaTypes(node, ctx, h.File.Source)
	if len(elements) != 1 {
		t.Fatal("supported factory result origin missing")
	}
	element := public213ArrayListText(javaTypeStringToGoTypeExpr(elements[0], inScopeTypeParameters(ctx), ctx))
	t.Logf("PUBLIC213_OBJECT_PHYSICAL=%s ELEMENT=%s", physical, element)
	if physical != "any" || element != "any" {
		t.Error("PUBLIC213_MISSING_WRITTEN_OBJECT_FACTORY_TYPE")
	}
}
func TestPublic213ArrayListSixSuppliedCollectionForms(t *testing.T) {
	for _, row := range []struct{ name, declaration, result, factory, stream, physical string }{
		{"native_constructor", "", "ArrayList<String>", "ArrayList<String>::new", `Stream.<String>of(first,null,last)`, "*stdjava.JavaString"},
		{"native_lambda", "", "ArrayList<String>", "()->new ArrayList<String>()", `Stream.<String>of(first,null,last)`, "*stdjava.JavaString"},
		{"declared_supplier", `class Factories{static Supplier<ArrayList<String>> factory=ArrayList<String>::new;}`, "ArrayList<String>", "Factories.factory", `Stream.<String>of(first,null,last)`, "*stdjava.JavaString"},
		{"generic_source", `class Values<E> extends ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`, "Values<String>", "Values<String>::new", `Stream.<String>of(first,null,last)`, "*stdjava.JavaString"},
		{"String_source", `final class Words extends ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}`, "Words", "Words::new", `Stream.<String>of(first,null,last)`, "*stdjava.JavaString"},
		// Null operands isolate supplier selection from the independently tested
		// written-Object-vs-String-LUB bug; the original author probe stays unchanged.
		{"Object_source", `class Plain extends ArrayList<Object>{}`, "Plain", "()->new Plain()", `Stream.<Object>of(null,null,null)`, "any"},
	} {
		h, ctx, node := public213ArrayListOccurrence(t, row.declaration, row.result, row.stream+`.collect(Collectors.toCollection(`+row.factory+`))`)
		receiver := public213ArrayListFactoryPremise(t, h, ctx, node, row.physical)
		t.Logf("PUBLIC213_RECEIVER_FACTORY_PROVEN case=%s", row.name)
		value := lowerCollectCall(receiver, node, ctx, h.File.Source)
		elements := receiverElementJavaTypes(node.ChildByFieldName("object"), ctx, h.File.Source)
		result, known := inferCollectorJavaType(invocationArgumentNode(node, 0), elements[0], ctx, h.File.Source)
		if value == nil || !known || result == "" {
			t.Errorf("PUBLIC213_MISSING_TO_COLLECTION case=%s nil=%t inferred=%t", row.name, value == nil, known)
		}
	}
}
func public213ArrayListInitializedSourceAndRawConsumerCase(t *testing.T, row struct{ name, args string }) {
	expression := `Stream.<String>of(` + row.args + `).collect(Collectors.toCollection(ArrayList<String>::new))`
	h, ctx, node := public213ArrayListOccurrence(t, "", "ArrayList<String>", expression)
	receiver := public213ArrayListFactoryPremise(t, h, ctx, node, "*stdjava.JavaString")
	t.Logf("PUBLIC213_ATTACHED_NATIVE_RETURN_PROVEN case=%s", row.name)
	raw := lowerCollectCall(receiver, node, ctx, h.File.Source)
	if raw == nil {
		t.Errorf("PUBLIC213_MISSING_INITIALIZED_RAW_C_PROTOCOL case=%s", row.name)
		return
	}
	rendered := public213ArrayListText(raw)
	if strings.Count(rendered, "StreamFromValues[*stdjava.JavaString](") != 1 || strings.Contains(rendered, "NewStream[") || !strings.Contains(rendered, "StreamCollectPlanExecution("+ctx.executionContextName+",") || strings.Contains(rendered, "NewExecution(") {
		t.Fatal("actual public initialized source/current Execution protocol incorrect")
	}
	projection := "ObjectView[*stdjava.List[*stdjava.JavaString]]"
	positive := func() {
		t.Helper()
		converted := public213ArrayListText(coerceArgumentToExpectedType(raw, node, "ArrayList<String>", ctx, h.File.Source))
		if !strings.Contains(converted, projection) || strings.Count(converted, "StreamCollectPlanExecution(") != 1 {
			t.Fatal("actual native C return lost one original-object consumer projection")
		}
	}
	positive()
	registered := symbol.GlobalScope
	foreign, other, foreignNode := public213ArrayListOccurrence(t, "", "ArrayList<String>", expression)
	foreignRaw := lowerCollectCall(ParseExpr(foreignNode.ChildByFieldName("object"), foreign.File.Source, other), foreignNode, other, foreign.File.Source)
	if foreignRaw == nil || !strings.Contains(public213ArrayListText(coerceArgumentToExpectedType(foreignRaw, foreignNode, "ArrayList<String>", other, foreign.File.Source)), projection) {
		t.Fatal("foreign graph did not establish its own positive premise")
	}
	symbol.GlobalScope = registered
	positive()
	wrong := ctx.Clone()
	wrong.localScope = ctx.currentClass.FindMethodByName("other", nil)
	for _, bad := range []struct {
		ctx    Ctx
		source []byte
	}{{other, h.File.Source}, {wrong, h.File.Source}, {ctx, append([]byte(nil), h.File.Source...)}} {
		if strings.Contains(public213ArrayListText(coerceArgumentToExpectedType(raw, node, "ArrayList<String>", bad.ctx, bad.source)), projection) {
			t.Fatal("foreign executable/copied source donated original C consumer authority")
		}
	}
	symbol.GlobalScope = registered
	positive()
	source := `import java.util.ArrayList;import java.util.stream.Stream;import java.util.stream.Collectors;class Use{static ArrayList<String> run(String first,String last){return ` + expression + `;}}`
	body := renderGoFileFromJava(t, source)
	implementation, err := public213ArrayListExecutionFlow(body)
	if err != nil {
		t.Fatalf("actual emitted execution flow: %v", err)
	}
	if strings.Count(body, "StreamFromValues[*stdjava.JavaString](") != 1 || strings.Contains(body, "NewStream[") || !strings.Contains(body, "StreamCollectPlanExecution(__java2goExecution") || strings.Contains(implementation, "NewExecution(") || !strings.Contains(body, projection) {
		t.Fatal("actual emitted public body lost initialized source/currentE/original C projection")
	}
	for _, mutation := range []string{"fresh_execution", "wrong_execution", "facade_reuses_execution", "facade_wrong_argument"} {
		t.Run(mutation, func(t *testing.T) {
			mutant, err := public213ArrayListExecutionFlowMutant(body, mutation)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := public213ArrayListExecutionFlow(mutant); err == nil {
				t.Fatalf("execution-flow assertion accepted %s", mutation)
			}
		})
	}
}
func TestPublic213ArrayListInitializedSourceAndRawConsumer(t *testing.T) {
	public213ArrayListInitializedSourceAndRawConsumerCase(t, struct{ name, args string }{"zero", ""})
}
func TestPublic213ArrayListInitializedSourceAndRawConsumerMulti(t *testing.T) {
	public213ArrayListInitializedSourceAndRawConsumerCase(t, struct{ name, args string }{"multi", "first,null,last"})
}
func TestPublic213ArrayListEmptyWrittenFactoryPhysicalType(t *testing.T) {
	h, ctx, node := public213ArrayListOccurrence(t, "", "Stream<String>", `Stream.<String>of()`)
	arguments := node.ChildByFieldName("type_arguments")
	selected, known := intrinsicStaticClassName(node.ChildByFieldName("object"), ctx, h.File.Source)
	if arguments == nil || arguments.NamedChildCount() != 1 || arguments.NamedChild(0).Content(h.File.Source) != "String" || !known || selected != "Stream" {
		t.Fatal("actual written empty String/canonical Stream premise missing")
	}
	value := ParseExpr(node, h.File.Source, ctx)
	call, ok := value.(*ast.CallExpr)
	if !ok {
		t.Fatal("supported factory did not render a call")
	}
	typed, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		t.Fatal("supported factory did not render its actual physical type")
	}
	physical := public213ArrayListText(typed.Index)
	elements := receiverElementJavaTypes(node, ctx, h.File.Source)
	if len(elements) != 1 {
		t.Fatal("supported factory result origin missing")
	}
	element := public213ArrayListText(javaTypeStringToGoTypeExpr(elements[0], inScopeTypeParameters(ctx), ctx))
	t.Logf("PUBLIC213_EMPTY_PHYSICAL=%s ELEMENT=%s", physical, element)
	if physical != "*stdjava.JavaString" || element != "*stdjava.JavaString" {
		t.Error("PUBLIC213_MISSING_EMPTY_WRITTEN_STRING_FACTORY_TYPE")
	}
}

// public213ArrayListExecutionFlow checks the Java implementation separately
// from the public Go ABI facade, whose job is to start a logical execution.
func public213ArrayListExecutionFlow(body string) (string, error) {
	file, err := goparser.ParseFile(token.NewFileSet(), "emitted.go", body, 0)
	if err != nil {
		return "", err
	}
	functions := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
			if functions[function.Name.Name] != nil {
				return "", fmt.Errorf("duplicate emitted function %s", function.Name.Name)
			}
			functions[function.Name.Name] = function
		}
	}
	for _, name := range []string{"newUse", "run"} {
		facade, implementation := functions[name], functions[name+"Java2goExecution"]
		if facade == nil || implementation == nil || facade.Body == nil || implementation.Body == nil || len(facade.Body.List) != 1 || implementation.Type.Params == nil || len(implementation.Type.Params.List) == 0 {
			return "", fmt.Errorf("missing execution facade/implementation for %s", name)
		}
		execution := implementation.Type.Params.List[0]
		if len(execution.Names) != 1 || execution.Names[0].Name != "__java2goExecution" || public213ArrayListText(execution.Type) != "*stdjava.Execution" || public213ArrayListText(&ast.FuncType{Params: &ast.FieldList{List: implementation.Type.Params.List[1:]}, Results: implementation.Type.Results}) != public213ArrayListText(facade.Type) {
			return "", fmt.Errorf("implementation execution/argument contract for %s", name)
		}
		returned, ok := facade.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 {
			return "", fmt.Errorf("facade must return implementation call for %s", name)
		}
		call, ok := returned.Results[0].(*ast.CallExpr)
		if !ok || public213ArrayListText(call.Fun) != implementation.Name.Name || len(call.Args) == 0 || public213ArrayListText(call.Args[0]) != "stdjava.NewExecution()" {
			return "", fmt.Errorf("facade must create and forward entry execution for %s", name)
		}
		arguments := []string{}
		for _, field := range facade.Type.Params.List {
			for _, parameter := range field.Names {
				arguments = append(arguments, parameter.Name)
			}
		}
		if len(call.Args) != len(arguments)+1 {
			return "", fmt.Errorf("facade argument count for %s", name)
		}
		for i, argument := range arguments {
			if public213ArrayListText(call.Args[i+1]) != argument {
				return "", fmt.Errorf("facade must forward argument %s", argument)
			}
		}
		fresh := false
		ast.Inspect(implementation.Body, func(node ast.Node) bool {
			if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "NewExecution" {
				fresh = true
			}
			return true
		})
		if fresh {
			return "", fmt.Errorf("fresh execution inside Java implementation %s", name)
		}
	}
	implementation := functions["runJava2goExecution"]
	if len(implementation.Body.List) != 1 {
		return "", fmt.Errorf("collection implementation must return without rebinding caller execution")
	}
	returned, ok := implementation.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return "", fmt.Errorf("collection implementation return missing")
	}
	projection, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || public213ArrayListText(projection.Fun) != "stdjava.ObjectView[*stdjava.List[*stdjava.JavaString]]" || len(projection.Args) != 2 {
		return "", fmt.Errorf("original C must be the returned ObjectView consumer")
	}
	collect, ok := projection.Args[0].(*ast.CallExpr)
	if !ok || public213ArrayListText(collect.Fun) != "stdjava.StreamCollectPlanExecution" || len(collect.Args) != 3 {
		return "", fmt.Errorf("original C must come directly from initialized collect")
	}
	execution, ok := collect.Args[0].(*ast.Ident)
	if !ok || execution.Name != implementation.Type.Params.List[0].Names[0].Name {
		return "", fmt.Errorf("collect must forward the same caller execution")
	}
	source, ok := collect.Args[1].(*ast.CallExpr)
	if !ok || public213ArrayListText(source.Fun) != "stdjava.StreamFromValues[*stdjava.JavaString]" {
		return "", fmt.Errorf("collect must consume the initialized source directly")
	}
	collector, ok := collect.Args[2].(*ast.CallExpr)
	if !ok || public213ArrayListText(collector.Fun) != "stdjava.CollectorToCollectionPlan[*stdjava.JavaString]" {
		return "", fmt.Errorf("collect must retain its supplied collection plan")
	}
	return public213ArrayListText(implementation.Body), nil
}

// These mutations target the actual emitted AST; they are never compiled or run
// as generated programs. Each must be rejected by the positive flow assertion.
func public213ArrayListExecutionFlowMutant(body, mutation string) (string, error) {
	file, err := goparser.ParseFile(token.NewFileSet(), "emitted.go", body, 0)
	if err != nil {
		return "", err
	}
	changed := false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		if function.Name.Name == "runJava2goExecution" && (mutation == "fresh_execution" || mutation == "wrong_execution") {
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok && public213ArrayListText(call.Fun) == "stdjava.StreamCollectPlanExecution" && len(call.Args) == 3 {
					if mutation == "fresh_execution" {
						call.Args[0] = &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("stdjava"), Sel: ast.NewIdent("NewExecution")}}
					} else {
						call.Args[0] = ast.NewIdent("nil")
					}
					changed = true
				}
				return true
			})
		}
		if function.Name.Name == "run" && (mutation == "facade_reuses_execution" || mutation == "facade_wrong_argument") {
			returned, ok := function.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 1 {
				return "", fmt.Errorf("mutation facade premise missing")
			}
			call, ok := returned.Results[0].(*ast.CallExpr)
			if !ok || len(call.Args) != 3 {
				return "", fmt.Errorf("mutation facade call premise missing")
			}
			if mutation == "facade_reuses_execution" {
				call.Args[0] = ast.NewIdent("nil")
			} else {
				call.Args[1] = ast.NewIdent("last")
			}
			changed = true
		}
	}
	if !changed {
		return "", fmt.Errorf("mutation did not target actual emitted flow: %s", mutation)
	}
	return public213ArrayListText(file), nil
}
