package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// A Java instance body may need a different local spelling from its nominal Go
// type. Keep that spelling in its lexical context, never in Java symbol data.
func sourceInstanceReceiverName(ctx Ctx) string {
	if ctx.emittedReceiverOwner != nil && ctx.emittedReceiverOwner == ctx.currentClass && ctx.emittedReceiverName != "" && ctx.localScope != nil && !ctx.localScope.IsStatic && !ctx.localScope.Constructor {
		return ctx.emittedReceiverName
	}
	name := ctx.className
	if name == "" && ctx.currentClass != nil && ctx.currentClass.Class != nil {
		name = ctx.currentClass.Class.Name
	}
	return ShortName(name)
}

func withSourceInstanceReceiver(ctx Ctx, name string) Ctx {
	ctx.emittedReceiverOwner = ctx.currentClass
	ctx.emittedReceiverName = name
	return ctx
}

// Late helper binders use declaration identity before allocating Go spellings.
// Disambiguation is read-only here: the existing helper generator remains the
// point where its final combined ABI names are committed to source metadata.
func sourceInstanceReceiverPlannedBinders(ctx Ctx) []symbol.TypeParam {
	var owner, method []symbol.TypeParam
	if ctx.currentClass != nil {
		owner = ctx.currentClass.TypeParameters
	}
	if ctx.localScope != nil {
		method = ctx.localScope.TypeParameters
	}
	original := symbol.AppendTypeParamsByDeclaration(owner, method)
	planned := append([]symbol.TypeParam(nil), original...)
	for index := range planned {
		if declaration := planned[index].Declaration; declaration != nil {
			copy := *declaration
			planned[index].Declaration = &copy
		}
	}
	symbol.DisambiguateTypeParamGoNames(planned)
	return planned
}

func sourceInstanceReceiverUsedNames(ctx Ctx, fields ...*ast.Field) map[string]struct{} {
	var used map[string]struct{}
	if ctx.currentFile != nil && ctx.localScope != nil && ctx.localScope.DeclarationNode != nil {
		used = generatedInstanceUsedNamesForDeclaration(ctx, ctx.localScope.DeclarationNode, fields...)
	} else {
		used = generatedInstanceUsedNames(ctx, fields...)
	}
	if ctx.executionContextName != "" {
		used[ctx.executionContextName] = struct{}{}
	}
	if ctx.localScope != nil && ctx.localScope.RequiresHelper && ctx.localScope.HelperName != "" {
		used[ctx.localScope.HelperName] = struct{}{}
	}
	for _, parameter := range sourceInstanceReceiverPlannedBinders(ctx) {
		used[parameter.EmittedName()] = struct{}{}
	}
	// The synthetic instance allocator handles ordinary type operands. Method
	// signatures additionally contain nested callback/interface field lists.
	var reserveType func(ast.Expr)
	reserveFields := func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, field := range list.List {
			if field != nil {
				reserveType(field.Type)
			}
		}
	}
	reserveType = func(expr ast.Expr) {
		switch typ := expr.(type) {
		case *ast.Ident:
			used[typ.Name] = struct{}{}
		case *ast.SelectorExpr:
			reserveType(typ.X)
		case *ast.StarExpr:
			reserveType(typ.X)
		case *ast.IndexExpr:
			reserveType(typ.X)
			reserveType(typ.Index)
		case *ast.IndexListExpr:
			reserveType(typ.X)
			for _, arg := range typ.Indices {
				reserveType(arg)
			}
		case *ast.ArrayType:
			reserveType(typ.Elt)
		case *ast.MapType:
			reserveType(typ.Key)
			reserveType(typ.Value)
		case *ast.ChanType:
			reserveType(typ.Value)
		case *ast.Ellipsis:
			reserveType(typ.Elt)
		case *ast.ParenExpr:
			reserveType(typ.X)
		case *ast.FuncType:
			reserveFields(typ.TypeParams)
			reserveFields(typ.Params)
			reserveFields(typ.Results)
		case *ast.InterfaceType:
			reserveFields(typ.Methods)
		case *ast.StructType:
			reserveFields(typ.Fields)
		case *ast.UnaryExpr:
			reserveType(typ.X)
		case *ast.BinaryExpr:
			reserveType(typ.X)
			reserveType(typ.Y)
		}
	}
	for _, field := range fields {
		if field != nil {
			reserveType(field.Type)
		}
	}
	return used
}
