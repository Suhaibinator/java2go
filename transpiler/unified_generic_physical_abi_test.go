package transpiler

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"
)

func runUnifiedGenericABIOriginal(t *testing.T, fixture, mainClass string) {
	t.Helper()
	root := filepath.Join("testdata", "unified_generic_abi", fixture)
	java, err := os.ReadFile(filepath.Join(root, "Main.java"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	javaPath := "src/main/java/Main.java"
	if mainClass == "defaultorigin.Main" {
		javaPath = "src/main/java/defaultorigin/Main.java"
	}
	if mainClass == "argsnapshot.Main" {
		javaPath = "src/main/java/argsnapshot/Main.java"
	}
	if mainClass == "arglate.Main" {
		javaPath = "src/main/java/arglate/Main.java"
	}
	if mainClass == "argabi.Main" {
		javaPath = "src/main/java/argabi/Main.java"
	}
	if mainClass == "edge.Main" {
		javaPath = "src/main/java/edge/Main.java"
	}
	if mainClass == "rawret.Main" {
		javaPath = "src/main/java/rawret/Main.java"
	}
	if mainClass == "rawfn.Main" {
		javaPath = "src/main/java/rawfn/Main.java"
	}
	if mainClass == "staticcast.Main" {
		javaPath = "src/main/java/staticcast/Main.java"
	}
	if mainClass == "staticabi.Main" {
		javaPath = "src/main/java/staticabi/Main.java"
	}
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>unified-generic-abi</artifactId><version>1</version></project>", javaPath: string(java)}
	observations := []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: string(expected)},
		{name: "repeat2", stdout: string(expected)},
		{name: "repeat3", stdout: string(expected)},
	}
	runCampaignCompilerStrictProjectObservations(t, files, mainClass, observations)
}
func TestUnifiedGenericABILateCastJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "latecast", "edge.Main")
}
func TestUnifiedGenericABICastTimingJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "cast-timing", "Main")
}
func TestUnifiedGenericABIQualifierExecutionMethodRefJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "qualifier-execution-methodref", "Main")
}
func TestUnifiedGenericABIBoundProvenanceNominalJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "bound-provenance-and-nominal", "staticabi.Main")
}

func TestUnifiedGenericABIPhysicalSlotDeclarationIdentity(t *testing.T) {
	h := setupParseHelper(t, `class Host<T>{<T,T2 extends T>T2 identity(T2 value){return value;}}`)
	owner := h.File.Symbols.FindClassScope("Host")
	method := owner.FindMethodByName("identity", nil)
	slots, _, ok := declaredGenericMethodPhysicalSlots(method)
	if !ok || len(slots) != 2 {
		t.Fatalf("missing proven slots: %#v %v", slots, ok)
	}
	for index, slot := range slots {
		if !slot.implicitObject || slot.javaType != "java.lang.Object" {
			t.Fatalf("slot%d followed a spelling instead of the declaration: %#v", index, slot)
		}
	}
}

func TestUnifiedGenericABICastBoundaryEffectsJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "cast-boundary-effects", "staticcast.Main")
}

func TestUnifiedGenericABIPhysicalCastAdmissionGuards(t *testing.T) {
	h := setupParseHelper(t, `class T2{} class Base{} class Host<T>{static<U extends Base>U admitted(java.lang.Object value){return (U)value;} <U>U instance(java.lang.Object value){return (U)value;} private static<U>U hidden(java.lang.Object value){return (U)value;} static<U extends Base & Runnable>U intersection(java.lang.Object value){return (U)value;}}`)
	owner := h.File.Symbols.FindClassScope("Host")
	for _, name := range []string{"admitted", "instance"} {
		def := owner.FindMethodByName(name, nil)
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = def
		if genericMethodPhysicalScalarCast(ast.NewIdent("value"), "U", ctx) == nil {
			t.Fatalf("proven %s scalar rejected", name)
		}
		for _, target := range []string{"T", "U[]", "T2", "Base"} {
			if genericMethodPhysicalScalarCast(ast.NewIdent("value"), target, ctx) != nil {
				t.Fatalf("%s acquired method-slot route", target)
			}
		}
		copied := *def
		ctx.localScope = &copied
		if genericMethodPhysicalScalarCast(ast.NewIdent("value"), "U", ctx) != nil {
			t.Fatal("copied declaration acquired physical plan")
		}
	}
	for _, name := range []string{"hidden", "intersection"} {
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = owner.FindMethodByName(name, nil)
		if genericMethodPhysicalScalarCast(ast.NewIdent("value"), "U", ctx) != nil {
			t.Fatalf("unsupported %s admitted", name)
		}
	}
}

