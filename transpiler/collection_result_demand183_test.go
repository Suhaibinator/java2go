package transpiler

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func collectionDemand183Invocation(t *testing.T, root *sitter.Node, source []byte, text string) *sitter.Node {
	t.Helper()
	var found *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" && n.Content(source) == text {
			found = n
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(root)
	if found == nil {
		t.Fatalf("actual parsed invocation absent: %s", text)
	}
	return found
}

// Reuse the existing bounded real-parent subprocess transport, but validate
// these work rows independently of the older superclass benchmark schema.
func collectionDemand183Isolated(t *testing.T, requiredRows ...int) bool {
	t.Helper()
	if request, child := superclass4059ChildRequest(t); child {
		if request.Mode != "collection-demand183" {
			t.Fatal("wrong work child mode")
		}
		return false
	}
	shared := superclass4059Flag("test.gocoverdir")
	if !superclass4059CanIsolate(testing.CoverMode(), shared) {
		t.Log("COLLECTION_DEMAND183_UNISOLATED_COVERAGE_FALLBACK: all original assertions retained")
		return false
	}
	result := superclass4059RunChild(t, superclass4059Request{Parent: t.Name(), Mode: "collection-demand183"}, superclass4059ProcessOptions{sharedDir: shared})
	superclass4059LogResult(t, result)
	if err := superclass4059Completed(result, t.Name()); err != nil {
		t.Fatal(err)
	}
	count := 6
	if len(requiredRows) != 0 {
		count = requiredRows[0]
	}
	if strings.Count(result.Stdout, "COLLECTION_DEMAND_WORK183 ") != count {
		t.Fatal("measurement child must complete every actual parsed work row")
	}
	return true
}

func collectionDemand183Native(t *testing.T, fn func() string) (int64, string) {
	t.Helper()
	want := fn()
	for i := 0; i < 2; i++ {
		if fn() != want {
			t.Fatal("unstable warm result")
		}
	}
	before := runtime.NumCgoCall()
	for i := 0; i < 3; i++ {
		if fn() != want {
			t.Fatal("unstable measured result")
		}
	}
	delta := runtime.NumCgoCall() - before
	if delta%3 != 0 {
		t.Fatalf("native finalizer/concurrent noise: %d", delta)
	}
	return delta / 3, want
}

type collectionDemand183Work struct {
	Invocation                    string
	Receiver                      string
	Native, NameRead, OneElements int64
	Result                        string
	ConsumesElements              bool
}

func collectionDemand183WorkRows(t *testing.T, budget bool) {
	t.Helper()
	source := `import java.util.List; import java.util.Map; import java.util.Optional; import java.util.OptionalInt;
class Probe {
 StringBuilder b; List<String> list; Map<String,Integer> map; Optional<String> optional; OptionalInt primitive;
 void run(){b.append("a").append("b").append("c").append("d").append("e").unknown(); list.size(); map.clear(); optional.isPresent(); primitive.getAsInt(); map.entrySet();}
}`
	h := setupParseHelper(t, source)
	defer runtime.KeepAlive(h)
	ctx := h.Ctx.Clone()
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	for _, tc := range []struct {
		invocation, receiver, want string
		consumes                   bool
	}{
		{`b.append("a").append("b").append("c").append("d").append("e").unknown()`, "StringBuilder", "/false", false},
		{"list.size()", "List", "int/true", false},
		{"map.clear()", "JavaMap", "void/true", false},
		{"optional.isPresent()", "Optional", "boolean/true", false},
		{"primitive.getAsInt()", "OptionalInt", "int/true", false},
		{"map.entrySet()", "JavaMap", "java.util.Set<java.util.Map.Entry<String, Integer>>/true", true},
	} {
		node := collectionDemand183Invocation(t, h.File.Ast, []byte(source), tc.invocation)
		actual, known := intrinsicReceiverTypeName(node.ChildByFieldName("object"), ctx, []byte(source))
		if !known || actual != tc.receiver {
			t.Fatalf("parsed receiver ownership: got %s/%t want %s", actual, known, tc.receiver)
		}
		nameCost, _ := collectionDemand183Native(t, func() string { return node.ChildByFieldName("name").Content([]byte(source)) })
		oneCost, _ := collectionDemand183Native(t, func() string {
			return strings.Join(receiverElementJavaTypes(node.ChildByFieldName("object"), ctx, []byte(source)), ",")
		})
		native, result := collectionDemand183Native(t, func() string {
			v, k := intrinsicCollectionMethodResultType(node, actual, ctx, []byte(source))
			return fmt.Sprintf("%s/%t", v, k)
		})
		if result != tc.want {
			t.Fatalf("result contract %s: got %s want %s", tc.invocation, result, tc.want)
		}
		row := collectionDemand183Work{tc.invocation, actual, native, nameCost, oneCost, result, tc.consumes}
		encoded, _ := json.Marshal(row)
		t.Log("COLLECTION_DEMAND_WORK183 " + string(encoded))
		if oneCost == 0 {
			t.Fatal("control did not execute actual receiver inference")
		}
		if budget && !tc.consumes && native > nameCost {
			t.Errorf("UNUSED_RECEIVER_INFERENCE183 %s: actual native calls %d exceed actual name-only work %d", tc.invocation, native, nameCost)
		}
		if budget && tc.consumes && native > nameCost+oneCost {
			t.Errorf("REPEATED_RECEIVER_INFERENCE183 %s: actual %d exceeds name+one derivation %d", tc.invocation, native, nameCost+oneCost)
		}
	}
}

