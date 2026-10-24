package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
)

func sourceArrayListParameterRankFixture(t *testing.T, methodSource, name string) (*ParseHelper, *symbol.ClassScope, *symbol.Definition, Ctx) {
	t.Helper()
	h := setupParseHelper(t, `class C extends java.util.ArrayList<Object>{`+methodSource+`}`)
	owner := h.Ctx.currentFile.FindClassScope("C")
	if owner == nil || owner.Class.DeclarationNode == nil {
		t.Fatal("actual named source owner missing before parameter-rank proof")
	}
	method := owner.FindMethodByName(name, nil)
	if method == nil || method.DeclarationNode == nil || len(method.Parameters) != 1 {
		t.Fatal("actual source method/formal missing before parameter-rank proof")
	}
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	return h, owner, method, ctx
}

func TestSourceArrayListParameterRankOrdinaryObjectAddControl(t *testing.T) {
	h, owner, method, ctx := sourceArrayListParameterRankFixture(t, `public boolean add(Object value){return super.add(value);}`, "add")
	written := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
	if len(written) != 1 || written[0].Type() != "formal_parameter" || !sourceArrayListNativeNoDimensions(written[0]) || method.IsVariadic {
		t.Fatal("ordinary scalar formal premise missing")
	}
	typ, _ := nodeutil.JavaParameterNodes(written[0])
	if typ == nil || typ.Content(h.File.Source) != "Object" || method.Parameters[0].OriginalType != "Object" {
		t.Fatal("actual ordinary Object prefix premise missing")
	}
	selected, declaring := sourceArrayListSelectedAdd(owner, ctx)
	if !sourceArrayListMethodOwned(method, owner, ctx) || !sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx) || sourceArrayListRoot(owner, ctx) != owner || selected != method || declaring != owner {
		t.Fatal("ordinary add(Object) lost actual source native-alias slot")
	}
	t.Log("ORDINARY_OBJECT_ADD_CONTROL actual formal scalar/owned/slot/root/selection confirmed")
}

func TestSourceArrayListParameterRankSuffixArrayAddIsNotObjectSlot(t *testing.T) {
	h, owner, method, ctx := sourceArrayListParameterRankFixture(t, `public boolean add(Object value[]){return super.add(value);}`, "add")
	written := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
	if len(written) != 1 || written[0].Type() != "formal_parameter" || sourceArrayListNativeNoDimensions(written[0]) || method.IsVariadic {
		t.Fatal("actual after-name array formal premise missing")
	}
	typ, _ := nodeutil.JavaParameterNodes(written[0])
	if typ == nil || typ.Content(h.File.Source) != "Object" || method.Parameters[0].OriginalType != "Object" {
		t.Fatal("actual suffix-array prefix-only sidecar premise missing")
	}
	selected, _ := sourceArrayListSelectedAdd(owner, ctx)
	slot := sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx)
	t.Logf("SUFFIX_ARRAY_ADD_ACTUAL owned=%v scalarSlot=%v rootAdmitted=%v selectedBusinessOverload=%v", sourceArrayListMethodOwned(method, owner, ctx), slot, sourceArrayListRoot(owner, ctx) != nil, selected == method)
	if slot || selected == method {
		t.Fatal("suffix-array business add(Object[]) acquired native add(Object) slot after actual formal-rank proof")
	}
	// Either retaining an independent inherited native add or refusing this
	// entire unsupported class is safe. Selecting this overload is not.
}

func TestSourceArrayListParameterRankVarargsAddIsNotObjectSlot(t *testing.T) {
	h, owner, method, ctx := sourceArrayListParameterRankFixture(t, `public boolean add(Object... values){return super.add(values);}`, "add")
	written := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
	if len(written) != 1 || written[0].Type() != "spread_parameter" || !executionParameterIsVariadic(method, 0) {
		t.Fatal("actual spread-parameter and existing execution variadic query premise missing")
	}
	typ, _ := nodeutil.JavaParameterNodes(written[0])
	if typ == nil || typ.Content(h.File.Source) != "Object" || method.Parameters[0].OriginalType != "Object" {
		t.Fatal("actual varargs prefix-only sidecar premise missing")
	}
	original := method.IsVariadic
	method.IsVariadic = true
	metadataTrueSlot := sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx)
	method.IsVariadic = false
	defer func() { method.IsVariadic = original }()
	selected, _ := sourceArrayListSelectedAdd(owner, ctx)
	slot := sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx)
	t.Logf("VARARGS_ADD_ACTUAL writtenSpread=true executionQueryVariadic=true metadataTrueSlot=%v variadicMetadata=false owned=%v scalarSlot=%v rootAdmitted=%v selectedBusinessOverload=%v", metadataTrueSlot, sourceArrayListMethodOwned(method, owner, ctx), slot, sourceArrayListRoot(owner, ctx) != nil, selected == method)
	if metadataTrueSlot || slot || selected == method {
		t.Fatal("varargs business add(Object...) acquired native add(Object) slot after actual spread-rank proof with metadata flag cleared")
	}
}

func TestSourceArrayListParameterRankPrefixArrayOverrideRefusalControl(t *testing.T) {
	_, owner, method, ctx := sourceArrayListParameterRankFixture(t, `public Object[] toArray(Object[] value){return value;}`, "toArray")
	if !sourceArrayListMethodOwned(method, owner, ctx) || !sourceArrayListSlotMatches(method, []string{"java.lang.Object[]"}, ctx) || !sourceArrayListUncoveredOverride(owner, ctx) || sourceArrayListRoot(owner, ctx) != nil {
		t.Fatal("ordinary prefix-array platform toArray override refusal premise missing")
	}
	t.Log("PREFIX_ARRAY_TO_ARRAY_CONTROL actual array override remains strictly refused")
}

func TestSourceArrayListParameterRankSuffixArrayOverrideCannotEvadeRefusal(t *testing.T) {
	h, owner, method, ctx := sourceArrayListParameterRankFixture(t, `public Object[] toArray(Object value[]){return value;}`, "toArray")
	written := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
	if len(written) != 1 || written[0].Type() != "formal_parameter" || sourceArrayListNativeNoDimensions(written[0]) {
		t.Fatal("actual suffix-array toArray formal premise missing")
	}
	typ, _ := nodeutil.JavaParameterNodes(written[0])
	if typ == nil || typ.Content(h.File.Source) != "Object" || method.Parameters[0].OriginalType != "Object" {
		t.Fatal("actual toArray suffix prefix-only sidecar premise missing")
	}
	t.Logf("SUFFIX_ARRAY_TO_ARRAY_ACTUAL owned=%v arraySlot=%v uncovered=%v rootAdmitted=%v", sourceArrayListMethodOwned(method, owner, ctx), sourceArrayListSlotMatches(method, []string{"java.lang.Object[]"}, ctx), sourceArrayListUncoveredOverride(owner, ctx), sourceArrayListRoot(owner, ctx) != nil)
	if sourceArrayListRoot(owner, ctx) != nil || !sourceArrayListUncoveredOverride(owner, ctx) {
		t.Fatal("suffix-array toArray(Object[]) override evaded unsupported native-slot refusal after actual formal-rank proof")
	}
}