func TestUnifiedGenericABIMethodRefConversionJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "methodref-conversion", "Main")
}

func TestUnifiedGenericABIRawFunctionEffectsJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "raw-function-effects", "Main")
}

func TestUnifiedGenericABIRawFunctionPackagedObjectJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "raw-function-packaged-object", "rawfn.Main")
}

func TestUnifiedGenericABIRawFunctionGenericReturnsJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "raw-function-generic-returns", "rawret.Main")
}
func TestUnifiedGenericABIOriginalFullMapJDK21(t *testing.T) {
	java, err := os.ReadFile(filepath.Join("testdata", "map_reference_projection", "Main.java"))
	if err != nil {
		t.Fatal(err)
	}
	want := "first:true:true:true:true:replacement:true:true:new-value:copy:true:true:true:true:true\n"
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>projection</groupId><artifactId>map</artifactId><version>1</version></project>", "src/main/java/projection/Main.java": string(java)}, "projection.Main", []campaignCompilerProjectObservation{{name: "repeat1", stdout: want}, {name: "repeat2", stdout: want}, {name: "repeat3", stdout: want}})
}

func TestUnifiedGenericABIImplicitObjectArgumentsJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "implicit-object-arguments", "argabi.Main")
}

func TestUnifiedGenericABIImplicitObjectDelayedArgumentsJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "implicit-object-delayed-arguments", "arglate.Main")
}

func TestUnifiedGenericABIImplicitArgumentLifetimeAndIdentity(t *testing.T) {
	h := setupParseHelper(t, `class Object{}class Host<C>{static<U>U choose(U first,int later){return first;}static<V>V read(){return null;}static<V extends Object>V written(){return null;}static<V>V[] array(){return null;}static Object nominal(){return null;}void body(){choose(read(),1);}}`)
	owner := h.File.Symbols.FindClassScope("Host")
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = owner.FindMethodByName("body", nil)
	invocation := findNode(h.File.Ast, "method_invocation")
	root := invocation.ChildByFieldName("arguments").NamedChild(0)
	choose := owner.FindMethodByName("choose", nil)
	entry := planStaticGenericMethodPhysicalABI(owner, choose, ctx)
	if entry == nil {
		t.Fatal("missing admitted choose")
	}
	origin := entry.implicitObjectArgument(0, root, ctx)
	if origin == nil {
		t.Fatal("missing declaration-owned argument")
	}
	ctx.genericImplicitArgument = origin
	read := owner.FindMethodByName("read", nil)
	if !ctx.genericImplicitArgument.defaultResult(read, nil, root, ctx.Clone()) {
		t.Fatal("same-root clone lost canonical default")
	}
	for _, binding := range []string{"String", "Object", "java.lang.Object"} {
		if origin.defaultResult(read, map[string]string{"V": binding}, root, ctx) {
			t.Fatalf("written/inferred %s lost checkcast", binding)
		}
	}
	for _, name := range []string{"written", "array", "nominal"} {
		if origin.defaultResult(owner.FindMethodByName(name, nil), nil, root, ctx) {
			t.Fatalf("unrelated %s acquired default route", name)
		}
	}
	copied := *read
	if origin.defaultResult(&copied, nil, root, ctx) {
		t.Fatal("copied declaration acquired default route")
	}
	copiedScope := *ctx.localScope
	sibling := ctx.Clone()
	sibling.localScope = &copiedScope
	if origin.defaultResult(read, nil, root, sibling) {
		t.Fatal("copied caller scope acquired argument origin")
	}
	sibling = ctx.Clone()
	sibling.currentClass = h.File.Symbols.FindClassScope("Object")
	if origin.defaultResult(read, nil, root, sibling) {
		t.Fatal("sibling class acquired argument origin")
	}
	sibling = ctx.Clone()
	sibling.currentFile = setupParseHelper(t, "class Other{}").Ctx.currentFile
	if origin.defaultResult(read, nil, root, sibling) {
		t.Fatal("foreign file acquired argument origin")
	}
	sibling = ctx.Clone()
	sibling.expectedType = "String"
	sibling.expectedTypeRoot = root
	if origin.defaultResult(read, nil, root, sibling) {
		t.Fatal("explicit consuming target lost checkcast")
	}
	if origin.defaultResult(read, nil, invocation, ctx) {
		t.Fatal("outer invocation acquired nested argument origin")
	}
	if entry.implicitObjectArgument(1, root, ctx) != nil {
		t.Fatal("primitive formal acquired implicit Object origin")
	}
}

