package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"sort"
	"strings"
)

// A native view is authorized by a resolved declared Java edge and an exact
// executable source SAM. Method shape alone is never a nominal proof.
type sourceNativeFunctionalContract struct {
	family                string
	arguments, parameters []string
	result, method        string
}

func nativeFunctionalFamily(javaType string, ctx Ctx) string {
	owner, ok := canonicalIntrinsicOwner(javaType, ctx)
	if !ok {
		return ""
	}
	for _, name := range []string{"Function", "BiFunction", "Consumer", "IntUnaryOperator", "IntBinaryOperator", "LongUnaryOperator", "LongBinaryOperator", "UnaryOperator", "BinaryOperator"} {
		if owner == "java.util.function."+name {
			return name
		}
	}
	return ""
}
func nativeFunctionalContract(family string, args []string) (sourceNativeFunctionalContract, bool) {
	c := sourceNativeFunctionalContract{family: family, arguments: append([]string(nil), args...), method: "apply"}
	count := 0
	switch family {
	case "Function":
		count = 2
	case "BiFunction":
		count = 3
	case "Consumer", "UnaryOperator", "BinaryOperator":
		count = 1
	}
	if len(c.arguments) == 0 && count > 0 {
		for i := 0; i < count; i++ {
			c.arguments = append(c.arguments, "java.lang.Object")
		}
	}
	if len(c.arguments) != count {
		return c, false
	}
	switch family {
	case "Function":
		c.parameters = []string{c.arguments[0]}
		c.result = c.arguments[1]
	case "BiFunction":
		c.parameters = c.arguments[:2]
		c.result = c.arguments[2]
	case "Consumer":
		c.parameters = c.arguments
		c.result = "void"
		c.method = "accept"
	case "UnaryOperator":
		c.parameters = c.arguments
		c.result = c.arguments[0]
	case "BinaryOperator":
		c.parameters = []string{c.arguments[0], c.arguments[0]}
		c.result = c.arguments[0]
	case "IntUnaryOperator":
		c.parameters = []string{"int"}
		c.result = "int"
		c.method = "applyAsInt"
	case "IntBinaryOperator":
		c.parameters = []string{"int", "int"}
		c.result = "int"
		c.method = "applyAsInt"
	case "LongUnaryOperator":
		c.parameters = []string{"long"}
		c.result = "long"
		c.method = "applyAsLong"
	case "LongBinaryOperator":
		c.parameters = []string{"long", "long"}
		c.result = "long"
		c.method = "applyAsLong"
	default:
		return c, false
	}
	return c, true
}
func sourceNativeFunctionalContracts(scope *symbol.ClassScope, ctx Ctx) []sourceNativeFunctionalContract {
	if scope == nil {
		return nil
	}
	seen := map[string]bool{}
	collected := map[string]sourceNativeFunctionalContract{}
	incompatible := map[string]bool{}
	var collect func(string, []string)
	collect = func(name string, args []string) {
		c, ok := nativeFunctionalContract(name, args)
		if !ok {
			return
		}
		if old, found := collected[name]; found && strings.Join(old.arguments, "\x00") != strings.Join(c.arguments, "\x00") {
			incompatible[name] = true
			return
		}
		collected[name] = c
		if name == "UnaryOperator" {
			collect("Function", []string{c.arguments[0], c.arguments[0]})
		}
		if name == "BinaryOperator" {
			collect("BiFunction", []string{c.arguments[0], c.arguments[0], c.arguments[0]})
		}
	}
	var visit func(*symbol.ClassScope, []string)
	visit = func(owner *symbol.ClassScope, args []string) {
		if owner == nil {
			return
		}
		key := javaClassBinaryName(owner) + "\x00" + strings.Join(args, "\x00")
		if seen[key] {
			return
		}
		seen[key] = true
		ownerCtx := classScopeCtx(owner, ctx)
		bindings := map[string]string{}
		for i, p := range owner.TypeParameters {
			if i < len(args) {
				bindings[p.Name] = args[i]
				bindings[p.EmittedName()] = args[i]
			}
		}
		for _, edge := range append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass) {
			q := substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(edge, owner), bindings)
			base, a := parseJavaTypeString(q)
			if f := nativeFunctionalFamily(q, ownerCtx); f != "" {
				collect(f, a)
			} else {
				visit(resolveClassScopeByQualifiedName(ownerCtx, base), a)
			}
		}
	}
	visit(scope, scope.GoTypeParameterNames())
	names := []string{}
	for name := range collected {
		if !incompatible[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := []sourceNativeFunctionalContract{}
	for _, name := range names {
		result = append(result, collected[name])
	}
	return result
}
func resolveSourceNativeSAM(scope *symbol.ClassScope, c sourceNativeFunctionalContract, ctx Ctx) (*symbol.Definition, *symbol.ClassScope, string) {
	for owner := scope; owner != nil; owner = resolveSuperclassScopeInDeclaringContext(ctx, owner) {
		arguments := mapClassTypeArgumentStringsToAncestor(scope, scope.GoTypeParameterNames(), owner, ctx)
		bindings := map[string]string{}
		for i, p := range owner.TypeParameters {
			if i < len(arguments) {
				bindings[p.Name] = arguments[i]
				bindings[p.EmittedName()] = arguments[i]
			}
		}
		for _, m := range owner.Methods {
			if m == nil || m.OriginalName != c.method || m.IsStatic || m.IsPrivate || len(m.Parameters) != len(c.parameters) || len(m.TypeParameters) != 0 || !m.HasBody {
				continue
			}
			match := true
			for i, want := range c.parameters {
				got := substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(m, i), owner), bindings)
				if !javaInferenceSameType(got, want, ctx) {
					match = false
					break
				}
			}
			if match {
				return m, owner, substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(m.OriginalType, owner), bindings)
			}
		}
	}
	return nil, nil, ""
}
func sourceNativeFunctionalViewExpr(scope *symbol.ClassScope, c sourceNativeFunctionalContract, receiver ast.Expr, ctx Ctx) ast.Expr {
	method, owner, result := resolveSourceNativeSAM(scope, c, ctx)
	if method == nil {
		return nil
	}
	params := &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}
	args := []ast.Expr{ast.NewIdent("execution")}
	for i, p := range c.parameters {
		name := "value" + string(rune('0'+i))
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(name)}, Type: javaTypeStringToGoTypeExpr(p, inScopeTypeParameters(ctx), ctx)})
		args = append(args, ast.NewIdent(name))
	}
	target := sourceClassViewExpr(scope, owner, receiver, ctx)
	if target == nil {
		return nil
	}
	call := ast.Expr(&ast.CallExpr{Fun: &ast.SelectorExpr{X: target, Sel: ast.NewIdent(executionImplementationName(method, owner, ctx))}, Args: args})
	if _, erased := directOwnerInterfaceErasure(owner, method, ctx); erased && c.result != "void" {
		descriptor, known := javaSourceTypeDescriptorExpr(c.result, ctx)
		if !known {
			return nil
		}
		call = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(c.result, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{call, descriptor})
	}
	ft := &ast.FuncType{Params: params}
	var body []ast.Stmt
	if c.result == "void" {
		body = []ast.Stmt{&ast.ExprStmt{X: call}}
	} else {
		if !javaInferenceSameType(result, c.result, ctx) {
			converted, ok := convertJavaValue(call, result, c.result, ctx)
			if ok {
				call = converted
			} else {
				if !javaInferenceTypeAssignable(result, c.result, ctx) {
					return nil
				}
				descriptor, known := javaSourceTypeDescriptorExpr(c.result, ctx)
				if !known {
					return nil
				}
				call = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(c.result, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{call, descriptor})
			}
		}
		ft.Results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(c.result, inScopeTypeParameters(ctx), ctx)}}}
		body = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}
	}
	closure := &ast.FuncLit{Type: ft, Body: &ast.BlockStmt{List: body}}
	nativeArgs := []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent("ObjectInfo")}, closure}
	if len(c.arguments) == 0 {
		return stdjavaCall(ctx, "New"+c.family+"SourceView", nativeArgs...)
	}
	types := []ast.Expr{}
	for _, a := range c.arguments {
		types = append(types, javaTypeStringToGoTypeExpr(a, inScopeTypeParameters(ctx), ctx))
	}
	return stdjavaGenericCall(ctx, "New"+c.family+"SourceView", types, nativeArgs)
}
func nativeFunctionalObjectProjection(value ast.Expr, expected string, ctx Ctx) ast.Expr {
	if nativeFunctionalFamily(expected, ctx) == "" {
		return nil
	}
	id, ok := javaSourceTypeDescriptorExpr(expected, ctx)
	if !ok {
		return nil
	}
	typ := javaTypeStringToGoTypeExpr(expected, inScopeTypeParameters(ctx), ctx)
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{typ}, []ast.Expr{value, id})
}
func additionalNativeFunctionalRuntimeType(javaType string, args, params []string, ctx Ctx) (ast.Expr, bool) {
	family := nativeFunctionalFamily(javaType, ctx)
	if family != "BiFunction" && family != "Consumer" {
		return nil, false
	}
	c, ok := nativeFunctionalContract(family, args)
	if !ok {
		return nil, false
	}
	types := []ast.Expr{}
	for _, a := range c.arguments {
		types = append(types, javaTypeStringToGoTypeExpr(a, params, ctx))
	}
	return applyTypeArguments(stdjavaQualifiedExpr(family, ctx), types), true
}
func additionalNativeFunctionalAdapter(value ast.Expr, javaType string, executionAware bool, ctx Ctx) ast.Expr {
	f := nativeFunctionalFamily(javaType, ctx)
	if f != "BiFunction" && f != "Consumer" {
		return nil
	}
	_, args := parseJavaTypeString(javaType)
	c, ok := nativeFunctionalContract(f, args)
	if !ok {
		return nil
	}
	constructor := "New" + f + "FuncAdapter"
	if !executionAware {
		constructor = "NewPlain" + f + "FuncAdapter"
	}
	types := []ast.Expr{}
	for _, a := range c.arguments {
		types = append(types, javaTypeStringToGoTypeExpr(a, inScopeTypeParameters(ctx), ctx))
	}
	return stdjavaGenericCall(ctx, constructor, types, []ast.Expr{value})
}
func registerAdditionalNativeFunctionalViews() {
	for _, family := range []string{"BiFunction", "Consumer"} {
		family := family
		registerIntrinsicOwner("java.util.function."+family, true)
		method := "apply"
		arity := 2
		if family == "Consumer" {
			method = "accept"
			arity = 1
		}
		registerInstanceIntrinsic(family, method, func(receiver ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != arity {
				return nil
			}
			return stdjavaCall(ctx, "Call"+family+"Execution", append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
		})
	}
}

func nativeFunctionalAssignable(actual, expected string, ctx Ctx) bool {
	target := nativeFunctionalFamily(expected, ctx)
	if target == "" {
		return false
	}
	_, wanted := parseJavaTypeString(expected)
	var available []sourceNativeFunctionalContract
	if family := nativeFunctionalFamily(actual, ctx); family != "" {
		_, args := parseJavaTypeString(actual)
		if c, ok := nativeFunctionalContract(family, args); ok {
			available = append(available, c)
			if family == "UnaryOperator" {
				c, _ = nativeFunctionalContract("Function", []string{c.arguments[0], c.arguments[0]})
				available = append(available, c)
			}
			if family == "BinaryOperator" {
				c, _ = nativeFunctionalContract("BiFunction", []string{c.arguments[0], c.arguments[0], c.arguments[0]})
				available = append(available, c)
			}
		}
	} else {
		base, args := parseJavaTypeString(actual)
		scope := resolveClassScopeByQualifiedName(ctx, base)
		if scope == nil {
			return false
		}
		available = sourceNativeFunctionalContracts(scope, ctx)
		bindings := map[string]string{}
		for i, p := range scope.TypeParameters {
			if i < len(args) {
				bindings[p.Name] = args[i]
				bindings[p.EmittedName()] = args[i]
			}
		}
		for i := range available {
			for j, a := range available[i].arguments {
				available[i].arguments[j] = substituteJavaTypeParams(a, bindings)
			}
		}
	}
	for _, c := range available {
		if c.family != target {
			continue
		}
		if len(wanted) == 0 {
			return true
		}
		if len(wanted) != len(c.arguments) {
			continue
		}
		matches := true
		for i, w := range wanted {
			a := c.arguments[i]
			if w == "?" {
				continue
			}
			if bound, ok := strings.CutPrefix(w, "? extends "); ok {
				matches = matches && javaInferenceTypeAssignable(a, bound, ctx)
			} else if bound, ok := strings.CutPrefix(w, "? super "); ok {
				matches = matches && javaInferenceTypeAssignable(bound, a, ctx)
			} else {
				matches = matches && javaInferenceSameType(a, w, ctx)
			}
		}
		if matches {
			return true
		}
	}
	return false
}

// Parent arguments are resolved from the same declaration-owned edge closure
// used by nominal registration, including source interface/superclass bindings.
func resolvedNativeFunctionalArguments(javaType, family string, ctx Ctx) ([]string, bool) {
	var contracts []sourceNativeFunctionalContract
	if native := nativeFunctionalFamily(javaType, ctx); native != "" {
		_, arguments := parseJavaTypeString(javaType)
		c, ok := nativeFunctionalContract(native, arguments)
		if !ok {
			return nil, false
		}
		contracts = append(contracts, c)
		if native == "UnaryOperator" {
			parent, _ := nativeFunctionalContract("Function", []string{c.arguments[0], c.arguments[0]})
			contracts = append(contracts, parent)
		}
		if native == "BinaryOperator" {
			parent, _ := nativeFunctionalContract("BiFunction", []string{c.arguments[0], c.arguments[0], c.arguments[0]})
			contracts = append(contracts, parent)
		}
	} else {
		base, arguments := parseJavaTypeString(javaType)
		scope := resolveClassScopeByQualifiedName(ctx, base)
		if scope == nil {
			return nil, false
		}
		contracts = sourceNativeFunctionalContracts(scope, ctx)
		bindings := map[string]string{}
		for i, parameter := range scope.TypeParameters {
			if i < len(arguments) {
				bindings[parameter.Name] = arguments[i]
				bindings[parameter.EmittedName()] = arguments[i]
			}
		}
		for i := range contracts {
			mapped := make([]string, len(contracts[i].arguments))
			for j, arg := range contracts[i].arguments {
				mapped[j] = substituteJavaTypeParams(arg, bindings)
			}
			contracts[i].arguments = mapped
		}
	}
	for _, contract := range contracts {
		if contract.family == family {
			return append([]string(nil), contract.arguments...), true
		}
	}
	return nil, false
}
