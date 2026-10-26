package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestJDK21SystematicGenericParentExistingAPI(t *testing.T) {
	ctx := rawReferenceTestContext(t)
	origin := func(s string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: s}, ctx)
	}
	for _, row := range []struct {
		actual, formal string
		applies        bool
	}{
		{"java.lang.StringBuilder", "java.lang.Comparable<java.lang.StringBuilder>", true},
		{"java.lang.StringBuilder", "java.lang.Comparable<java.lang.Object>", false},
		{"java.util.HashMap<java.lang.String,java.lang.Integer>", "java.util.Map<java.lang.String,java.lang.Integer>", true},
		{"java.util.HashMap<java.lang.String,java.lang.Integer>", "java.util.Map<java.lang.String,java.lang.Number>", false},
	} {
		t.Run(row.actual+"->"+row.formal, func(t *testing.T) {
			a, f := origin(row.actual), origin(row.formal)
			c, e, ok, h := declaredOriginConversion(a, f, ctx)
			t.Logf("conversion=(%d,%t,%t,%t)", c, e, ok, h)
			if ok != row.applies || !h || ok && (e || c < 1) {
				t.Error("actual JDK generic ancestry/invariance differs")
			}
		})
	}
	raw := origin("java.util.HashMap")
	projected, ok := projectJDK21GenericParentOrigin(raw, "java.util.Map")
	t.Logf("rawHashMapProjection=%t origin=%s", ok, gsonParentDiagnosticOrigin(projected, nil))
	if !ok || projected.externalName != "java.util.Map" || len(projected.arguments) != 0 {
		t.Error("raw generic declaration must project erased raw parent")
	}
}

func TestJDK21SystematicEveryEmittedRecordProof(t *testing.T) {
	tokens := map[*symbol.TypeParamDeclaration]string{}
	if len(jdk21GenericParentRecords) < 1000 {
		t.Fatal("systematic profile unexpectedly absent")
	}
	for owner, record := range jdk21GenericParentRecords {
		if record.certificate == "" || record.certificate != jdk21GenericParentCertificates[owner] {
			t.Fatalf("uncertified %s", owner)
		}
		owned := map[*symbol.TypeParamDeclaration]bool{}
		for _, p := range record.parameters {
			if p == nil || owned[p] || tokens[p] != "" {
				t.Fatalf("declaration token reused %s", owner)
			}
			owned[p] = true
			tokens[p] = owner
		}
		for _, parent := range record.parents {
			if !validJDK21GenericParentRecipe(parent, owned, jdk21GenericParentRecords) {
				t.Fatalf("unrepresentable/incomplete recipe emitted %s", owner)
			}
		}
		if len(record.parameters) > 0 {
			actual := inferredJavaTypeOrigin{externalName: owner, javaType: owner}
			for _, p := range record.parameters {
				actual.arguments = append(actual.arguments, inferredJavaTypeOrigin{parameter: p, javaType: p.SourceName})
			}
			projected, ok := projectJDK21GenericParentOrigin(actual, owner)
			if !ok || !sameJavaTypeOrigin(actual, projected) {
				t.Fatalf("whole closure/declaration identity failed %s", owner)
			}
		}
	}
	// These supported JDK declarations have metadata shapes outside this Go schema.
	// Their facts remain in the held ledger, rather than partial graph entries.
	for _, name := range []string{"java.lang.Class", "java.lang.invoke.MethodType", "java.time.chrono.ChronoLocalDateTime", "java.time.chrono.ChronoZonedDateTime"} {
		if _, ok := jdk21GenericParentRecords[name]; ok {
			t.Fatalf("wildcard-bearing recipe must remain held %s", name)
		}
	}
}

func TestJDK21SystematicCallerDeclarationAndSourceShields(t *testing.T) {
	ctx := rawReferenceTestContext(t)
	source := declaredJavaTypeOrigin(symbol.JavaType{Original: "p.Value"}, ctx)
	classParam := ctx.currentClass.OwnTypeParameters()[0].Declaration
	var methodParam *symbol.TypeParamDeclaration
	for _, def := range ctx.currentClass.Methods {
		if def.OriginalName == "binder" {
			methodParam = def.TypeParameters[0].Declaration
		}
	}
	if source.nominalScope == nil || classParam == nil || methodParam == nil {
		t.Fatal("actual source/caller declarations missing")
	}
	for _, p := range []*symbol.TypeParamDeclaration{classParam, methodParam} {
		a := inferredJavaTypeOrigin{externalName: "java.util.HashMap", javaType: "java.util.HashMap", arguments: []inferredJavaTypeOrigin{{parameter: p, javaType: p.SourceName}, source}}
		projected, ok := projectJDK21GenericParentOrigin(a, "java.util.Map")
		if !ok || len(projected.arguments) != 2 || projected.arguments[0].parameter != p || projected.arguments[1].nominalScope != source.nominalScope {
			t.Fatal("native declaration substitution recaptured supplied source/binder identity")
		}
		if a.arguments[0].parameter != p || a.arguments[1].nominalScope != source.nominalScope {
			t.Fatal("supplied origin mutated")
		}
	}
	ctx = ordinaryDemandTestContext(t, map[string]string{"p/Use.java": `package p; class HashMap<K,V>{} public class Use<Map>{ <HashMap> void method(HashMap value){} }`}, "p/Use.java")
	ctx.currentClass = ctx.currentFile.FindClassScope("Use")
	origin := func(s string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: s}, ctx)
	}
	formal := origin("java.util.Map<java.lang.String,java.lang.Integer>")
	for _, actual := range []inferredJavaTypeOrigin{origin("HashMap<java.lang.String,java.lang.Integer>"), origin("missing.HashMap<java.lang.String,java.lang.Integer>")} {
		if _, ok := projectJDK21GenericParentOrigin(actual, "java.util.Map"); ok {
			t.Fatal("source/unknown HashMap became canonical JDK declaration")
		}
		if _, _, ok, _ := declaredOriginConversion(actual, formal, ctx); ok {
			t.Fatal("source/unknown HashMap gained native ancestry")
		}
	}
}

func TestJDK21SystematicMetadataConcurrentReads(t *testing.T) {
	ctx := rawReferenceTestContext(t)
	actual := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.util.HashMap<java.lang.String,p.Value>"}, ctx)
	source := actual.arguments[1].nominalScope
	if source == nil {
		t.Fatal("actual supplied source identity absent")
	}
	done := make(chan bool, 32)
	for worker := 0; worker < 32; worker++ {
		go func() {
			valid := true
			for iteration := 0; iteration < 16; iteration++ {
				p, ok := projectJDK21GenericParentOrigin(actual, "java.util.Map")
				valid = valid && ok && len(p.arguments) == 2 && p.arguments[1].nominalScope == source
			}
			done <- valid
		}()
	}
	for worker := 0; worker < 32; worker++ {
		if !<-done {
			t.Error("concurrent immutable declaration projection changed")
		}
	}
	if actual.arguments[1].nominalScope != source {
		t.Fatal("supplied source origin mutated")
	}
}
