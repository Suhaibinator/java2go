package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestReaderLockCanonicalSelection131(t *testing.T) {
	helper := setupParseHelper(t, `import java.io.Reader;class Base extends Reader{}class Child extends Base{}class Hidden extends Base{Object lock;}class CyclicA extends CyclicB{}class CyclicB extends CyclicA{}`)
	base := helper.File.Symbols.FindClassScope("Base")
	child := helper.File.Symbols.FindClassScope("Child")
	hidden := helper.File.Symbols.FindClassScope("Hidden")
	ctx := Ctx{currentFile: helper.File.Symbols, currentClass: child, className: child.Class.Name}
	if !canonicalReaderNativeEdge(base, ctx) || canonicalReaderNativeEdge(child, ctx) || !sourceSelectsReaderLock(child, ctx) || sourceSelectsReaderLock(hidden, ctx) {
		t.Fatal("declaration-bound native carrier/hidden field selection changed")
	}
	if readerNativeCarrierPath(child, ast.NewIdent("actual"), ctx) == nil {
		t.Fatal("source hierarchy lost exact carrier")
	}
	if sourceSelectsReaderLock(helper.File.Symbols.FindClassScope("CyclicA"), ctx) || readerNativeCarrierPath(helper.File.Symbols.FindClassScope("CyclicA"), ast.NewIdent("actual"), ctx) != nil {
		t.Fatal("cyclic/unknown ancestry admitted")
	}
	if canonicalReaderNativeEdge(nil, ctx) || sourceSelectsReaderLock(nil, ctx) || readerNativeCarrierPath(nil, nil, ctx) != nil {
		t.Fatal("nil owner admitted")
	}
	static := ctx
	static.localScope = &symbol.Definition{IsStatic: true}
	if unqualifiedReaderLockAccess(static) != nil {
		t.Fatal("static inherited field access admitted")
	}
	parameter := ctx
	parameter.localScope = &symbol.Definition{Parameters: []*symbol.Definition{{Name: "lock", OriginalName: "lock", OriginalType: "java.lang.Object"}}}
	if unqualifiedReaderLockAccess(parameter) != nil {
		t.Fatal("source parameter lost precedence")
	}
	field := ctx
	field.currentClass = hidden
	if unqualifiedReaderLockAccess(field) != nil {
		t.Fatal("source field lost precedence")
	}
	foreign := *base
	foreign.Superclass = "foreign.Reader"
	if canonicalReaderNativeEdge(&foreign, ctx) || sourceSelectsReaderLock(&foreign, ctx) {
		t.Fatal("qualified external lookalike admitted")
	}
}
