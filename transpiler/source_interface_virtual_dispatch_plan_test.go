package transpiler

import (
	"fmt"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestSourceInterfaceVirtualEntryDeclarationGuards(t *testing.T) {
	helper := setupParseHelper(t, `public class Guard {interface Valid {Object value();}interface Owner<T>{T value();}interface Method{<T>T value();}interface Defaults{default Object value(){return null;}}static class R implements Valid {public Object value(){return null;}}static class D extends R {public Object value(){return null;}}static class DefaultOnly implements Defaults{}}`)
	owner := overrideBridgeTestScope(t, helper, "Valid")
	method := overrideBridgeTestMethod(t, owner, "value")
	if planSourceInterfaceVirtualEntry(owner, method, helper.Ctx) == nil {
		t.Fatal("actual declared nongeneric interface contract missing")
	}
	copied := *method
	for _, c := range []struct {
		name   string
		owner  *symbol.ClassScope
		method *symbol.Definition
	}{
		{"ownerless", nil, method}, {"class", overrideBridgeTestScope(t, helper, "R"), method}, {"copied definition", owner, &copied}, {"class binder", overrideBridgeTestScope(t, helper, "Owner"), overrideBridgeTestMethod(t, overrideBridgeTestScope(t, helper, "Owner"), "value")}, {"method binder", overrideBridgeTestScope(t, helper, "Method"), overrideBridgeTestMethod(t, overrideBridgeTestScope(t, helper, "Method"), "value")},
	} {
		t.Run(c.name, func(t *testing.T) {
			if planSourceInterfaceVirtualEntry(c.owner, c.method, helper.Ctx) != nil {
				t.Fatal("unproven method contract admitted")
			}
		})
	}
	defaults := overrideBridgeTestScope(t, helper, "DefaultOnly")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = defaults
	if got := generateSourceInterfaceVirtualEntries(ctx); len(got) != 0 {
		t.Fatalf("default-only class acquired a fabricated class-body entry: %d", len(got))
	}
	declarationless := *method
	declarationless.DeclarationNode = nil
	owner.Methods = append(owner.Methods, &declarationless)
	if planSourceInterfaceVirtualEntry(owner, &declarationless, helper.Ctx) != nil {
		t.Fatal("synthetic method acquired source interface entry")
	}
}

func TestSourceInterfaceVirtualEntryReservesFormals(t *testing.T) {
	helper := setupParseHelper(t, `public class Formals{interface View{void put(int __java2goInterfaceParent,int __java2goExecution);}static class R implements View{public void put(int x,int y){}}static class D extends R{public void put(int x,int y){}}}`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = overrideBridgeTestScope(t, helper, "R")
	declarations := generateSourceInterfaceVirtualEntries(ctx)
	if len(declarations) != 1 {
		t.Fatalf("entries=%d", len(declarations))
	}
	declaration := declarations[0].(*ast.FuncDecl)
	names := map[string]bool{}
	receiver := declaration.Recv.List[0].Names[0].Name
	names[receiver] = true
	for _, p := range declaration.Type.Params.List {
		for _, n := range p.Names {
			if names[n.Name] {
				t.Fatalf("allocated wrapper local shadows %q", n.Name)
			}
			names[n.Name] = true
		}
	}
}

func TestSourceInterfaceVirtualEntryHandwrittenFallbacks(t *testing.T) {
	const source = `interface View {Object value();}class Child extends Parent{public String value(){return "child";}}public class Parent implements View{public Object value(){return "parent";}public static Object invoke(View receiver){try{return receiver.value();}catch(NullPointerException e){return "npe";}}}`
	helper := setupParseHelper(t, source)
	view := overrideBridgeTestScope(t, helper, "View")
	parent := overrideBridgeTestScope(t, helper, "Parent")
	child := overrideBridgeTestScope(t, helper, "Child")
	method := overrideBridgeTestMethod(t, view, "value")
	invoke := overrideBridgeTestMethod(t, parent, "invoke")
	selector := method.Name
	hidden := executionImplementationName(method, view, helper.Ctx)
	invocation := executionImplementationName(invoke, parent, helper.Ctx)
	ctor := defaultConstructorName(child.Class.Name)
	generated := renderGoFileFromJava(t, source)
	driver := fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
type publicOnly struct{}
func(publicOnly) %[1]s() any{return j.JavaStringFromHostUTF8("public")}
type oldCompanion struct{last *j.Execution}
func(*oldCompanion) %[1]s() any{return j.JavaStringFromHostUTF8("public")}
func(c *oldCompanion) %[2]s(e *j.Execution) any{c.last=e;return j.JavaStringFromHostUTF8("hidden")}
func TestFallback(t *testing.T){e:=j.NewExecution();assert:=func(got any,want string){s,ok:=got.(*j.JavaString);if !ok||!s.Equals(j.JavaStringFromHostUTF8(want)){t.Fatalf("got %%v want %%q",got,want)}};assert(%[3]s(e,publicOnly{}),"public");c:=&oldCompanion{};assert(%[3]s(e,c),"hidden");if c.last!=e{t.Fatal("caller Execution lost")};assert(%[3]s(e,&%[4]s{}),"parent");assert(%[3]s(e,(%[5]s)(nil)),"npe");assert(%[3]s(e,(*%[4]s)(nil)),"npe");assert(%[3]s(e,%[6]s()),"child")}
`, selector, hidden, invocation, parent.Class.Name, view.Class.Name, ctor)
	root := repoRoot(t)
	dir := t.TempDir()
	prepareCampaignReadonlyRuntimeModule(t, dir, root)
	for name, data := range map[string]string{"generated.go": generated, "fallback_test.go": driver} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "test", "-race", "-p=1", "-mod=readonly", "-timeout=30s", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("handwritten Go fallback contract: %v\n%s", err, strings.TrimSpace(string(output)))
	}
}
