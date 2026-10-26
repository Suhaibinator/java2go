package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestJDKGenericParentProjectionPreservesOrigins(t *testing.T) {
	ctx := jdkParentExistingAPIContext(t)
	origin := func(text string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: text}, ctx)
	}
	for _, text := range []string{"List<Value>", "ArrayDeque<Value>", "List<List<Value>>", "List<Value[]>", "List<? extends Value>", "List<? super Value>", "List"} {
		actual := origin(text)
		projected, ok := projectJDK21GenericParentOrigin(actual, "java.util.Collection")
		if !ok || projected.externalName != "java.util.Collection" || len(projected.arguments) != len(actual.arguments) {
			t.Fatalf("actual %s projection unavailable", text)
		}
		for i, arg := range actual.arguments {
			if !sameJavaTypeOrigin(arg, projected.arguments[i]) {
				t.Fatalf("%s supplied argument origin changed", text)
			}
		}
		again := origin(text)
		if !sameJavaTypeOrigin(actual, again) {
			t.Fatal("projection mutated supplied origin")
		}
	}
	base := origin("List<Value>")
	cases := map[string]inferredJavaTypeOrigin{"unresolved": base, "nested unresolved": base, "array": base, "source homonym": base, "foreign owner": base, "partial arity": base, "nil": {}}
	v := cases["unresolved"]
	v.unresolved = true
	cases["unresolved"] = v
	v = cases["nested unresolved"]
	v.arguments = []inferredJavaTypeOrigin{{externalName: "java.lang.String", unresolved: true}}
	cases["nested unresolved"] = v
	v = cases["array"]
	v.arrayRank = 1
	cases["array"] = v
	v = cases["source homonym"]
	v.externalName = ""
	v.nominalScope = ctx.currentClass
	cases["source homonym"] = v
	v = cases["foreign owner"]
	v.externalName = "foreign.List"
	cases["foreign owner"] = v
	v = cases["partial arity"]
	v.arguments = append(append([]inferredJavaTypeOrigin{}, v.arguments...), v.arguments[0])
	cases["partial arity"] = v
	for name, actual := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := projectJDK21GenericParentOrigin(actual, "java.util.Collection"); ok {
				t.Fatal("unknown/noncanonical/malformed origin admitted")
			}
		})
	}
	for _, target := range []string{"foreign.Collection", "Collection", "java.lang.Object", "java.util.Missing"} {
		if _, ok := projectJDK21GenericParentOrigin(base, target); ok {
			t.Fatalf("uncertified/non-generic target %s admitted", target)
		}
	}
	for _, name := range []string{"T", "E"} {
		left, right := symbol.NewTypeParam(name, nil), symbol.NewTypeParam(name, nil)
		a, b := base, base
		a.arguments = []inferredJavaTypeOrigin{{parameter: left.Declaration, javaType: name}}
		b.arguments = []inferredJavaTypeOrigin{{parameter: right.Declaration, javaType: name}}
		pa, oka := projectJDK21GenericParentOrigin(a, "java.util.Collection")
		pb, okb := projectJDK21GenericParentOrigin(b, "java.util.Collection")
		if !oka || !okb || pa.arguments[0].parameter != left.Declaration || pb.arguments[0].parameter != right.Declaration || sameJavaTypeOrigin(pa, pb) {
			t.Fatal("same-spelled independent binder identity conflated")
		}
	}
}

