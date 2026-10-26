package transpiler

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestJavaDollarIdentifiersKeepEscapedSpellingBindingsDistinct(t *testing.T) {
	out := renderGoFileFromJava(t, `
class A$B {
 int value$;
 A$B(int value$){this.value$=value$;}
 int read$(){return value$;}
 static int make$(){return 13;}
}
class java2goIdentifier_612442 {
 int value$;
 java2goIdentifier_612442(int value$){this.value$=value$;}
 int read$(){return value$;}
 static int make$(){return 17;}
}
public class DollarBindings {
 public static int run(){
  int $value=7;
  int java2goIdentifier_2476616c7565=11;
  A$B first=new A$B(2);
  java2goIdentifier_612442 second=new java2goIdentifier_612442(3);
  return first.read$()+second.read$()+A$B.make$()+java2goIdentifier_612442.make$()+$value+java2goIdentifier_2476616c7565;
 }
}
`)
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, parser.AllErrors); err != nil {
		t.Fatalf("legal Java dollar bindings emitted invalid Go: %v", err)
	}
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestDistinctDollarBindings(t *testing.T){if got:=Run();got!=53{t.Fatalf("Run()=%d, want 53",got)}}
`)
}

func TestJavaDollarIdentifiersNestedConstructorsStaticsAndTypeParameters(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class Dollar$Owner<T$> {
 T$ item$;
 public Dollar$Owner(T$ item$){this.item$=item$;}
 public T$ read$(){return item$;}
 public static class Nested$ {
  public int value$;
  public Nested$(int value$){this.value$=value$;}
  public int read$(){return value$;}
 }
 public static int field$=5;
 public static int method$(int value$){return value$+field$;}
 public static int run(){
  Dollar$Owner<Integer> owner$=new Dollar$Owner<Integer>(7);
  Nested$ nested$=new Nested$(3);
  java.util.function.Supplier<Nested$> factory$=()->new Nested$(11);
  return owner$.read$()+nested$.read$()+method$(2)+factory$.get().read$();
 }
}
`)
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, parser.AllErrors); err != nil {
		t.Fatalf("nested dollar constructors emitted invalid Go: %v", err)
	}
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestDollarNestedAndGenericBindings(t *testing.T){if got:=Run();got!=28{t.Fatalf("Run()=%d, want 28",got)}}
`)
}

func TestJavaDollarIdentifiersRetainJavaReflectionNames(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class Dollar$Reflection {
 public int field$=9;
 public int method$(){return field$;}
 public static int run() throws Exception {
  Dollar$Reflection value$=new Dollar$Reflection();
  int first=((Integer)Dollar$Reflection.class.getField("field$").get(value$)).intValue();
  int second=((Integer)Dollar$Reflection.class.getMethod("method$").invoke(value$)).intValue();
  return first+second;
 }
}
`)
	for _, javaMetadata := range []string{`stdjava.TypeID("Dollar$Reflection")`, `SimpleName: "Dollar$Reflection"`, `Name: "field$"`, `Name: "method$"`} {
		if !strings.Contains(out, javaMetadata) {
			t.Fatalf("Java identity metadata lost %q:\n%s", javaMetadata, out)
		}
	}
	if strings.Contains(out, `GoName: "Field$"`) || strings.Contains(out, `GoName: "Method$Java2goExecution"`) {
		t.Fatalf("reflective selectors retained illegal Go names:\n%s", out)
	}
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestDollarReflectiveSelectors(t *testing.T){if got:=Run();got!=18{t.Fatalf("Run()=%d, want 18",got)}}
`)
}

func TestJavaDollarIdentifiersPublicLeadingDollarAcrossPackagesJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>visibility</artifactId><version>1</version></project>`,
		"src/main/java/names/$Owner.java": `package names;
public class $Owner {
 public int $field;
 public $Owner(int $field){this.$field=$field;}
 public int $read(){return $field;}
 public static int $static(){return 5;}
 public static class $Nested {public static int $value(){return 7;}}
}`,
		"src/main/java/app/$Entry.java": `package app;
import names.$Owner;
public class $Entry {public static void main(String[] args){
 $Owner $value=new $Owner(3);
 System.out.println($value.$read()+$Owner.$static()+$Owner.$Nested.$value());
}}`,
	}, "app.$Entry", "15\n")
}
