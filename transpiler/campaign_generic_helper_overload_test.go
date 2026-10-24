package transpiler

import (
	"fmt"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"reflect"
	"strings"
	"testing"
)

// Reduced from the unchanged Gson fromJson overload family. Different Java
// parameter descriptors require distinct generated generic helper types and
// constructors even though the Java method names are identical.
func TestCampaignGenericHelperOverloadJVMParity(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>generic-overloads</artifactId><version>1</version></project>`,
		"src/main/java/p/Overloaded.java": `package p;
public final class Overloaded {
 public static final class Token<T> {}
 @SuppressWarnings("unchecked")
 public <T> T fromJson(String json,Class<T> type){return (T)json;}
 @SuppressWarnings("unchecked")
 public <T> T fromJson(String json,Token<T> type){return (T)json;}
 public static void main(String[]args){int NewOverloadedFromJsonHelper=0;if(NewOverloadedFromJsonHelper!=0){throw new IllegalStateException();}Overloaded codec=new Overloaded();String left=codec.fromJson("a",String.class);String right=codec.fromJson("b",new Token<String>());System.out.println(left+right);}
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "p.Overloaded", "ab\n")
}

func TestCampaignGenericHelperNamesStableAcrossDeclarationOrder(t *testing.T) {
	previous := symbol.GlobalScope
	t.Cleanup(func() { symbol.GlobalScope = previous })
	var baseline map[string]string
	for _, reverse := range []bool{false, true} {
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		methods := []string{`public <T> T choose(T value){return value;}`, `public <T> T choose(int tag,T value){return value;}`}
		if reverse {
			methods[0], methods[1] = methods[1], methods[0]
		}
		sources := []string{
			`package p; class Outer {static class Inner {` + methods[0] + methods[1] + `}}`,
			`package p; class innerChooseHelper {} public class NewinnerChooseHelper1 {}`,
			`package p; class Other {static class Inner {public <T> T choose(T value){return value;}}}`,
		}
		if reverse {
			sources[0], sources[2] = sources[2], sources[0]
		}
		var files []parsing.SourceFile
		for i, source := range sources {
			file := parsing.SourceFile{Name: fmt.Sprintf("Case%d.java", i), Source: []byte(source)}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			file.ParseSymbols()
			symbol.AddSymbolsToPackage(file.Symbols)
			files = append(files, file)
		}
		for _, file := range files {
			ResolveFile(file)
		}
		actual := map[string]string{}
		used := map[string]bool{}
		for _, scope := range allSourceClassScopes() {
			for _, method := range scope.Methods {
				if method.RequiresHelper {
					key := qualifiedSourceClassName(scope) + ":" + interfaceOverloadSignature(scope, method)
					actual[key] = method.HelperName
					packageName := findJavaPackageForClassScope(scope)
					for _, name := range []string{method.HelperName, "New" + method.HelperName} {
						if used[packageName+":"+name] {
							t.Fatalf("duplicate helper declaration %s", name)
						}
						if generatedIdentifierExists(name, scope) {
							t.Fatalf("helper declaration %s collides with source", name)
						}
						used[packageName+":"+name] = true
					}
				}
			}
		}
		if len(actual) != 3 {
			t.Fatalf("helpers %v", actual)
		}
		for _, file := range files {
			ResolveFile(file)
		}
		for _, scope := range allSourceClassScopes() {
			for _, method := range scope.Methods {
				if method.RequiresHelper {
					key := qualifiedSourceClassName(scope) + ":" + interfaceOverloadSignature(scope, method)
					if actual[key] != method.HelperName {
						t.Fatalf("non-idempotent helper %s: %s -> %s", key, actual[key], method.HelperName)
					}
				}
			}
		}
		if baseline == nil {
			baseline = actual
		} else if !reflect.DeepEqual(baseline, actual) {
			t.Fatalf("declaration order changed helpers:\n%v\n%v", baseline, actual)
		}
	}
}

func TestCampaignGenericHelperNestedCycleJVMParity(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>nested-helpers</artifactId><version>1</version></project>`,
		"src/main/java/p/Outer.java": `package p;public final class Outer {
 public static final class Inner {
  public <T> T choose(T value){return value;}
  public <T> T choose(T value,int ignored){return value;}
 }
 public static String suffix(){return q.Bridge.suffix();}
}`,
		"src/main/java/p/OuterInnerChooseHelper.java":     `package p; public class OuterInnerChooseHelper {}`,
		"src/main/java/p/NewOuterInnerChooseHelper1.java": `package p; public class NewOuterInnerChooseHelper1 {}`,
		"src/main/java/q/Bridge.java": `package q; import p.Outer; import p.Outer.Inner; import java.util.function.Function;
public class Bridge {
 public static String suffix(){return "c";}
 public static void main(String[]args){Inner codec=new Inner();Function<String,Object> reference=codec::choose;String first=(String)reference.apply("a");String second=codec.choose("b",1);System.out.println(first+second+Outer.suffix());}
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "q.Bridge", "abc\n")
}

func TestCampaignGenericHelperIdentityDistinguishesBinders(t *testing.T) {
	helper := setupParseHelper(t, `package p; class Holder<T> {
 public <U> T choose(T value,U token){return value;}
 public <T> T choose(T value,int token){return value;}
 public <T extends Number> T bound(T value){return value;}
 public <T extends CharSequence> T bound(T value){return value;}
 }`)
	owner := helper.File.Symbols.FindClassScope("Holder")
	if owner == nil {
		t.Fatal("missing Holder")
	}
	identities := map[string]bool{}
	classBinder, methodBinder := false, false
	for _, method := range owner.Methods {
		if !method.RequiresHelper {
			continue
		}
		key := genericMethodHelperIdentity(owner, method)
		if identities[key] {
			t.Fatalf("distinct declarations have identical helper identity %s", key)
		}
		identities[key] = true
		classBinder = classBinder || strings.Contains(key, "@class:p.Holder:0")
		methodBinder = methodBinder || strings.Contains(key, "@method:0")
	}
	if len(identities) != 4 || !classBinder || !methodBinder {
		t.Fatalf("binder identities lost: %v", identities)
	}
}