func TestJDKGenericParentRecipeGraphGuards(t *testing.T) {
	parameter := func(name string) *symbol.TypeParamDeclaration { return symbol.NewTypeParam(name, nil).Declaration }
	a, b, c, d, q, r, p, s := parameter("A"), parameter("B"), parameter("C"), parameter("D"), parameter("Q"), parameter("R"), parameter("P"), parameter("S")
	varOf := func(p *symbol.TypeParamDeclaration) inferredJavaTypeOrigin {
		return inferredJavaTypeOrigin{parameter: p, javaType: p.SourceName}
	}
	nominal := func(name string, args ...inferredJavaTypeOrigin) inferredJavaTypeOrigin {
		return inferredJavaTypeOrigin{externalName: name, javaType: name, arguments: args}
	}
	arrayA := varOf(a)
	arrayA.arrayRank = 1
	listC := nominal("java.util.List", varOf(c))
	listQ := nominal("java.util.List", varOf(q))
	records := map[string]jdk21GenericParentRecord{}
	certificates := map[string]string{}
	for owner, record := range jdk21GenericParentRecords {
		records[owner] = record
		certificates[owner] = jdk21GenericParentCertificates[owner]
	}
	add := func(owner string, parameters []*symbol.TypeParamDeclaration, parents ...inferredJavaTypeOrigin) {
		records[owner] = jdk21GenericParentRecord{certificate: owner, parameters: parameters, parents: parents}
		certificates[owner] = owner
	}
	add("proof.Start", []*symbol.TypeParamDeclaration{a, b}, nominal("proof.Left", varOf(b), arrayA), nominal("proof.Right", varOf(b), arrayA))
	add("proof.Left", []*symbol.TypeParamDeclaration{c, d}, nominal("proof.Target", listC, varOf(d)))
	add("proof.Right", []*symbol.TypeParamDeclaration{q, r}, nominal("proof.Target", listQ, varOf(r)))
	add("proof.Target", []*symbol.TypeParamDeclaration{p, s})
	value, other := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Value"}}, &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Other"}}
	actual := nominal("proof.Start", inferredJavaTypeOrigin{nominalScope: value, javaType: "Value"}, inferredJavaTypeOrigin{nominalScope: other, javaType: "Other"})
	projected, ok := projectJDK21GenericParentWithRecords(actual, "proof.Target", records, certificates)
	if !ok || len(projected.arguments) != 2 || projected.arguments[0].externalName != "java.util.List" || projected.arguments[0].arguments[0].nominalScope != other || projected.arguments[1].nominalScope != value || projected.arguments[1].arrayRank != 1 {
		t.Fatal("reordered/nested declaration-backed diamond projection failed")
	}
	clone := func() map[string]jdk21GenericParentRecord {
		copy := map[string]jdk21GenericParentRecord{}
		for k, v := range records {
			copy[k] = v
		}
		return copy
	}
	for _, name := range []string{"conflicting diamond", "cycle", "missing", "stale", "arity", "nested arity", "foreign parameter", "duplicate parameter", "nil parameter"} {
		t.Run(name, func(t *testing.T) {
			graph := clone()
			switch name {
			case "conflicting diamond":
				rec := graph["proof.Right"]
				rec.parents = []inferredJavaTypeOrigin{nominal("proof.Target", nominal("java.util.List", varOf(r)), varOf(q))}
				graph["proof.Right"] = rec
			case "cycle":
				rec := graph["proof.Left"]
				rec.parents = append(append([]inferredJavaTypeOrigin{}, rec.parents...), nominal("proof.Start", varOf(c), varOf(d)))
				graph["proof.Left"] = rec
			case "missing":
				delete(graph, "proof.Right")
			case "stale":
				rec := graph["proof.Right"]
				rec.certificate = "stale"
				graph["proof.Right"] = rec
			case "arity":
				rec := graph["proof.Start"]
				rec.parents = []inferredJavaTypeOrigin{nominal("proof.Left", varOf(a))}
				graph["proof.Start"] = rec
			case "nested arity":
				rec := graph["proof.Left"]
				rec.parents = []inferredJavaTypeOrigin{nominal("proof.Target", nominal("java.util.List", varOf(c), varOf(d)), varOf(d))}
				graph["proof.Left"] = rec
			case "foreign parameter":
				rec := graph["proof.Left"]
				rec.parents = []inferredJavaTypeOrigin{nominal("proof.Target", varOf(a), varOf(d))}
				graph["proof.Left"] = rec
			case "duplicate parameter":
				rec := graph["proof.Left"]
				rec.parameters = []*symbol.TypeParamDeclaration{c, c}
				graph["proof.Left"] = rec
			case "nil parameter":
				rec := graph["proof.Left"]
				rec.parameters = []*symbol.TypeParamDeclaration{nil, d}
				graph["proof.Left"] = rec
			}
			if _, ok := projectJDK21GenericParentWithRecords(actual, "proof.Target", graph, certificates); ok {
				t.Fatal("uncertified/ambiguous recipe graph admitted")
			}
		})
	}
}
