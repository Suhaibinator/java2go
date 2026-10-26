package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// Prepared only: requires the proposed source-instance context API. The future
// lane must compile and execute these; source preparation is not acceptance.
func TestSourceInstanceReceiverContextOwnership(t *testing.T) {
	owner := &symbol.ClassScope{Class: &symbol.Definition{Name: "a1", OriginalName: "A1"}}
	other := &symbol.ClassScope{Class: &symbol.Definition{Name: "a1", OriginalName: "A1"}}
	original := Ctx{currentClass: owner, className: "a1", localScope: &symbol.Definition{OriginalName: "read"}}
	bound := withSourceInstanceReceiver(original, "a1_2")
	if sourceInstanceReceiverName(original) != "a1" {
		t.Fatal("binding mutated parent context")
	}
	cloned := bound.Clone()
	if sourceInstanceReceiverName(cloned) != "a1_2" {
		t.Fatal("Clone dropped emitted binding")
	}
	cloned.currentClass = other
	if sourceInstanceReceiverName(cloned) != "a1" {
		t.Fatal("binding leaked to a same-spelled different owner")
	}
	cloned = bound.Clone()
	cloned.localScope = &symbol.Definition{OriginalName: "A1", Constructor: true}
	if sourceInstanceReceiverName(cloned) != "a1" {
		t.Fatal("explicit constructor fallback changed")
	}
	cloned = bound.Clone()
	cloned.localScope = &symbol.Definition{OriginalName: "staticRead", IsStatic: true}
	if sourceInstanceReceiverName(cloned) != "a1" {
		t.Fatal("binding leaked to static context")
	}
	lookup := classScopeCtx(owner, bound)
	if sourceInstanceReceiverName(lookup) != "a1" {
		t.Fatal("declaration lookup retained a body receiver")
	}
}

func TestSourceInstanceReceiverEmissionStamp(t *testing.T) {
	helper := setupParseHelper(t, `class A1 {int read(){return 1;}}`)
	owner := helper.File.Symbols.FindClassScope("A1")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.className = owner.Class.Name
	ctx.localScope = owner.FindMethod().ByOriginalName("read")[0]
	first := withSourceInstanceReceiver(ctx, "a1_0")
	second := withSourceInstanceReceiver(ctx, "a1_1")
	if anonymousReceiverStamp(helper.File.Source, first) == anonymousReceiverStamp(helper.File.Source, second) {
		t.Fatal("anonymous cache ignores emitted receiver name")
	}
	if anonymousReceiverStamp(helper.File.Source, first) != anonymousReceiverStamp(helper.File.Source, first.Clone()) {
		t.Fatal("Clone changes emission ownership")
	}
}

func TestSourceInstanceReceiverDeclarationMetadata(t *testing.T) {
	helper := setupParseHelper(t, `class A1<T>{int state;<U>A1<T> identity(U a1_0,int a1_1){return (A1<T>)this;}} class a1_2{}`)
	owner := helper.File.Symbols.FindClassScope("A1")
	method := owner.FindMethod().ByOriginalName("identity")[0]
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.className = owner.Class.Name
	ctx.localScope = method
	type binding struct{ Name, OriginalName, OriginalType string }
	snapshot := func() []binding {
		rows := []binding{{owner.Class.Name, owner.Class.OriginalName, owner.Class.OriginalType}}
		for _, f := range owner.Fields {
			rows = append(rows, binding{f.Name, f.OriginalName, f.OriginalType})
		}
		for _, arg := range method.Parameters {
			rows = append(rows, binding{arg.Name, arg.OriginalName, arg.OriginalType})
		}
		return rows
	}
	before := snapshot()
	classBinders := owner.GoTypeParameterNames()
	methodBinders := method.GoTypeParameterNames()
	decls := ParseDecl(method.DeclarationNode, helper.File.Source, ctx)
	if !reflect.DeepEqual(before, snapshot()) || !reflect.DeepEqual(classBinders, owner.GoTypeParameterNames()) || !reflect.DeepEqual(methodBinders, method.GoTypeParameterNames()) {
		t.Fatal("receiver allocation mutated nominal/selector/parameter/ordinal/binder metadata")
	}
	forbidden := map[string]bool{owner.Class.Name: true, "a1_2": true}
	for _, name := range append(classBinders, methodBinders...) {
		forbidden[name] = true
	}
	for _, arg := range method.Parameters {
		forbidden[arg.Name] = true
	}
	bodies := 0
	for _, decl := range decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		for _, stmt := range fn.Body.List {
			assignment, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assignment.Rhs) != 1 {
				continue
			}
			selected, ok := assignment.Rhs[0].(*ast.SelectorExpr)
			if !ok || selected.Sel.Name != "recv" {
				continue
			}
			bodies++
			sourceReceiver := assignment.Lhs[0].(*ast.Ident).Name
			if forbidden[sourceReceiver] {
				t.Fatalf("helper source receiver %q hides nominal/binder/source formal", sourceReceiver)
			}
			if fn.Recv != nil && fn.Recv.List[0].Names[0].Name == sourceReceiver {
				t.Fatal("helper physical and Java receiver collide")
			}
		}
	}
	if bodies == 0 {
		t.Fatal("generic-helper Java receiver binding absent")
	}
}

