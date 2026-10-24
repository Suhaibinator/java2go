package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestClassObjectContract99Registration(t *testing.T) {
	for _, method := range []struct {
		name, helper, result string
		arity                int
	}{
		{"hashCode", "ObjectHashCodeExecution", "int", 0},
		{"equals", "ObjectEqualsExecution", "boolean", 1},
	} {
		t.Run(method.name, func(t *testing.T) {
			gen := instanceIntrinsics[intrinsicKey{"Class", method.name}]
			if gen == nil {
				t.Fatal("canonical Class inherited Object registration absent")
			}
			if actual := instanceIntrinsicResultTypes[intrinsicKey{"Class", method.name}]; actual != method.result {
				t.Fatalf("Java result=%q want %q", actual, method.result)
			}
			helper := setupParseHelper(t, `class Probe {}`)
			for _, execution := range []string{"", "currentExecution99"} {
				ctx := helper.Ctx.Clone()
				ctx.executionContextName = execution
				args := []ast.Expr{}
				if method.arity == 1 {
					args = append(args, ast.NewIdent("other99"))
				}
				expr := gen(ast.NewIdent("receiver99"), args, ctx)
				var buf bytes.Buffer
				if err := printer.Fprint(&buf, token.NewFileSet(), expr); err != nil {
					t.Fatal(err)
				}
				wantExecution := execution
				if wantExecution == "" {
					wantExecution = "nil"
				}
				want := "stdjava." + method.helper + "(" + wantExecution + ", receiver99"
				if method.arity == 1 {
					want += ", other99"
				}
				want += ")"
				if buf.String() != want {
					t.Fatalf("current execution/receiver/argument lowering=%q want %q", buf.String(), want)
				}
				if gen(ast.NewIdent("receiver99"), append(args, ast.NewIdent("extra99")), ctx) != nil {
					t.Fatal("wrong arity admitted")
				}
				if method.arity == 1 && gen(ast.NewIdent("receiver99"), nil, ctx) != nil {
					t.Fatal("missing equals argument admitted")
				}
			}
		})
	}
}

func TestClassObjectContract99CanonicalOwner(t *testing.T) {
	for _, control := range []struct {
		name, source, owner string
		known               bool
	}{
		{"implicit", `class Probe {int check(Class<?> c){return c.hashCode();}}`, "Class", true},
		{"qualified-shadow", `class Class {public int hashCode(){return 73;}} class Probe {int check(java.lang.Class<?> c){return c.hashCode();}}`, "Class", true},
		{"top-level-shadow", `class Class {public int hashCode(){return 73;}} class Probe {int check(Class c){return c.hashCode();}}`, "", false},
		{"member-shadow", `class Probe {static class Class {public int hashCode(){return 73;}} int check(Class c){return c.hashCode();}}`, "", false},
		{"local-shadow", `class Probe {int check(){class Class {public int hashCode(){return 73;}} Class c=new Class();return c.hashCode();}}`, "", false},
		{"method-binder-shadow", `class Probe {<Class> int check(Class c){return c.hashCode();}}`, "Object", true},
		{"literal-shadow", `class Probe {static class Class {} int check(){return String.class.hashCode();}}`, "Class", true},
	} {
		t.Run(control.name, func(t *testing.T) {
			helper := setupParseHelper(t, control.source)
			probe := helper.File.Symbols.FindClassScope("Probe")
			if probe == nil {
				t.Fatal("Probe source declaration absent")
			}
			methods := probe.FindMethod().ByOriginalName("check")
			if len(methods) != 1 {
				t.Fatal("check declaration absent")
			}
			ctx := classScopeCtx(probe, helper.Ctx)
			ctx.localScope = methods[0]
			if control.name == "local-shadow" {
				// Initial symbol parsing does not register method-local classes.
				// Mirror the production body parser before querying the receiver.
				declaration := findNode(ctx.localScope.DeclarationNode, "class_declaration")
				if declaration == nil {
					t.Fatal("real local Class declaration absent")
				}
				ctx.className = probe.Class.Name
				ctx.hoistedDecls = &[]ast.Decl{}
				ctx.localClasses = make(map[string]*localClassInfo)
				ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
				ctx.importAliases = make(map[string]string)
				ctx.usedImports = make(map[string]bool)
				ctx.anonClassCounter = new(int)
				hoistLocalClass(declaration, helper.File.Source, ctx)
				registered := ctx.localClasses["Class"]
				if registered == nil || registered.scope == nil || registered.scope.Class.DeclarationNode != declaration {
					t.Fatal("local registration did not preserve exact real AST declaration")
				}
			}
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if call == nil {
				t.Fatal("invocation absent")
			}
			owner, known := intrinsicMethodReceiverTypeName(call.ChildByFieldName("object"), "hashCode", ctx, helper.File.Source)
			if known != control.known || known && owner != control.owner {
				t.Fatalf("owner=%q/%v want %q/%v", owner, known, control.owner, control.known)
			}
		})
	}
}

func TestClassObjectContract99GeneratedCalls(t *testing.T) {
	out := renderGoFileFromJava(t, `class Probe {
 static int hash(java.lang.Class<?> c){return c.hashCode();}
 static boolean same(java.lang.Class<?> c,java.lang.Object o){return c.equals(o);}
 static int fromObject(java.lang.Object o){return o.getClass().hashCode();}
 static int source(){class Class {public int hashCode(){return 73;}}return new Class().hashCode();}
}`)
	if count := strings.Count(out, "stdjava.ObjectHashCodeExecution("); count != 2 {
		t.Fatalf("canonical hash lowering count=%d want2:\n%s", count, out)
	}
	if count := strings.Count(out, "stdjava.ObjectEqualsExecution("); count != 1 {
		t.Fatalf("canonical equals lowering count=%d want1:\n%s", count, out)
	}
	if !strings.Contains(out, "HashCodeJava2goExecution") {
		t.Fatalf("source Class hash override disappeared:\n%s", out)
	}
}

func TestClassObjectContract99EqualsArgumentDeclaration(t *testing.T) {
	for _, shadow := range []bool{false, true} {
		name, source := "implicit", `class Probe {boolean check(java.lang.Class<?> c){return c.equals(42);}}`
		if shadow {
			name = "source-Object-shadow"
			source = `class Object {} class Probe {boolean check(java.lang.Class<?> c){return c.equals(42);}}`
		}
		t.Run(name, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			probe := helper.File.Symbols.FindClassScope("Probe")
			ctx := classScopeCtx(probe, helper.Ctx)
			ctx.localScope = probe.FindMethod().ByOriginalName("check")[0]
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			got := intrinsicInvocationExpectedArgumentTypes(call, call.ChildByFieldName("object"), "", "equals", ctx, helper.File.Source)
			if len(got) != 1 || got[0] != "java.lang.Object" {
				t.Fatalf("Class.equals JDK declared argument=%q want [java.lang.Object]", got)
			}
			out := renderGoFileFromJava(t, source)
			if !strings.Contains(out, "stdjava.BoxInteger(") {
				t.Fatalf("Object argument did not box primitive:\n%s", out)
			}
		})
	}
}
