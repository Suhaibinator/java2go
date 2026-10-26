package transpiler

import (
	"go/ast"
	"testing"
)

func TestReaderPlatformMemberMetadataAndEdge132(t *testing.T) {
	helper := setupParseHelper(t, `import java.io.Reader;interface View{Object JavaReaderState();}class R<T> extends Reader implements View{public Object ReaderBase;public Object JavaReaderState0;public Object JavaReaderState(){return this;}public Object JavaReaderState(int first,long second){return this;}public int read(char[] b,int off,int size){return -1;}public void close(){}}class D<T> extends R<T>{public Object JavaReaderState0;public Object JavaReaderState(){return this;}}class W extends java.io.Writer{public Object WriterBase;public void write(char[]b,int o,int n){}public void close(){}public void flush(){}}`)
	owner := helper.File.Symbols.FindClassScope("R")
	child := helper.File.Symbols.FindClassScope("D")
	view := helper.File.Symbols.FindClassScope("View")
	ctx := classScopeCtx(owner, helper.Ctx)
	if owner.Class.OriginalName != "R" || child.Class.OriginalName != "D" {
		t.Fatal("Java nominal identity changed")
	}
	field := owner.FindField().ByOriginalName("ReaderBase")[0]
	if field.OriginalName != "ReaderBase" || field.OriginalType != "Object" || field.Name != "ReaderBase" {
		t.Fatal("source carrier-named field identity changed")
	}
	methods := owner.FindMethod().ByOriginalName("JavaReaderState")
	if len(methods) != 2 {
		t.Fatal("source overloads lost")
	}
	var noarg string
	for _, method := range methods {
		if method.OriginalName != "JavaReaderState" || method.OriginalType != "Object" {
			t.Fatal("Java reflection name/result changed")
		}
		if method.Name == "JavaReaderState" {
			t.Fatal("source method occupies fixed runtime selector")
		}
		if len(method.Parameters) == 0 {
			noarg = method.Name
			continue
		}
		if len(method.Parameters) != 2 || method.Parameters[0].OriginalName != "first" || method.Parameters[0].OriginalType != "int" || method.Parameters[1].OriginalName != "second" || method.Parameters[1].OriginalType != "long" {
			t.Fatal("source parameter ordinal/name/type changed")
		}
	}
	if child.FindMethod().ByOriginalName("JavaReaderState")[0].Name != noarg || view.FindMethod().ByOriginalName("JavaReaderState")[0].Name != noarg {
		t.Fatal("genuine override/interface family split")
	}
	if len(owner.TypeParameters) != 1 || owner.TypeParameters[0].Name != "T" || len(child.TypeParameters) != 1 || child.TypeParameters[0].Name != "T" {
		t.Fatal("source binder identity changed")
	}
	if !superclassEmbeddingNeedsAlias(owner, ctx) {
		t.Fatal("carrier/source field collision has no storage alias")
	}
	selector := superclassEmbeddedSelectorName(owner, ctx)
	if selector == "" || selector == "ReaderBase" {
		t.Fatal("carrier still occupies source selector")
	}
	if superclassEmbeddingAliasDecl(owner, ctx) == nil {
		t.Fatal("allocated carrier alias missing declaration")
	}
	carrier, ok := readerNativeCarrierPath(owner, ast.NewIdent("actual"), ctx).(*ast.SelectorExpr)
	if !ok || carrier.Sel.Name != selector {
		t.Fatal("carrier consumer disagrees with declaration-owned selector")
	}
	writer := helper.File.Symbols.FindClassScope("W")
	if superclassEmbeddingNeedsAlias(writer, classScopeCtx(writer, helper.Ctx)) {
		t.Fatal("Reader alias scope expanded into native Writer")
	}
}