func TestSourceInstanceReceiverEmptyClassName(t *testing.T) {
	owner := &symbol.ClassScope{Class: &symbol.Definition{Name: "a1"}}
	ctx := Ctx{currentClass: owner, localScope: &symbol.Definition{OriginalName: "read"}}
	if got := sourceInstanceReceiverName(ctx); got != ShortName(owner.Class.Name) {
		t.Fatalf("empty class name fallback = %q", got)
	}
	bound := withSourceInstanceReceiver(ctx, "a1_0")
	if got := sourceInstanceReceiverName(bound); got != "a1_0" {
		t.Fatalf("active empty-class-name receiver = %q", got)
	}
	bound.currentClass = nil
	if got := sourceInstanceReceiverName(bound); got != "" {
		t.Fatalf("nil owner inherited another owner's receiver: %q", got)
	}
	if got := sourceInstanceReceiverName(Ctx{className: "Main"}); got != ShortName("Main") {
		t.Fatalf("unowned fallback = %q", got)
	}
}

func TestSourceInstanceReceiverReadOnlyCombinedBindersAndSignature(t *testing.T) {
	outer := symbol.NewTypeParam("a1", []symbol.JavaType{{Original: "java.lang.Object"}})
	method := symbol.NewTypeParam("a1", []symbol.JavaType{{Original: "java.lang.Number"}})
	suffix := symbol.NewTypeParam("a12", nil)
	// The method captures a copy of the exact outer declaration. Deduplicating
	// after pointer copies would incorrectly allocate a fourth ABI binder.
	owner := &symbol.ClassScope{Class: &symbol.Definition{Name: "a1"}, TypeParameters: []symbol.TypeParam{outer, suffix}}
	def := &symbol.Definition{OriginalName: "read", RequiresHelper: true, HelperName: "a1Helper", TypeParameters: []symbol.TypeParam{outer, method}}
	ctx := Ctx{currentClass: owner, className: "a1", localScope: def, executionContextName: "__java"}
	before := []symbol.TypeParam{outer, suffix, method}
	declarations := []symbol.TypeParamDeclaration{*outer.Declaration, *suffix.Declaration, *method.Declaration}
	nested := &ast.Field{Type: &ast.FuncType{
		Params:  &ast.FieldList{List: []*ast.Field{{Type: &ast.InterfaceType{Methods: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("unrelatedMethod")}, Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent("foreign"), Sel: ast.NewIdent("ForeignMember")}}}}}}}}}}}}},
		Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("lateType")}}},
	}}
	planned := sourceInstanceReceiverPlannedBinders(ctx)
	if len(planned) != 3 {
		t.Fatalf("captured declaration was duplicated: %d", len(planned))
	}
	used := sourceInstanceReceiverUsedNames(ctx, nested)
	for _, name := range []string{"a1", "a12", "a13", "a1Helper", "__java", "foreign", "lateType"} {
		if _, ok := used[name]; !ok {
			t.Errorf("missing actual/planned type or formal reservation %q", name)
		}
	}
	for _, name := range []string{"ForeignMember", "unrelatedMethod"} {
		if _, ok := used[name]; ok {
			t.Errorf("foreign selector/member over-reserved %q", name)
		}
	}
	for index, parameter := range []symbol.TypeParam{owner.TypeParameters[0], owner.TypeParameters[1], def.TypeParameters[1]} {
		if parameter.Declaration != before[index].Declaration || !reflect.DeepEqual(parameter.Bounds, before[index].Bounds) || *parameter.Declaration != declarations[index] {
			t.Fatal("read-only planner changed Java declaration identity, names or bounds")
		}
	}
}
