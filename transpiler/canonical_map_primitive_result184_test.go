package transpiler

import (
	"fmt"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func mapPrimitive184Invocation(t *testing.T, javaType, method, result string) (*sitter.Node, Ctx, []byte) {
	t.Helper()
	arguments := ""
	if method == "equals" {
		arguments = "other"
	}
	source := []byte(fmt.Sprintf("class Probe {%s probe(%s value,java.lang.Object other){return value.%s(%s);}}", result, javaType, method, arguments))
	helper := setupParseHelper(t, string(source))
	scope := helper.File.Symbols.FindClassScope("Probe")
	if scope == nil {
		t.Fatal("Probe declaration missing")
	}
	definitions := scope.FindMethod().ByOriginalName("probe")
	if len(definitions) != 1 {
		t.Fatal("probe method declaration missing")
	}
	ctx := classScopeCtx(scope, helper.Ctx)
	ctx.localScope = definitions[0]
	var invocation *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || invocation != nil {
			return
		}
		if node.Type() == "method_invocation" {
			invocation = node
			return
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}
	walk(definitions[0].DeclarationNode)
	if invocation == nil {
		t.Fatal("probe invocation missing")
	}
	return invocation, ctx, source
}

func TestCanonicalMapPrimitive184ExistingInference(t *testing.T) {
	for _, row := range []struct{ javaType, key string }{
		{"java.util.Map<java.lang.String,java.lang.Integer>", "JavaMap"},
		{"java.util.concurrent.ConcurrentMap<java.lang.String,java.lang.Integer>", "JavaConcurrentMap"},
		{"java.util.concurrent.ConcurrentHashMap<java.lang.String,java.lang.Integer>", "JavaConcurrentHashMap"},
	} {
		for _, operation := range []struct{ name, result string }{{"equals", "boolean"}, {"hashCode", "int"}} {
			t.Run(row.key+"/"+operation.name, func(t *testing.T) {
				node, ctx, source := mapPrimitive184Invocation(t, row.javaType, operation.name, operation.result)
				receiver, owned := intrinsicMethodReceiverTypeName(node.ChildByFieldName("object"), operation.name, ctx, source)
				if !owned || receiver != row.key {
					t.Fatalf("fixture receiver ownership=%q,%v want %q", receiver, owned, row.key)
				}
				got, known := inferIntrinsicMethodResultType(node, ctx, source)
				if !known || got != operation.result {
					t.Errorf("canonical map result missing: %s.%s got=%q known=%v want=%s", row.key, operation.name, got, known, operation.result)
				}
				origin := inferredJavaTypeOrigin{externalName: "must-be-replaced", unresolved: true}
				typ, known := inferExprJavaType(node, ctx, source, &origin)
				if !known || typ != operation.result || origin.javaType != operation.result || origin.arrayRank != 0 || origin.externalName != "" || origin.nominalScope != nil || origin.parameter != nil || origin.unresolved {
					t.Errorf("canonical map primitive inference/origin missing: %s.%s got=%q known=%v origin=%+v", row.key, operation.name, typ, known, origin)
				}
			})
		}
	}
}

func TestCanonicalMapPrimitive184ExistingConcatLowering(t *testing.T) {
	for _, javaType := range []string{"java.util.Map<java.lang.String,java.lang.Integer>", "java.util.concurrent.ConcurrentMap<java.lang.String,java.lang.Integer>", "java.util.concurrent.ConcurrentHashMap<java.lang.String,java.lang.Integer>"} {
		t.Run(javaType, func(t *testing.T) {
			output := renderGoFileFromJava(t, fmt.Sprintf(`class Probe {String probe(%s value,Object other){return ":"+value.equals(other)+":"+value.hashCode();}}`, javaType))
			for _, want := range []string{"stdjava.JavaStringValueOfBoolean(stdjava.ObjectEqualsExecution(", "stdjava.JavaStringValueOfInt(int32(stdjava.ObjectHashCodeExecution("} {
				if !strings.Contains(output, want) {
					t.Errorf("primitive concat lowering missing %q; generated=%s", want, output)
				}
			}
			for _, bad := range []string{"stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.ObjectEqualsExecution(", "stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.ObjectHashCodeExecution("} {
				if strings.Contains(output, bad) {
					t.Errorf("primitive result incorrectly routed through reference operand: %s", bad)
				}
			}
		})
	}
}

func TestCanonicalMapPrimitive184LegacyInferenceShield(t *testing.T) {
	for _, row := range []struct{ javaType, key string }{
		{"java.util.HashMap<java.lang.String,java.lang.Integer>", "HashMap"},
		{"java.util.LinkedHashMap<java.lang.String,java.lang.Integer>", "LinkedHashMap"},
		{"java.util.TreeMap<java.lang.String,java.lang.Integer>", "TreeMap"},
		{"java.util.List<java.lang.Integer>", "List"},
		{"java.util.Set<java.lang.Integer>", "Set"},
	} {
		for _, operation := range []struct{ name, result string }{{"equals", "boolean"}, {"hashCode", "int"}} {
			t.Run(row.key+"/"+operation.name, func(t *testing.T) {
				node, ctx, source := mapPrimitive184Invocation(t, row.javaType, operation.name, operation.result)
				got, known := inferIntrinsicMethodResultType(node, ctx, source)
				if !known || got != operation.result {
					t.Fatalf("legacy inference shield changed: %s.%s got=%q known=%v want=%s", row.key, operation.name, got, known, operation.result)
				}
			})
		}
	}
}