func TestCollectionResultDemand183WorkBaseline(t *testing.T) {
	if collectionDemand183Isolated(t) {
		return
	}
	collectionDemand183WorkRows(t, false)
}
func TestCollectionResultDemand183NativeBudget(t *testing.T) {
	if collectionDemand183Isolated(t) {
		return
	}
	collectionDemand183WorkRows(t, true)
}

func TestCollectionResultDemand183ElementContracts(t *testing.T) {
	source := `import java.util.*; import java.util.concurrent.*; import java.util.function.*;
class Probe {
 List<String> list; Map<String,Integer> map; Set<String> set; Iterator<String> iterator; Iterator raw;
 Future<String> future; ThreadLocal<String> thread; Callable<String> callable;
 Optional<String> optional; OptionalInt primitive; Function<String,Long> function; BiFunction<String,Integer,Long> binary; Supplier<String> supplier;
 void run(){list.get(0);list.set(0,"x");list.add("x");map.get("x");map.entrySet();map.keySet();map.values();map.remove("x",1);set.isEmpty();iterator.next();raw.next();future.get();thread.get();callable.call();optional.get();optional.filter(x->true);optional.map(x->x.length());optional.map(x->null);optional.flatMap(x->Optional.of(x));primitive.getAsInt();function.apply("x");binary.apply("x",1);supplier.get();}
}`
	h := setupParseHelper(t, source)
	defer runtime.KeepAlive(h)
	ctx := h.Ctx.Clone()
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	for _, tc := range []struct{ call, receiver, want string }{
		{`list.get(0)`, "List", "String"}, {`list.set(0,"x")`, "List", "String"}, {`list.add("x")`, "List", "boolean"},
		{`map.get("x")`, "JavaMap", "Integer"}, {`map.entrySet()`, "JavaMap", "java.util.Set<java.util.Map.Entry<String, Integer>>"}, {`map.keySet()`, "JavaMap", "java.util.Set<String>"}, {`map.values()`, "JavaMap", "java.util.Collection<Integer>"}, {`map.remove("x",1)`, "JavaMap", "boolean"},
		{`set.isEmpty()`, "Set", "boolean"}, {`iterator.next()`, "Iterator", "String"}, {`raw.next()`, "Iterator", "Object"},
		{`future.get()`, "Future", "String"}, {`thread.get()`, "ThreadLocal", "String"}, {`callable.call()`, "Callable", "String"},
		{`optional.get()`, "Optional", "String"}, {`optional.filter(x->true)`, "Optional", "Optional<String>"}, {`optional.map(x->x.length())`, "Optional", "Optional<java.lang.Integer>"}, {`optional.flatMap(x->Optional.of(x))`, "Optional", "Optional<String>"},
		{`primitive.getAsInt()`, "OptionalInt", "int"}, {`function.apply("x")`, "Function", "Long"}, {`binary.apply("x",1)`, "BiFunction", "Long"}, {`supplier.get()`, "Supplier", "String"},
	} {
		t.Run(tc.call, func(t *testing.T) {
			node := collectionDemand183Invocation(t, h.File.Ast, []byte(source), tc.call)
			actual, known := intrinsicReceiverTypeName(node.ChildByFieldName("object"), ctx, []byte(source))
			if !known || actual != tc.receiver {
				t.Fatalf("owner=%s/%t want%s", actual, known, tc.receiver)
			}
			got, ok := intrinsicCollectionMethodResultType(node, actual, ctx, []byte(source))
			if !ok || got != tc.want {
				t.Fatalf("got %s/%t want %s/true", got, ok, tc.want)
			}
		})
	}
	node := collectionDemand183Invocation(t, h.File.Ast, []byte(source), `optional.map(x->null)`)
	for _, rootMatches := range []bool{false, true} {
		local := ctx.Clone()
		local.expectedType = "Optional<String>"
		local.expectedTypeRoot = h.File.Ast
		if rootMatches {
			local.expectedTypeRoot = node
		}
		got, ok := intrinsicCollectionMethodResultType(node, "Optional", local, []byte(source))
		want := "Optional<Object>"
		if rootMatches {
			want = "Optional<String>"
		}
		if !ok || got != want {
			t.Fatalf("expected-root%t: got %s/%t want%s", rootMatches, got, ok, want)
		}
	}
}
