package transpiler

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// A method variable's Java bound supplies callable members, but an exact Go
// pointer constraint does not expose its fields. Prove the actual helper's
// constraint before converting that scalar receiver to its source subobject.
// This leaves existing witness and class-owned projections separate.
func methodBoundConcretePointerReceiverExpr(value ast.Expr, origin inferredJavaTypeOrigin, expectedScope *symbol.ClassScope, ctx Ctx) (ast.Expr, bool) {
	if value == nil || origin.parameter == nil || origin.nominalScope != nil || origin.unresolved || expectedScope == nil || ctx.currentClass == nil || ctx.localScope == nil {
		return nil, false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(origin.javaType))
	_, arguments := parseJavaTypeString(base)
	if base == "" || rank != 0 || len(arguments) != 0 || len(origin.arguments) != 0 {
		return nil, false
	}
	method := ctx.localScope
	declaring, known := genericMethodOwnerContext(method)
	if !known || declaring.currentClass != ctx.currentClass {
		return nil, false
	}
	if method.IsStatic {
		if planStaticGenericMethodPhysicalABI(ctx.currentClass, method, ctx) == nil {
			return nil, false
		}
	} else if planGenericMethodEntryABI(ctx.currentClass, method, ctx) == nil {
		return nil, false
	}
	slots, declaring, known := declaredGenericMethodPhysicalSlots(method)
	if !known {
		return nil, false
	}
	for _, slot := range slots {
		if slot.parameter.Declaration != origin.parameter {
			continue
		}
		if slot.implicitObject {
			return nil, false
		}
		bound := resolveClassScopeByQualifiedName(declaring, slot.javaType)
		if bound == nil || bound != expectedScope || bound.Class == nil || bound.Class.DeclarationNode == nil || bound.IsInterface || abstractClassUsesInterfaceView(bound) {
			return nil, false
		}
		// Use the same declaration list and constraint emitter as the typed
		// helper. A textual upper bound or an inferred caller argument alone
		// cannot prove that a pointer conversion is legal in the emitted body.
		parameters := symbol.AppendTypeParamsByDeclaration(ctx.currentClass.TypeParameters, method.TypeParameters)
		fields := makeTypeParamFieldsInContext(parameters, ctx)
		for index, parameter := range parameters {
			if parameter.Declaration != origin.parameter || index >= len(fields) {
				continue
			}
			pointer, ok := fields[index].Type.(*ast.StarExpr)
			if !ok {
				return nil, false
			}
			var goArguments []ast.Expr
			switch head := pointer.X.(type) {
			case *ast.IndexExpr:
				goArguments = []ast.Expr{head.Index}
			case *ast.IndexListExpr:
				goArguments = head.Indices
			}
			if !reflect.DeepEqual(pointer, classSubobjectPointerTypeWithGoArguments(bound, goArguments, ctx)) {
				return nil, false
			}
			return &ast.CallExpr{Fun: pointer, Args: []ast.Expr{value}}, true
		}
		return nil, false
	}
	return nil, false
}
