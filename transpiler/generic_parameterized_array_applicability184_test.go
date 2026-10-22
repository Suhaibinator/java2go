package transpiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

const poolArray184Body = `
class Owner {
 static class MemoryRegionCache<T> { T value; }
 MemoryRegionCache<java.nio.ByteBuffer>[] direct;
 MemoryRegionCache<byte[]>[] heap;
 private MemoryRegionCache<?> cache(Object area,int sizeIdx,int sizeClass){return null;}
 private static <T> MemoryRegionCache<T> cache(MemoryRegionCache<T>[] cache,int sizeIdx){
  if(cache==null || sizeIdx>cache.length-1){return null;} return cache[sizeIdx];
 }
 MemoryRegionCache<?> direct(int sizeIdx){return cache(direct,sizeIdx);}
 MemoryRegionCache<?> heap(int sizeIdx){return cache(heap,sizeIdx);}
}
`

func poolArray184Origin(o inferredJavaTypeOrigin) map[string]any {
	children := make([]map[string]any, 0, len(o.arguments))
	for _, a := range o.arguments {
		children = append(children, poolArray184Origin(a))
	}
	return map[string]any{"java": o.javaType, "rank": o.arrayRank, "nominal": fmt.Sprintf("%p", o.nominalScope), "parameter": fmt.Sprintf("%p", o.parameter), "external": o.externalName, "unresolved": o.unresolved, "wildcard": o.wildcard, "arguments": children}
}
func poolArray184Definition(d *symbol.Definition) map[string]any {
	if d == nil {
		return nil
	}
	p := []string{}
	for _, a := range d.Parameters {
		p = append(p, a.OriginalType)
	}
	params := []string{}
	for _, a := range d.TypeParameters {
		params = append(params, fmt.Sprintf("%s/%s/%p", a.Name, a.EmittedName(), a.Declaration))
	}
	return map[string]any{"ptr": fmt.Sprintf("%p", d), "name": d.Name, "original": d.OriginalName, "static": d.IsStatic, "private": d.IsPrivate, "parameters": p, "typeParameters": params, "result": d.OriginalType}
}
func poolArray184Call(t *testing.T, h *ParseHelper, owner *symbol.ClassScope, callerName string) (*symbol.Definition, *sitter.Node, Ctx) {
	t.Helper()
	ctx := h.Ctx.Clone()
	ctx.currentClass = owner
	ctx.className = owner.Class.Name
	for _, d := range owner.Methods {
		if d.OriginalName == callerName {
			ctx.localScope = d
			break
		}
	}
	if ctx.localScope == nil {
		t.Fatal("actual caller missing")
	}
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if call != nil {
			return
		}
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(h.File.Source) == "cache" {
			call = n
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if call == nil {
		t.Fatal("actual call missing")
	}
	var target *symbol.Definition
	for _, d := range owner.Methods {
		if d.OriginalName == "cache" && d.IsStatic && len(d.Parameters) == 2 && len(d.TypeParameters) > 0 {
			target = d
		}
	}
	if target == nil {
		t.Fatal("registered private static generic two-argument definition missing")
	}
	return target, call, ctx
}
func poolArray184Facts(t *testing.T, h *ParseHelper, owner *symbol.ClassScope, caller string) (*methodResolution, *symbol.Definition) {
	t.Helper()
	def, call, ctx := poolArray184Call(t, h, owner, caller)
	args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
	formals := methodFormalOrigins(def, owner, nil, ctx)
	rows := []map[string]any{}
	for i, arg := range args {
		var actual inferredJavaTypeOrigin
		typ, known := inferExprJavaType(arg, ctx, h.File.Source, &actual)
		actual.javaType = typ
		c, e, a, handled := declaredOriginConversion(actual, formals[i], ctx)
		expected := receiverInstantiatedMethodFormal(def.Parameters[i], def, owner, nil, ctx)
		fc, fe, fa := javaInvocationConversionCost(arg, expected, methodCandidateTypeParameterNames(owner, def), ctx, h.File.Source)
		projected, proved := declaredMethodArgumentInferredOrigin(formals[i], def, formals, args, false, ctx, h.File.Source)
		pc, pe, pa, ph := declaredOriginConversion(actual, projected, ctx)
		rows = append(rows, map[string]any{"arg": arg.Content(h.File.Source), "known": known, "actual": poolArray184Origin(actual), "formal": poolArray184Origin(formals[i]), "conversion": []any{c, e, a, handled}, "fallbackExpected": expected, "fallback": []any{fc, fe, fa}, "projectionProved": proved, "projected": poolArray184Origin(projected), "projectedConversion": []any{pc, pe, pa, ph}})
	}
	bounds, boundsOK := methodInvocationConstraintBindings(def, owner, args, false, ctx, h.File.Source)
	score, applicable := scoreMethodCandidate(def, owner, methodCandidateTypeParameterNames(owner, def), args, ctx, h.File.Source)
	selected := findBestMethodInHierarchy(owner, "cache", call.ChildByFieldName("arguments"), true, true, ctx, h.File.Source)
	var selectedDef *symbol.Definition
	if selected != nil {
		selectedDef = selected.def
	}
	physical := planStaticGenericMethodPhysicalABI(owner, def, ctx)
	physicalBindings := genericArrayInvocationTypeBindings(def, call, ctx, h.File.Source)
	javaBindings := genericJavaInvocationTypeBindings(def, call, ctx, h.File.Source)
	result, resultKnown := inferredGenericInvocationResult(def, call, ctx, h.File.Source)
	b, err := json.Marshal(map[string]any{"caller": caller, "invocation": call.Content(h.File.Source), "registeredDefinition": poolArray184Definition(def), "selectedDefinition": poolArray184Definition(selectedDef), "args": rows, "bounds": bounds, "boundsApplicable": boundsOK, "score": fmt.Sprintf("%+v", score), "scoreApplicable": applicable, "privatePhysicalABIRefused": physical == nil, "physicalTypeBindings": physicalBindings, "javaTypeBindings": javaBindings, "inferredResult": result, "resultKnown": resultKnown})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("POOL_ARRAY184_FACTS " + string(b))
	if physical != nil {
		t.Fatal("private static ABI admission unexpectedly widened")
	}
	return selected, def
}
func TestGenericParameterizedArrayPrivateStaticFacts(t *testing.T) {
	for _, imports := range []string{"", "import static java.lang.Math.max;\n"} {
		t.Run(fmt.Sprintf("staticImport%t", imports != ""), func(t *testing.T) {
			h := setupParseHelper(t, imports+poolArray184Body)
			owner := h.File.Symbols.FindClassScope("Owner")
			if owner == nil {
				t.Fatal("owner missing")
			}
			for _, name := range []string{"direct", "heap"} {
				poolArray184Facts(t, h, owner, name)
			}
		})
	}
}
func TestGenericParameterizedArrayPrivateStaticSelection(t *testing.T) {
	for _, imports := range []string{"", "import static java.lang.Math.max;\n"} {
		t.Run(fmt.Sprintf("staticImport%t", imports != ""), func(t *testing.T) {
			h := setupParseHelper(t, imports+poolArray184Body)
			owner := h.File.Symbols.FindClassScope("Owner")
			for _, name := range []string{"direct", "heap"} {
				got, want := poolArray184Facts(t, h, owner, name)
				if got == nil || got.def != want || got.owner != owner || !got.def.IsStatic || !got.def.IsPrivate {
					t.Errorf("%s did not select exact registered private static generic two-argument definition", name)
				}
			}
		})
	}
}
func TestGenericParameterizedArrayPrivateStaticStrictEntry(t *testing.T) {
	h := setupParseHelper(t, "import static java.lang.Math.max;\n"+poolArray184Body)
	previous := diagnostics.strict
	setStrictMode(true)
	t.Cleanup(func() { setStrictMode(previous) })
	var caught any
	var result any
	func() { defer func() { caught = recover() }(); result = ParseNode(h.File.Ast, h.File.Source, h.Ctx) }()
	if caught != nil {
		t.Fatalf("strict actual-source entry failed: %v", caught)
	}
	file, ok := result.(*ast.File)
	if !ok {
		t.Fatalf("expected Go AST file got %T", result)
	}
	var out bytes.Buffer
	if err := printer.Fprint(&out, token.NewFileSet(), file); err != nil {
		t.Fatal(err)
	}
	t.Log("POOL_ARRAY184_GO_AST\n" + out.String())
	if strings.Contains(out.String(), "unsupported intrinsic invocation") {
		t.Fatal("strict entry retained unsupported invocation")
	}
}

type arrayApplicability184Case struct {
	name, declarations, caller string
	generic                    bool
	negative                   bool
}

func arrayApplicability184Cases() []arrayApplicability184Case {
	strictGeneric := `private static <T extends Base> int cache(Cell<T>[] v,int n){return 1;}`
	boxedOther := `private static long cache(Object v,Integer n){return 2;}`
	genericBoxed := `private static <T extends Base> int cache(Cell<T>[] v,Integer n){return 1;}`
	strictOther := `private static long cache(Object v,int n){return 2;}`
	return []arrayApplicability184Case{
		{"strict generic before boxed", strictGeneric + boxedOther, `static int run(Cell<Child>[] v){return cache(v,0);}`, true, false},
		{"strict generic after boxed", boxedOther + strictGeneric, `static int run(Cell<Child>[] v){return cache(v,0);}`, true, false},
		{"other argument requires boxing", genericBoxed + strictOther, `static long run(Cell<Child>[] v){return cache(v,0);}`, false, false},
		{"other argument boxing reversed", strictOther + genericBoxed, `static long run(Cell<Child>[] v){return cache(v,0);}`, false, false},
		{"caller binder identity", strictGeneric, `static <T extends Base> int run(Cell<T>[] v){return cache(v,0);}`, true, false},
		// Invalid-Java parser shields. No JVM parity is claimed for these cases.
		{"explicit incompatible origin", strictGeneric, `static int run(Cell<Child>[] v){return Owner.<Other>cache(v,0);}`, false, true},
		{"declared bound violated", `private static <T extends Child> int cache(Cell<T>[] v,int n){return 1;}`, `static int run(Cell<Other>[] v){return cache(v,0);}`, false, true},
		{"invariant conflicting arrays", `private static <T extends Base> int cache(Cell<T>[] a,Cell<T>[] b){return 1;}`, `static int run(Cell<Child>[] a,Cell<Other>[] b){return cache(a,b);}`, false, true},
		{"array rank mismatch", strictGeneric, `static int run(Cell<Child>[][] v){return cache(v,0);}`, false, true},
		// Retain the existing deferred capture policy; this is not a claim Java rejects capture.
		{"wildcard remains deferred", strictGeneric, `static int run(Cell<? extends Base>[] v){return cache(v,0);}`, false, true},
	}
}
func arrayApplicability184Setup(t *testing.T, tc arrayApplicability184Case) (*ParseHelper, *symbol.ClassScope, *symbol.Definition, *sitter.Node, Ctx) {
	t.Helper()
	h := setupParseHelper(t, `class Base{} class Child extends Base{} class Other extends Base{} class Cell<X>{} class Owner{`+tc.declarations+tc.caller+`}`)
	owner := h.File.Symbols.FindClassScope("Owner")
	def, call, ctx := poolArray184Call(t, h, owner, "run")
	return h, owner, def, call, ctx
}
func TestGenericParameterizedArrayPhaseFacts(t *testing.T) {
	for _, tc := range arrayApplicability184Cases() {
		t.Run(tc.name, func(t *testing.T) {
			h, owner, _, _, _ := arrayApplicability184Setup(t, tc)
			poolArray184Facts(t, h, owner, "run")
		})
	}
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("canonical raw explicit%t", explicit), func(t *testing.T) {
			call := "cache(v,0)"
			if explicit {
				call = "Owner.<String>cache(v,0)"
			}
			tc := arrayApplicability184Case{declarations: `private static<T>int cache(java.util.List<T>[]v,int n){return 1;}`, caller: `static int run(java.util.List[]v){return ` + call + `;}`}
			h, owner, _, _, _ := arrayApplicability184Setup(t, tc)
			poolArray184Facts(t, h, owner, "run")
		})
	}
}
func TestGenericParameterizedArrayPhasesAndRefusals(t *testing.T) {
	for _, tc := range arrayApplicability184Cases() {
		t.Run(tc.name, func(t *testing.T) {
			h, owner, def, call, ctx := arrayApplicability184Setup(t, tc)
			got, _ := poolArray184Facts(t, h, owner, "run")
			if tc.negative {
				if got != nil {
					t.Fatalf("retained refusal selected %v", poolArray184Definition(got.def))
				}
				return
			}
			if got == nil {
				t.Fatal("no applicable source declaration selected")
			}
			if (got.def == def) != tc.generic {
				t.Fatalf("wrong overload: got=%v generic=%t", poolArray184Definition(got.def), tc.generic)
			}
			if tc.generic {
				args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
				score, ok := scoreMethodCandidate(def, owner, methodCandidateTypeParameterNames(owner, def), args, ctx, h.File.Source)
				if !ok || score.phase != 0 {
					t.Fatalf("exact generic conversion must stay strict, score=%+v applies=%t", score, ok)
				}
			}
			if tc.name == "caller binder identity" {
				formals := methodFormalOrigins(def, owner, nil, ctx)
				args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
				projected, ok := declaredMethodArgumentInferredOrigin(formals[0], def, formals, args, false, ctx, h.File.Source)
				if !ok || len(projected.arguments) != 1 || projected.arguments[0].parameter != ctx.localScope.TypeParameters[0].Declaration || projected.arguments[0].parameter == def.TypeParameters[0].Declaration {
					t.Fatal("caller binder identity was rebound to candidate method binder")
				}
			}
		})
	}
}
func TestGenericParameterizedArrayCanonicalRawPolicy(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit%t", explicit), func(t *testing.T) {
			call := "cache(v,0)"
			if explicit {
				call = "Owner.<String>cache(v,0)"
			}
			tc := arrayApplicability184Case{declarations: `private static<T>int cache(java.util.List<T>[]v,int n){return 1;}`, caller: `static int run(java.util.List[]v){return ` + call + `;}`}
			h, owner, def, n, ctx := arrayApplicability184Setup(t, tc)
			args := nodeutil.SemanticNamedChildrenOf(n.ChildByFieldName("arguments"))
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(args[0], ctx, h.File.Source, &actual)
			actual.javaType = typ
			formal := methodFormalOrigins(def, owner, nil, ctx)[0]
			_, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
			if !known || !exact || !applies || !handled {
				t.Fatalf("canonical raw conversion no longer uses existing authoritative exact path: %t/%t/%t/%t", known, exact, applies, handled)
			}
			got, _ := poolArray184Facts(t, h, owner, "run")
			if got == nil || got.def != def {
				t.Fatal("existing raw generic declaration admission changed")
			}
		})
	}
}
