package transpiler

import (
	"go/ast"
	"testing"
)

func TestGeneratedInstanceIdentifierStrictJVM(t *testing.T) {
	for _, test := range []struct{ name, source, want string }{
		{"privateNominalClone", `package instance;class Root {int inherited=4;}class A1 extends Root implements Cloneable {int own=7;Object shared=new Object();A1 copy()throws CloneNotSupportedException{return (A1)super.clone();}}public class Main {public static void main(String[]args)throws Exception{A1 first=new A1();A1 second=first.copy();second.own=9;System.out.println((first!=second)+":"+(first.shared==second.shared)+":"+first.inherited+":"+first.own+":"+second.own);}}`, "true:true:4:7:9\n"},
		{"threadReceiver", `package instance;class A1 extends Thread {int ran;public void run(){ran=7;}}public class Main{public static void main(String[]args)throws Exception{A1 value=new A1();value.start();value.join();System.out.println(value.ran);}}`, "7\n"},
		{"capturedLocalOrder", `package instance;class Root{int base;Root(){base=3;}}public class Main{static int events;static int mark(int digit,int value){events=10*events+digit;return value;}static int work(int a1){class A1 extends Root{int own=mark(2,a1);int sum(){return base+own;}}events=0;A1 first=new A1();A1 second=new A1();System.out.println(events);return first.sum()+second.sum();}public static void main(String[]args){System.out.println(work(5));}}`, "22\n16\n"},
		{"innerReceiver", `package instance;class Owner{int value=8;class A1{int read(){return value;}}}public class Main{public static void main(String[]args){Owner owner=new Owner();Owner.A1 value=owner.new A1();System.out.println(value.read());}}`, "8\n"},
		{"suffixNominal", `package instance;class A1{}class a1_0{}class a1_1{}public class Main{public static void main(String[]args){System.out.println(new A1()!=null);}}`, "true\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>instance</groupId><artifactId>hygiene</artifactId><version>1</version></project>`, "src/main/java/instance/Main.java": test.source}, "instance.Main", test.want)
		})
	}
}

// Inspect the emitted scope, not a chosen suffix spelling. Nominal selectors,
// source parameter identity, and constructor ABI must remain untouched.
func TestGeneratedInstanceIdentifierScope(t *testing.T) {
	helper := setupParseHelper(t, `class Root{} class A1<T> extends Root {int a1; int read(int a1_0){return a1_0;}} class a1_0{} class a1_1{}`)
	owner := helper.File.Symbols.FindClassScope("A1")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.className = owner.Class.Name
	clones := generateObjectCloneDecls(ctx)
	if len(clones) != 1 {
		t.Fatalf("clone declarations=%d", len(clones))
	}
	clone := clones[0].(*ast.FuncDecl)
	receiver := clone.Recv.List[0].Names[0].Name
	forbidden := map[string]bool{owner.Class.Name: true, "a1_0": true, "a1_1": true}
	for _, name := range owner.GoTypeParameterNames() {
		forbidden[name] = true
	}
	if forbidden[receiver] {
		t.Fatalf("clone receiver %q hides nominal/binder/source binding", receiver)
	}
	defaults := buildDefaultConstructorDeclsWithOptions(ctx, constructorLoweringOptions{})
	found := false
	for _, decl := range defaults {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		for _, stmt := range fn.Body.List {
			assign, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				continue
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != "new" {
				continue
			}
			found = true
			local := assign.Lhs[0].(*ast.Ident).Name
			if forbidden[local] {
				t.Fatalf("constructor instance %q hides nominal/binder/source binding", local)
			}
		}
	}
	if !found {
		t.Fatal("default constructor allocation absent")
	}
	if owner.Class.Name != "a1" || owner.Class.OriginalName != "A1" || owner.FindField().ByOriginalName("a1")[0].Name != "a1" {
		t.Fatal("nominal or field ABI changed")
	}
	method := owner.FindMethod().ByOriginalName("read")[0]
	if len(method.Parameters) != 1 || method.Parameters[0].OriginalName != "a1_0" || method.Parameters[0].OriginalType != "int" {
		t.Fatal("source parameter identity changed")
	}
}

func TestGeneratedInstanceHiddenFormalAndCapture(t *testing.T) {
	helper := setupParseHelper(t, `class A1 {int captured;} class a1_0 {}`)
	owner := helper.File.Symbols.FindClassScope("A1")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.className = owner.Class.Name
	options := localConstructorOptions(owner.Class.Name, []localConstructorCapture{{capture: capturedLocal{name: "captured", goType: ast.NewIdent("int32")}, parameterName: "a1_1"}}, "")
	decls := buildDefaultConstructorDeclsWithOptions(ctx, options)
	found := false
	for _, decl := range decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		instance := ""
		for _, stmt := range fn.Body.List {
			assign, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				continue
			}
			if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "new" {
					instance = assign.Lhs[0].(*ast.Ident).Name
				}
			}
			if sel, ok := assign.Lhs[0].(*ast.SelectorExpr); ok && sel.Sel.Name == "captured" {
				found = true
				if instance == "" || sel.X.(*ast.Ident).Name != instance {
					t.Fatalf("capture assigned to %s instead of allocated instance %s", sel.X.(*ast.Ident).Name, instance)
				}
				if instance == "a1" || instance == "a1_0" || instance == "a1_1" {
					t.Fatalf("instance %s hides nominal/hidden formal", instance)
				}
				if assign.Rhs[0].(*ast.Ident).Name != "a1_1" {
					t.Fatal("capture ordinal/formal changed")
				}
			}
		}
	}
	if !found {
		t.Fatal("pre-super capture assignment absent")
	}
}

func TestGeneratedInstanceBuiltinAndTypeReservations(t *testing.T) {
	helper := setupParseHelper(t, `class A1<T>{} class a1_0 {}`)
	owner := helper.File.Symbols.FindClassScope("A1")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.importAliases = map[string]string{"foreign.pkg": "external"}
	fields := []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("a1_1")}, Type: &ast.IndexExpr{X: ast.NewIdent("LaterWitness"), Index: ast.NewIdent("LateBinder")}}}
	used := generatedInstanceUsedNames(ctx, fields...)
	for _, name := range []string{"new", "nil", "a1", "a1_0", "a1_1", "external", "LaterWitness", "LateBinder", owner.GoTypeParameterNames()[0]} {
		allocated := synchronizedUniqueLocalName(name, used)
		if allocated == name {
			t.Fatalf("synthetic binding hides required builtin/type/formal %s", name)
		}
	}
}
