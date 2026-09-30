package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// Exercise both a source enum receiver and the Java result of its synthetic
// factory. Enum.name returns a canonical Java String even when the source enum
// is package private; source overloads and unrelated names retain their bodies.
func TestCampaignCanonicalEnumFinalDispatchLowering(t *testing.T) {
	for _, item := range []struct{ name, source string }{
		{"chained-package-private", `package canonicalenum; enum Choice {ONE} public class Dispatch {static String read(String text){return Choice.valueOf(text).name();}}`},
		{"chained-public", `public enum Choice {ONE; public static String read(String text){return Choice.valueOf(text).name();}}`},
	} {
		t.Run(item.name, func(t *testing.T) {
			generated := renderGoFileFromJava(t, item.source)
			file := canonicalEnumDispatchGoAST(t, generated)
			chained := false
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "EnumNameJavaString" || len(call.Args) != 1 {
					return true
				}
				factory, ok := call.Args[0].(*ast.CallExpr)
				if !ok {
					return true
				}
				function, ok := factory.Fun.(*ast.Ident)
				if ok && strings.Contains(function.Name, "ValueOf") {
					chained = true
				}
				return true
			})
			if !chained {
				t.Fatalf("canonical Enum final dispatch must retain the synthetic factory Java result:\n%s", generated)
			}
		})
	}
	t.Run("selected-inherited-and-source-overloads", func(t *testing.T) {
		generated := renderGoFileFromJava(t, `package canonicalenum;
 enum Choice {ONE,TWO; String name(int ignored){return "source-name-overload";} int compareTo(String ignored){return 37;} public String toString(){return "source-toString-override";}}
 class Named {String name(){return "source-unrelated-name";}}
 public class Dispatch {
  static String inherited(Choice value){return value.name();}
  static int ordinal(Choice value){return value.ordinal();}
  static int order(Choice left,Choice right){return left.compareTo(right);}
  static Class<?> owner(Choice value){return value.getDeclaringClass();}
  static String overload(Choice value){return value.name(1);}
  static int overloadOrder(Choice value){return value.compareTo("other");}
  static String sourceText(Choice value){return value.toString();}
  static String unrelated(Named value){return value.name();}
 }`)
		for _, required := range []string{"stdjava.EnumNameJavaString(value)", "stdjava.EnumOrdinal(value)", "stdjava.EnumCompareTo(left, right)", "stdjava.EnumGetDeclaringClass(value)"} {
			if !strings.Contains(generated, required) {
				t.Errorf("canonical Enum final dispatch missing %s", required)
			}
		}
		file := canonicalEnumDispatchGoAST(t, generated)
		sourceName, sourceOrder, sourceText := false, false, false
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if strings.HasPrefix(strings.ToLower(selector.Sel.Name), "name") && len(call.Args) > 1 {
				sourceName = true
			}
			if strings.HasPrefix(strings.ToLower(selector.Sel.Name), "compareto") && len(call.Args) > 1 {
				sourceOrder = true
			}
			if strings.HasPrefix(selector.Sel.Name, "ToString") {
				sourceText = true
			}
			return true
		})
		if !sourceName || !sourceOrder || !sourceText {
			t.Errorf("source enum overload/override declarations must remain callable: name=%t compareTo=%t toString=%t\n%s", sourceName, sourceOrder, sourceText, generated)
		}
	})
}

func canonicalEnumDispatchGoAST(t *testing.T, generated string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if err != nil {
		t.Fatalf("parse generated enum dispatch Go: %v\n%s", err, generated)
	}
	return file
}
