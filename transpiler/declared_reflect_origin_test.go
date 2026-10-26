package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestDeclaredReflectOriginalChainOrigins(t *testing.T) {
	bytes, err := os.ReadFile(filepath.Join("testdata", "canonical_class_physical129", "CanonicalClassReflectionOracle", "CanonicalClassReflectionOracle.java"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"CanonicalClassReflectionOracle.java": string(bytes)})
	owner := findQualifiedSourceClass("classproof.CanonicalClassReflectionOracle")
	if owner == nil {
		t.Fatal("original owner missing")
	}
	ctx = classScopeCtx(owner, ctx)
	ctx.localScope = owner.FindMethod().ByOriginalName("main")[0]
	fields := owner.FindMethod().ByOriginalName("field")
	if len(fields) != 1 {
		t.Fatal("original field helper missing")
	}
	method := fields[0]
	formals := methodFormalOrigins(method, owner, nil, ctx)
	for i, formal := range formals {
		t.Logf("formal%d declared=%s external=%q unresolved=%t args=%d owner=%p", i, method.Parameters[i].OriginalType, formal.externalName, formal.unresolved, len(formal.arguments), formal.declaringOwner)
	}
	if len(formals) != 2 || formals[0].externalName != "java.lang.Class" || formals[0].unresolved || formals[1].externalName != "java.lang.String" || formals[1].unresolved {
		t.Error("original helper formals lost declaration certificates")
	}
	var sites []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" && node.ChildByFieldName("name").Content(bytes) == "getActualTypeArguments" {
			recv := node.ChildByFieldName("object")
			if recv != nil && recv.Type() == "method_invocation" && recv.ChildByFieldName("name").Content(bytes) == "field" {
				sites = append(sites, node)
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if len(sites) != 4 {
		t.Fatalf("original chainedsites=%d", len(sites))
	}
	for _, site := range sites {
		recv := site.ChildByFieldName("object")
		args := recv.ChildByFieldName("arguments")
		selected := findBestMethodInHierarchy(owner, "field", args, false, true, ctx, bytes)
		t.Logf("site%d selected=%t", site.StartByte(), selected != nil)
		for i, arg := range nodeutil.SemanticNamedChildrenOf(args) {
			var origin inferredJavaTypeOrigin
			typ, known := inferExprJavaType(arg, ctx, bytes, &origin)
			t.Logf("site%d arg%d type=%q known=%t external=%q unresolved=%t", site.StartByte(), i, typ, known, origin.externalName, origin.unresolved)
		}
		var origin inferredJavaTypeOrigin
		typ, known := inferUserMethodReturnType(recv, ctx, bytes, &origin)
		t.Logf("site%d result=%q known=%t external=%q unresolved=%t owner=%p", site.StartByte(), typ, known, origin.externalName, origin.unresolved, origin.declaringOwner)
		if selected == nil || selected.def != method || !known || origin.externalName != "java.lang.reflect.ParameterizedType" || origin.unresolved || origin.declaringOwner != owner {
			t.Errorf("site%d exact original result declaration lost", site.StartByte())
		}
		receiver, eligible := intrinsicMethodReceiverTypeName(recv, "getActualTypeArguments", ctx, bytes)
		result, admitted := inferIntrinsicMethodResultType(site, ctx, bytes)
		t.Logf("site%d receiver=%q/%t accessorResult=%q/%t", site.StartByte(), receiver, eligible, result, admitted)
		if !eligible || receiver != "ParameterizedType" || !admitted || result != "java.lang.reflect.Type[]" {
			t.Errorf("site%d canonical accessor not selected", site.StartByte())
		}
	}
}

func TestDeclaredReflectDemandNativeOrigins(t *testing.T) {
	for _, header := range []string{"import java.lang.reflect.*;", "import java.lang.reflect.ParameterizedType;", ""} {
		t.Run(header, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": header + `class Probe{}`}, "Probe.java")
			for _, native := range []string{"Type", "ParameterizedType", "GenericArrayType", "WildcardType", "TypeVariable", "GenericDeclaration", "AnnotatedElement"} {
				written := "java.lang.reflect." + native
				if strings.Contains(header, ".*") || native == "ParameterizedType" && header != "" {
					written = native
				}
				if got := declaredExternalNominalOwner(written, nil, ctx); got != "java.lang.reflect."+native {
					t.Errorf("declared %s owner=%q", written, got)
				}
			}
			for _, typ := range []string{"Class", "String"} {
				if got := declaredExternalNominalOwner(typ, nil, ctx); got != "java.lang."+typ {
					t.Errorf("implicit %s owner=%q", typ, got)
				}
			}
			if !declaredInterfaceDemandPrefixAbsent("java", ctx) {
				t.Error("complete reflect package cannot prove absent java prefix")
			}
		})
	}
}

func TestDeclaredReflectNamespaceShields(t *testing.T) {
	for name, java := range map[string]string{
		"source":         `import java.lang.reflect.*;class ParameterizedType{}class Probe{}`,
		"binder":         `import java.lang.reflect.*;class Probe<ParameterizedType>{}`,
		"foreign":        `import foreign.ParameterizedType;class Probe{}`,
		"unknown-demand": `import foreign.*;import java.lang.reflect.*;class Probe{}`,
		"source-prefix":  `import java.lang.reflect.*;class Probe{static class java{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": java}, "Probe.java")
			written := "ParameterizedType"
			if name == "source-prefix" {
				written = "java.lang.reflect.ParameterizedType"
			}
			bindings := map[string]*symbol.TypeParamDeclaration{}
			for _, p := range ctx.currentClass.TypeParameters {
				bindings[p.Name] = p.Declaration
			}
			if got := declaredExternalNominalOwner(written, bindings, ctx); got != "" {
				t.Fatalf("source/binder/import/unknown type became native %q", got)
			}
		})
	}
	ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": `import java.lang.reflect.*;class Probe{}`}, "Probe.java")
	for _, unsupported := range []string{"InvocationHandler", "Proxy", "Array", "AnnotatedType", "RecordComponent"} {
		if got := declaredExternalNominalOwner(unsupported, nil, ctx); got != "" {
			t.Errorf("presence catalog invented native API identity %q", got)
		}
	}
}

func TestDeclaredReflectDonorReturnContext(t *testing.T) {
	files := map[string]string{
		"p/Donor.java": `package p;import java.lang.reflect.*;public class Donor{public static ParameterizedType value(Class<?> owner,String name){return null;}}`,
		"q/Probe.java": `package q;class Probe{static class ParameterizedType{}static class Type{}static class Class{}static class String{}static class java{} void run(){p.Donor.value(Object.class,"x").getActualTypeArguments();}}`,
	}
	ctx := ordinaryDemandTestContext(t, files, "q/Probe.java")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
	call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	if call == nil {
		t.Fatal("missing original donor invocation")
	}
	recv := call.ChildByFieldName("object")
	if recv == nil || recv.Type() != "method_invocation" {
		t.Fatal("missing chained donor receiver")
	}
	var origin inferredJavaTypeOrigin
	typ, known := inferExprJavaType(recv, ctx, ctx.currentFile.Source, &origin)
	t.Logf("actual donor return %s/%t external=%q unresolved=%t", typ, known, origin.externalName, origin.unresolved)
	if !known || origin.unresolved || origin.externalName != "java.lang.reflect.ParameterizedType" || origin.declaringOwner == ctx.currentClass {
		t.Fatal("caller shadow recaptured donor return")
	}
}
