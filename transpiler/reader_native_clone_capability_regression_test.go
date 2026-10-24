package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// The first row is the full original nativeCloneHeld Java source. The alias
// row is additive: a Java field may own ReaderBase while native storage uses
// the already-selected declaration-owned alias.
func TestReaderNativeCloneActualSelectionAndCopier(t *testing.T) {
	for _, row := range []struct{ name, source, owner string }{
		{"original", readerNativeClone4059OriginalJava, "Main.R"},
		{"carrierAlias", `import java.io.Reader;class R extends Reader implements java.lang.Cloneable{public Object ReaderBase;public int read(char[] b,int o,int n){return -1;}public void close(){}R copy()throws CloneNotSupportedException{return(R)super.clone();}}`, "R"},
	} {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, row.source)
			owner := resolveClassScopeByQualifiedName(helper.Ctx, row.owner)
			if owner == nil || owner.Class == nil {
				t.Fatal("actual registered source owner missing")
			}
			ctx := classScopeCtx(owner, helper.Ctx)
			ctx.className = owner.Class.Name
			ctx.localScope = owner.FindMethodByName("copy", nil)
			ctx.executionContextName = "__cloneExecution"
			if ctx.localScope == nil {
				t.Fatal("actual copy declaration missing")
			}
			header := classHeaderTypeCtx(owner, ctx)
			origin := declaredJavaTypeOrigin(symbol.JavaType{Original: owner.Superclass, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(header))}, header)
			if origin.unresolved || origin.externalName != "java.io.Reader" || origin.nominalScope != nil || origin.parameter != nil {
				t.Fatalf("actual native declaration origin not established: %+v", origin)
			}
			slot := superclassEmbeddedSelectorName(owner, ctx)
			projection, ok := readerNativeCarrierPath(owner, ast.NewIdent("actualAllocation"), ctx).(*ast.SelectorExpr)
			if !ok || slot == "" || slot == "Reader" || projection.Sel.Name != slot {
				t.Fatalf("native storage certificate/projection inconsistent: slot=%q projection=%#v", slot, projection)
			}
			if row.name == "carrierAlias" && slot == "ReaderBase" {
				t.Fatal("native carrier stole Java source ReaderBase field")
			}
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if invocation == nil {
				t.Fatal("actual super.clone invocation missing")
			}
			emitted := ParseExpr(invocation, helper.File.Source, ctx)
			var text bytes.Buffer
			if err := printer.Fprint(&text, token.NewFileSet(), emitted); err != nil {
				t.Fatal(err)
			}
			t.Logf("actual source=%s selectedObjectClone=%t nativeSlot=%s invocation=%s", row.owner, sourceSuperSelectsObjectClone(owner, ctx), slot, text.String())
			call, callOK := emitted.(*ast.CallExpr)
			canonicalCall := false
			if callOK {
				if selector, yes := call.Fun.(*ast.SelectorExpr); yes && selector.Sel.Name == "ObjectCloneExecution" && len(call.Args) == 2 {
					execution, executionOK := call.Args[0].(*ast.Ident)
					receiver, receiverOK := call.Args[1].(*ast.Ident)
					canonicalCall = executionOK && execution.Name == "__cloneExecution" && receiverOK && receiver.Name == sourceInstanceReceiverName(ctx)
				}
			}
			if !canonicalCall {
				t.Error("actual canonical Reader super.clone did not lower to inherited Object.clone with exact invocation Execution and source receiver")
			}
			copiers := generateObjectCloneDecls(ctx)
			if len(copiers) != 1 {
				t.Errorf("actual canonical Reader source copier declarations=%d, want one", len(copiers))
				return
			}
			copier, ok := copiers[0].(*ast.FuncDecl)
			if !ok || copier.Name.Name != generatedObjectCloneMethod || copier.Body == nil {
				t.Fatal("source clone copier protocol missing")
			}
			nativeCopies := 0
			ast.Inspect(copier.Body, func(node ast.Node) bool {
				assignment, ok := node.(*ast.AssignStmt)
				if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
					return true
				}
				lhs, ok := assignment.Lhs[0].(*ast.SelectorExpr)
				if !ok || lhs.Sel.Name != slot {
					return true
				}
				rhs, ok := assignment.Rhs[0].(*ast.CallExpr)
				if !ok || len(rhs.Args) != 1 {
					return true
				}
				fn, ok := rhs.Fun.(*ast.SelectorExpr)
				if !ok || fn.Sel.Name != "CloneReaderBaseReference" {
					return true
				}
				value, ok := rhs.Args[0].(*ast.SelectorExpr)
				if !ok || value.Sel.Name != slot {
					return true
				}
				receiver, ok := value.X.(*ast.Ident)
				if ok && receiver.Name == copier.Recv.List[0].Names[0].Name {
					nativeCopies++
				}
				return true
			})
			if nativeCopies != 1 {
				t.Errorf("native carrier copy at actual selected slot=%d, want one independent shallow native-state copy", nativeCopies)
			}
		})
	}
}

func TestReaderNativeCloneSourceOverrideAndOverload(t *testing.T) {
	for _, row := range []struct {
		name, source string
		wantObject   bool
	}{
		{"sourceOverride", `class Parent{protected Object clone(){return this;}}class R extends Parent implements java.lang.Cloneable{R copy(){return(R)super.clone();}}`, false},
		{"sourceReaderHomonym", `class Reader{protected Object clone(){return this;}}class R extends Reader implements java.lang.Cloneable{R copy(){return(R)super.clone();}}`, false},
		{"sourceOverload", `class Parent implements java.lang.Cloneable{Object clone(int ignored){return this;}}class R extends Parent{R copy()throws CloneNotSupportedException{return(R)super.clone();}}`, true},
	} {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, row.source)
			owner := resolveClassScopeByQualifiedName(helper.Ctx, "R")
			if owner == nil {
				t.Fatal("actual source owner missing")
			}
			ctx := classScopeCtx(owner, helper.Ctx)
			if got := sourceSuperSelectsObjectClone(owner, ctx); got != row.wantObject {
				t.Fatalf("actual selected declaration Object.clone=%t want %t", got, row.wantObject)
			}
		})
	}
}

const readerNativeClone4059OriginalJava = `package locks;import java.io.Reader;public class Main{
static class R extends Reader implements Cloneable{public int read(char[] b,int o,int n){return -1;}public void close(){} Object get(){return lock;}void put(Object v){lock=v;}R copy()throws CloneNotSupportedException{return(R)super.clone();}}
public static void main(String[]args)throws Exception{R first=new R();R clone=first.copy();boolean a=clone.get()==first;Object s=new Object();first.put(s);R other=first.copy();boolean b=other.get()==s;other.put(null);System.out.println(a+":"+b+":"+(first.get()==s)+":"+(other.get()==null));}}`
