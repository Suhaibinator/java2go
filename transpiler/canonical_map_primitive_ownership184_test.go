package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestCanonicalMapPrimitive184SourceOwnershipShields(t *testing.T) {
	for _, row := range []struct {
		name  string
		files map[string]string
	}{
		{"Map", map[string]string{"Probe.java": `class Map{String equals(String x){return x;}String hashCode(int x){return "source";}}class Probe{String probe(Map value){return value.equals("source")+value.hashCode(1);}}`}},
		{"ConcurrentMap", map[string]string{"Probe.java": `class ConcurrentMap{String equals(String x){return x;}String hashCode(int x){return "source";}}class Probe{String probe(ConcurrentMap value){return value.equals("source")+value.hashCode(1);}}`}},
		{"ConcurrentHashMap", map[string]string{"Probe.java": `class ConcurrentHashMap{String equals(String x){return x;}String hashCode(int x){return "source";}}class Probe{String probe(ConcurrentHashMap value){return value.equals("source")+value.hashCode(1);}}`}},
		{"dedicated_key_source_name", map[string]string{"Probe.java": `class JavaMap{String equals(String x){return x;}String hashCode(int x){return "source";}}class Probe{String probe(JavaMap value){return value.equals("source")+value.hashCode(1);}}`}},
		{"class_binder", map[string]string{"Probe.java": `class Source{String equals(String x){return x;}String hashCode(int x){return "source";}}class Probe<Map extends Source>{String probe(Map value){return value.equals("source")+value.hashCode(1);}}`}},
		{"method_binder", map[string]string{"Probe.java": `class Source{String equals(String x){return x;}String hashCode(int x){return "source";}}class Probe{<ConcurrentMap extends Source>String probe(ConcurrentMap value){return value.equals("source")+value.hashCode(1);}}`}},
		{"foreign_import", map[string]string{"foreign/Map.java": `package foreign;public class Map{public String equals(String x){return x;}public String hashCode(int x){return "source";}}`, "Probe.java": `import foreign.Map;class Probe{String probe(Map value){return value.equals("source")+value.hashCode(1);}}`}},
		{"qualified_source", map[string]string{"foreign/Map.java": `package foreign;public class Map{public String equals(String x){return x;}public String hashCode(int x){return "source";}}`, "Probe.java": `class Probe{String probe(foreign.Map value){return value.equals("source")+value.hashCode(1);}}`}},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, row.files, "Probe.java")
			scope := ctx.currentFile.FindClassScope("Probe")
			if scope == nil {
				t.Fatal("Probe missing")
			}
			ctx = classScopeCtx(scope, ctx)
			definitions := scope.FindMethod().ByOriginalName("probe")
			if len(definitions) != 1 {
				t.Fatal("probe missing")
			}
			ctx.localScope = definitions[0]
			source := ctx.currentFile.Source
			var invocations []*sitter.Node
			var walk func(*sitter.Node)
			walk = func(node *sitter.Node) {
				if node == nil {
					return
				}
				if node.Type() == "method_invocation" {
					invocations = append(invocations, node)
				}
				for i := 0; i < int(node.NamedChildCount()); i++ {
					walk(node.NamedChild(i))
				}
			}
			walk(definitions[0].DeclarationNode)
			if len(invocations) != 2 {
				t.Fatalf("source calls=%d", len(invocations))
			}
			for _, node := range invocations {
				name := node.ChildByFieldName("name").Content(source)
				if key, owned := intrinsicMethodReceiverTypeName(node.ChildByFieldName("object"), name, ctx, source); owned {
					t.Fatalf("source method stolen by intrinsic owner %q", key)
				}
				if got, known := inferIntrinsicMethodResultType(node, ctx, source); known {
					t.Fatalf("source result stolen by primitive catalog: %q", got)
				}
				got, known := inferExprJavaType(node, ctx, source)
				if !known || !isBuiltinJavaString(got, ctx) {
					t.Fatalf("source overloaded result lost: %q,%v", got, known)
				}
			}
		})
	}
}

func TestCanonicalMapPrimitive184QualifiedOwnerDespiteSourceNames(t *testing.T) {
	for _, row := range []struct{ method, result string }{{"equals", "boolean"}, {"hashCode", "int"}} {
		node, ctx, source := mapPrimitive184Invocation(t, "java.util.Map<java.lang.String,java.lang.Integer>", row.method, row.result)
		// Live caller type parameters share library simple names; they do not own
		// an exact canonical declared receiver or its primitive method result.
		ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("Map", nil), symbol.NewTypeParam("Boolean", nil), symbol.NewTypeParam("Integer", nil)}
		origin := inferredJavaTypeOrigin{}
		got, known := inferExprJavaType(node, ctx, source, &origin)
		if !known || got != row.result || origin.javaType != row.result || origin.parameter != nil || origin.nominalScope != nil || origin.externalName != "" {
			t.Fatalf("canonical primitive result rebound: %q,%v,%+v", got, known, origin)
		}
	}
}