func TestUnifiedGenericABIImplicitArgumentSnapshotsJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "implicit-argument-snapshots", "argsnapshot.Main")
}

func TestUnifiedGenericABIDefaultOriginStandaloneJDK21(t *testing.T) {
	runUnifiedGenericABIOriginal(t, "standalone-default-origin", "defaultorigin.Main")
}

func TestUnifiedGenericABIImplicitDefaultResultAdmission(t *testing.T) {
	h := setupParseHelper(t, `class Object{}class java{static class lang{static class Object{}}}class Host<C>{static<T>T plain(){return null;}static<T,U extends T>U linked(T value){return null;}static<T extends Object>T written(){return null;}static<T>T[] arrays(){return null;}private static<T>T hidden(){return null;}<T>C classReturn(T value){return null;}static<T>Object nominal(){return null;}}`)
	owner := h.File.Symbols.FindClassScope("Host")
	ctx := classScopeCtx(owner, h.Ctx)
	plain := owner.FindMethodByName("plain", nil)
	for _, target := range []string{"", "var"} {
		if !genericMethodImplicitDefaultResult(plain, nil, target, ctx) {
			t.Fatalf("canonical default rejected for %q", target)
		}
	}
	for _, target := range []string{"String", "Object", "java.lang.Object", "C"} {
		if genericMethodImplicitDefaultResult(plain, nil, target, ctx) {
			t.Fatalf("written target %q acquired default projection", target)
		}
	}
	for _, binding := range []string{"String", "Object", "java.lang.Object"} {
		if genericMethodImplicitDefaultResult(plain, map[string]string{"T": binding}, "", ctx) {
			t.Fatalf("bound %q lost checkcast", binding)
		}
	}
	linked := owner.FindMethodByName("linked", nil)
	if !genericMethodImplicitDefaultResult(linked, nil, "", ctx) {
		t.Fatal("transitive implicit Object rejected")
	}
	if genericMethodImplicitDefaultResult(linked, map[string]string{"T": "String"}, "", ctx) {
		t.Fatal("inferred method bound lost checkcast")
	}
	for _, name := range []string{"written", "arrays", "hidden", "classReturn", "nominal"} {
		if genericMethodImplicitDefaultResult(owner.FindMethodByName(name, nil), nil, "", ctx) {
			t.Fatalf("unrelated/refused %s acquired default projection", name)
		}
	}
	copied := *plain
	if genericMethodImplicitDefaultResult(&copied, nil, "", ctx) {
		t.Fatal("copied declaration acquired default projection")
	}
}
