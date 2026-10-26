package transpiler

import (
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestJavaDollarEnumNamespaceCollisionsProjectJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                 `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>namespace</artifactId><version>1</version></project>`,
		"src/main/java/p/E.java":  `package p; public enum E {$A,small$,Small$,java2goIdentifier_2441;}`,
		"src/main/java/p/F.java":  `package p; public enum F {$A;}`,
		"src/main/java/p/C.java":  `package p; public class C {public static int $A=3; public static int small$(){return 5;}}`,
		"src/main/java/p/$A.java": `package p; public class $A {public int read(){return 7;}}`,
		"src/main/java/app/Main.java": `package app;
import p.E; import p.F; import p.C; import p.$A; import static p.E.$A;
public class Main {public static void main(String[] args) throws Exception {
 System.out.println(E.$A.name()+":"+F.$A.name()+":"+C.$A+":"+C.small$()+":"+new $A().read()+":"+
  (E.values()[0]==$A)+":"+(E.valueOf("$A")==E.$A)+":"+
  (E.class.getField("$A").get(null)==E.$A)+":"+
  (F.class.getField("$A").get(null)==F.$A)+":"+
  (E.valueOf("java2goIdentifier_2441")==E.java2goIdentifier_2441));
}}`,
	}, "app.Main", "$A:$A:3:5:7:true:true:true:true:true\n")
}

// Check the compiler binding contract separately from enum runtime construction:
// every reference consumes the same allocated implicit field, with no duplicate
// package declarations and no changes to Java reflection/lookup identity.
func TestJavaDollarEnumAllocatedFieldBindingIdentity(t *testing.T) {
	helper := setupParseHelper(t, `
enum E {$A;}
enum F {$A;}
class C {public static int $A=3; public static int small$(){return 5;}}
public class Main {public static Object read(){return E.$A;}}
`)
	file, ok := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx).(*ast.File)
	if !ok {
		t.Fatal("compiler did not produce a Go file")
	}
	bindings := map[string]bool{}
	for _, declaration := range file.Decls {
		var names []*ast.Ident
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if value.Recv == nil && value.Name.Name != "init" {
				names = append(names, value.Name)
			}
		case *ast.GenDecl:
			for _, specification := range value.Specs {
				switch value := specification.(type) {
				case *ast.ValueSpec:
					names = append(names, value.Names...)
				case *ast.TypeSpec:
					names = append(names, value.Name)
				}
			}
		}
		for _, name := range names {
			if bindings[name.Name] {
				t.Fatalf("duplicate generated package binding %q", name.Name)
			}
			bindings[name.Name] = true
		}
	}
	for _, scope := range helper.File.Symbols.TopLevelClasses {
		for _, constant := range scope.EnumConstants {
			if constant.Field == nil || constant.Field.OriginalName != constant.Name || constant.Name != "$A" {
				t.Fatalf("Java constant identity changed in %s", scope.Class.OriginalName)
			}
			if constant.EmittedName() != constant.Field.Name || !bindings[symbol.GoIdentifier(constant.Field.Name)] {
				t.Fatalf("enum %s does not consume its allocated implicit field", scope.Class.OriginalName)
			}
		}
	}
}

func TestJavaDollarCaseDistinctMethodsAndVirtualOverridesProjectJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                     `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>caseidentity</artifactId><version>1</version></project>`,
		"src/main/java/p/Choice.java": `package p; public interface Choice {int small$(); int Small$();}`,
		"src/main/java/p/Base.java":   `package p; public class Base implements Choice {public int small$(){return 2;} public int Small$(){return 3;}}`,
		"src/main/java/p/Child.java":  `package p; public class Child extends Base {public int small$(){return 11;} public int Small$(){return 13;}}`,
		"src/main/java/p/Local.java": `package p; public class Local {
 private int small$=5; private int Small$=7;
 private int small$(){return small$;} private int Small$(){return Small$;}
 public int read(){return small$()+Small$();}
}`,
		"src/main/java/app/Main.java": `package app; import p.*;
public class Main {public static void main(String[] args){
 Choice first=new Child(); Base second=new Child(); Child third=new Child();
 System.out.println((first.small$()+first.Small$())+":"+(second.small$()+second.Small$())+":"+
  (third.small$()+third.Small$())+":"+new Local().read());
}}`,
	}, "app.Main", "24:24:24:12\n")
}
