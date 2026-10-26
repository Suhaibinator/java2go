package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMapReferenceProjectionPhysicalAndConsumer(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.util.Map;import java.util.HashMap;import java.util.concurrent.ConcurrentMap;import java.util.concurrent.ConcurrentHashMap;
class Carrier<K,V>{Map<K,V> values;ConcurrentMap<K,V> concurrent;Carrier(){values=new HashMap<>();concurrent=new ConcurrentHashMap<>();} V read(K key){return values.get(key);} Object erased(K key){return values.get(key);} V atomic(K key,V value){return concurrent.putIfAbsent(key,value);} }`)
	for _, want := range []string{"values stdjava.JavaMap", "concurrent stdjava.JavaConcurrentMap", "stdjava.NewConcurrentHashMap[K, V]", "stdjava.MapGetExecution", "stdjava.MapPutIfAbsentExecution"} {
		if !strings.Contains(strings.Join(strings.Fields(out), " "), want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, ".get(") || strings.Contains(out, ".putIfAbsent(") {
		t.Errorf("unlowered external map invocation:\n%s", out)
	}
}
func TestMapReferenceProjectionNominalControls(t *testing.T) {
	out := renderGoFileFromJava(t, `class Map<K,V>{V get(K key){return null;}} class ConcurrentMap<K,V>{V get(K key){return null;}} class ConcurrentHashMap<K,V>{} class Control{Map<String,String> local;ConcurrentMap<String,String> parallel;ConcurrentHashMap<String,String> concrete;String read(String key){return local.get(key);}}`)
	if strings.Contains(out, "stdjava.JavaMap") || strings.Contains(out, "stdjava.JavaConcurrentMap") || strings.Contains(out, "stdjava.MapGetExecution") {
		t.Fatal("source names borrowed canonical map representation")
	}
}
func TestMapReferenceProjectionFullConcurrentControlJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/map_reference_projection/ConcurrentMapControl.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>campaign</groupId><artifactId>concurrentmap</artifactId><version>1</version></project>`, "src/main/java/campaign/concurrentmap/ConcurrentMapControl.java": string(source)}, "campaign.concurrentmap.ConcurrentMapControl", "basic-erasure-source=true\nputAll-order-prefix=true\nlive-views-entry=true\ncallbacks-atomic=true\n")
}

func TestMapReferenceProjectionSourceAndPollutionJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/map_reference_projection/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>projection</groupId><artifactId>map</artifactId><version>1</version></project>`, "src/main/java/projection/Main.java": string(source)}, "projection.Main", "first:true:true:true:true:replacement:true:true:new-value:copy:true:true:true:true:true\n")
}
func TestMapReferenceProjectionArrayAndBinderControls(t *testing.T) {
	helper := setupParseHelper(t, `class Map<K,V>{} class Control<Map>{<ConcurrentMap> ConcurrentMap keep(ConcurrentMap value){return value;}}`)
	if expression := collectionTypeExpr("Map", []string{"String", "String"}, inScopeTypeParameters(helper.Ctx), helper.Ctx); expression != nil {
		t.Fatal("source Map acquired external representation")
	}
	out := renderGoFileFromJava(t, `class ArrayControl{java.util.Map<String,String>[] values=new java.util.Map[1];java.util.concurrent.ConcurrentMap<String,String>[] parallel=new java.util.concurrent.ConcurrentMap[1];}`)
	if strings.Count(out, "stdjava.ReferenceArray") < 2 || !strings.Contains(out, `"java.util.Map"`) || !strings.Contains(out, `"java.util.concurrent.ConcurrentMap"`) {
		t.Errorf("array physical or nominal descriptor lost:\n%s", out)
	}
}

func TestMapReferenceProjectionVoidLambdaConsumer(t *testing.T) {
	out := renderGoFileFromJava(t, `class Probe{java.util.Map<String,String> value;java.lang.Runnable body(){return ()->value.get("key");}}`)
	if strings.Contains(out, "*void") || strings.Contains(out, `TypeID("void")`) {
		t.Fatalf("discarded map value acquired void checkcast:\n%s", out)
	}
}

func TestMapReferenceProjectionCoreBoundariesJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/map_reference_projection/Boundary.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>boundary</groupId><artifactId>map</artifactId><version>1</version></project>`, "src/main/java/boundary/Main.java": string(source)}, "boundary.Main", "true:true:true:replacement\ntrue:absent-result:copied:true:true:true\ntrue:true:true:true:true\n")
}

func TestMapReferenceProjectionOriginalComputeJDK21(t *testing.T) {
	bytes, err := os.ReadFile("campaign_runtime_mapcompute_test.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "campaign_runtime_mapcompute_test.go", bytes, 0)
	if err != nil {
		t.Fatal(err)
	}
	var source string
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if name.Name == "source" {
				literal, ok := spec.Values[i].(*ast.BasicLit)
				if !ok {
					t.Fatal("original source literal changed")
				}
				source, err = strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		return true
	})
	if source == "" {
		t.Fatal("missing original complete source")
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>original</groupId><artifactId>compute</artifactId><version>1</version></project>`, "src/main/java/CampaignRuntimeMapComputeIfAbsent.java": source, "src/main/java/Driver.java": `public class Driver{public static void main(String[] args){System.out.println(CampaignRuntimeMapComputeIfAbsent.run());}}`}, "Driver", "kept:nil-value:2:true:true:true:effect:false:true:effect:false:true:callback|kept:nil-value:2:true:true:true:effect:false:true:effect:false:true:null|true:0\n")
}

func TestMapReferenceProjectionDeclarationArguments(t *testing.T) {
	helper := setupParseHelper(t, `import java.util.Map;import java.util.concurrent.ConcurrentMap;abstract class Parent<K> implements Map<K,String>{} abstract class Child<X> extends Parent<X>{} class Bound<T extends ConcurrentMap<String,Integer>>{}`)
	args, ok := mapReferenceArguments("Child<Integer>", "java.util.Map", helper.Ctx)
	if !ok || len(args) != 2 || args[0] != "java.lang.Integer" || args[1] != "java.lang.String" {
		t.Fatalf("declared generic ancestry = %v,%v", args, ok)
	}
	scope := helper.File.Symbols.FindClassScope("Bound")
	ctx := classScopeCtx(scope, helper.Ctx)
	args, ok = mapReferenceArguments("T", "java.util.Map", ctx)
	if !ok || len(args) != 2 || args[0] != "java.lang.String" || args[1] != "java.lang.Integer" {
		t.Fatalf("bound ancestry = %v,%v", args, ok)
	}
	if _, ok := mapReferenceArguments("T[]", "java.util.Map", ctx); ok {
		t.Fatal("array became scalar nominal map")
	}
	if mapReferenceArgumentIndependent("java.util.Map<String,T>[]", ctx) {
		t.Fatal("generic array acquired scalar proof")
	}
}

func assertMapReferenceTypedRead(t *testing.T, element string) {
	t.Helper()
	out := renderGoFileFromJava(t, "class TypedRead{java.util.Map<String,"+element+"> values;"+element+" read(){return values.get(\"key\");}}")
	if !strings.Contains(out, "stdjava.MapGetExecution") || !strings.Contains(out, "stdjava.ObjectView[") {
		t.Fatalf("typed map consumer omitted read projection:\n%s", out)
	}
	helper := setupParseHelper(t, "class TypedRead{}")
	descriptor, known := javaTypeDescriptorExpr(element, helper.Ctx)
	if !known || descriptor == nil {
		t.Fatalf("typed read descriptor absent: %s", element)
	}
}
