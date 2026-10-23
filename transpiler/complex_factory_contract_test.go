package transpiler

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strconv"
	"testing"
)

// This contract follows the original program's actual declaration and local
// bindings. It deliberately does not contain an expected generated function.
type complexFactoryContractError struct{ why string }

func (e *complexFactoryContractError) Error() string { return "COMPLEX_FACTORY_CONTRACT: " + e.why }
func factoryReject(why string) error                 { return &complexFactoryContractError{why} }
func factoryText(node ast.Node) string {
	var out bytes.Buffer
	if node == nil || format.Node(&out, token.NewFileSet(), node) != nil {
		return ""
	}
	return out.String()
}
func factoryIdent(expr ast.Expr) *ast.Ident { id, _ := expr.(*ast.Ident); return id }
func factorySame(expr ast.Expr, binding *ast.Ident) bool {
	id := factoryIdent(expr)
	return id != nil && binding != nil && binding.Obj != nil && id.Obj == binding.Obj
}
func factoryCall(expr ast.Expr) *ast.CallExpr { call, _ := expr.(*ast.CallExpr); return call }
func factoryReturn(body *ast.BlockStmt) ast.Expr {
	if body == nil || len(body.List) != 1 {
		return nil
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil
	}
	return ret.Results[0]
}
func factoryParam(fields *ast.FieldList, index int) *ast.Ident {
	if fields == nil || index >= len(fields.List) || len(fields.List[index].Names) != 1 {
		return nil
	}
	return fields.List[index].Names[0]
}
func factoryTypes(fields *ast.FieldList, expected ...string) bool {
	if fields == nil {
		return len(expected) == 0
	}
	if len(fields.List) != len(expected) {
		return false
	}
	for i, field := range fields.List {
		if len(field.Names) > 1 || factoryText(field.Type) != expected[i] {
			return false
		}
	}
	return true
}
func factoryGenericBase(expr ast.Expr, argument string) *ast.Ident {
	index, ok := expr.(*ast.IndexExpr)
	if !ok || factoryText(index.Index) != argument {
		return nil
	}
	return factoryIdent(index.X)
}
func factorySelector(call *ast.CallExpr, receiver *ast.Ident, method string, arguments ...*ast.Ident) bool {
	if call == nil || len(call.Args) != len(arguments) {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != method || !factorySame(selector.X, receiver) {
		return false
	}
	for i, argument := range arguments {
		if !factorySame(call.Args[i], argument) {
			return false
		}
	}
	return true
}
func factorySourceJava(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile("complex_program_integration_test.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "original-test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "TestComplexProgram_GenericFactoryAndDiamondInLambda" {
			continue
		}
		assignment, ok := fn.Body.List[0].(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) != 1 {
			t.Fatal("original Java initializer changed")
		}
		literal, ok := assignment.Rhs[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			t.Fatal("original Java literal missing")
		}
		java, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		return java
	}
	t.Fatal("original factory test missing")
	return ""
}

type factoryDeclarations struct {
	functions map[string]*ast.FuncDecl
	types     map[string]*ast.TypeSpec
	positions *token.FileSet
	file      *ast.File
}

func factoryParse(out string) (*ast.File, factoryDeclarations, error) {
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "generated.go", out, 0)
	decls := factoryDeclarations{functions: map[string]*ast.FuncDecl{}, types: map[string]*ast.TypeSpec{}, positions: positions, file: file}
	if err != nil {
		return nil, decls, err
	}
	imports := 0
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, decls, err
		}
		if path == "github.com/NickyBoy89/java2go/stdjava" {
			if spec.Name != nil && spec.Name.Name != "stdjava" {
				return nil, decls, factoryReject("runtime import binding changed")
			}
			imports++
		} else if spec.Name != nil && spec.Name.Name == "stdjava" {
			return nil, decls, factoryReject("runtime import spoofed")
		}
	}
	if imports != 1 {
		return nil, decls, factoryReject("canonical runtime import missing")
	}
	shadow := false
	ast.Inspect(file, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == "stdjava" && id.Obj != nil {
			shadow = true
		}
		return true
	})
	if shadow {
		return nil, decls, factoryReject("runtime import shadowed")
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			key := declaration.Name.Name
			if declaration.Recv != nil {
				if len(declaration.Recv.List) != 1 {
					return nil, decls, factoryReject("invalid receiver")
				}
				star, ok := declaration.Recv.List[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				index, ok := star.X.(*ast.IndexExpr)
				if !ok {
					continue
				}
				owner := factoryIdent(index.X)
				if owner == nil {
					continue
				}
				key = owner.Name + "." + key
			}
			if decls.functions[key] != nil {
				return nil, decls, factoryReject("duplicate function " + key)
			}
			decls.functions[key] = declaration
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok {
					if decls.types[typ.Name.Name] != nil {
						return nil, decls, factoryReject("duplicate type")
					}
					decls.types[typ.Name.Name] = typ
				}
			}
		}
	}
	return file, decls, nil
}
func factoryOwnCall(call *ast.CallExpr, declaration *ast.FuncDecl, argument string) bool {
	if call == nil || declaration == nil {
		return false
	}
	var id *ast.Ident
	if argument != "" {
		id = factoryGenericBase(call.Fun, argument)
	} else {
		id = factoryIdent(call.Fun)
	}
	return id != nil && id.Obj != nil && id.Obj.Decl == declaration
}
func factoryBinder(fields *ast.FieldList) *ast.Ident {
	if !factoryTypes(fields, "any") {
		return nil
	}
	return factoryParam(fields, 0)
}
func factoryBoundTypes(node ast.Node, binder *ast.Ident) bool {
	if binder == nil || binder.Obj == nil {
		return false
	}
	ok := true
	ast.Inspect(node, func(node ast.Node) bool {
		if id, yes := node.(*ast.Ident); yes && id.Name == binder.Name && id != binder && id.Obj != binder.Obj {
			ok = false
		}
		return true
	})
	return ok
}

// These two opaque runtime types only let go/types resolve the untouched
// signature's import. No runtime body, field service or behavior is modeled.
// The original generic declaration, receiver, parameters and result AST nodes
// all enter the checker unchanged, so it must discover their binding itself.
type factorySignatureImporter struct{ runtime *types.Package }

func (i factorySignatureImporter) Import(path string) (*types.Package, error) {
	if path != "github.com/NickyBoy89/java2go/stdjava" {
		return nil, factoryReject("unexpected signature import")
	}
	return i.runtime, nil
}

func factoryReceiverBinder(fn *ast.FuncDecl) *ast.Ident {
	if fn == nil || fn.Recv == nil || len(fn.Recv.List) != 1 {
		return nil
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return nil
	}
	index, ok := star.X.(*ast.IndexExpr)
	if !ok {
		return nil
	}
	return factoryIdent(index.Index)
}

func factoryReceiverBinding(declarations factoryDeclarations, fn *ast.FuncDecl, binder *ast.Ident) (*types.Info, *types.TypeName, error) {
	owner := declarations.types["FactoryFuncAdapter"]
	if binder == nil || binder != factoryReceiverBinder(fn) || owner == nil || declarations.file == nil || declarations.positions == nil {
		return nil, nil, factoryReject("foreign receiver declaration cannot attest result")
	}
	runtime := types.NewPackage("github.com/NickyBoy89/java2go/stdjava", "stdjava")
	for _, name := range []string{"Execution", "JavaString"} {
		typ := types.NewTypeName(token.NoPos, runtime, name, nil)
		types.NewNamed(typ, types.NewStruct(nil, nil), nil)
		runtime.Scope().Insert(typ)
	}
	runtime.MarkComplete()
	var imported *ast.ImportSpec
	for _, spec := range declarations.file.Imports {
		if path, _ := strconv.Unquote(spec.Path.Value); path == runtime.Path() {
			imported = spec
		}
	}
	if imported == nil {
		return nil, nil, factoryReject("actual signature runtime import missing")
	}
	// Only wrapper containers are new. Every relevant declaration/type node is
	// the original pointer from the actual printed generated program.
	projection := &ast.File{Package: declarations.file.Package, Name: declarations.file.Name,
		FileStart: declarations.file.FileStart, FileEnd: declarations.file.FileEnd,
		Imports: []*ast.ImportSpec{imported}, Decls: []ast.Decl{
			&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{imported}},
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{owner}}, fn,
		}}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	config := types.Config{Importer: factorySignatureImporter{runtime}, IgnoreFuncBodies: true, DisableUnusedImportCheck: true}
	if _, err := config.Check("factory.signature", declarations.positions, []*ast.File{projection}, info); err != nil {
		return nil, nil, factoryReject("actual receiver signature does not type check: " + err.Error())
	}
	object, ok := info.Defs[binder].(*types.TypeName)
	if !ok {
		return nil, nil, factoryReject("actual receiver declaration has no resolved type object")
	}
	method, ok := info.Defs[fn.Name].(*types.Func)
	if !ok {
		return nil, nil, factoryReject("actual receiver method has no resolved object")
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.RecvTypeParams().Len() != 1 || signature.RecvTypeParams().At(0).Obj() != object || signature.Results().Len() != 1 {
		return nil, nil, factoryReject("result must use actual method receiver type parameter")
	}
	result := factoryIdent(fn.Type.Results.List[0].Type)
	if result == nil || info.Uses[result] != object || signature.Results().At(0).Type() != signature.RecvTypeParams().At(0) {
		return nil, nil, factoryReject("actual result node is bound to another declaration")
	}
	index := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.IndexExpr)
	base := factoryIdent(index.X)
	if base == nil || info.Uses[base] != info.Defs[owner.Name] {
		return nil, nil, factoryReject("receiver must use actual adapter declaration")
	}
	return info, object, nil
}
func factoryInterface(typ *ast.TypeSpec, execution bool) bool {
	if typ == nil {
		return false
	}
	binder := factoryBinder(typ.TypeParams)
	iface, ok := typ.Type.(*ast.InterfaceType)
	if !ok || binder == nil || len(iface.Methods.List) != 1 {
		return false
	}
	method := iface.Methods.List[0]
	fn, ok := method.Type.(*ast.FuncType)
	name := "Create"
	args := []string{"*stdjava.JavaString"}
	if execution {
		name += "Java2goExecution"
		args = append([]string{"*stdjava.Execution"}, args...)
	}
	return ok && len(method.Names) == 1 && method.Names[0].Name == name && factoryTypes(fn.Params, args...) && factoryTypes(fn.Results, binder.Name) && factoryBoundTypes(typ.Type, binder)
}

func complexFactoryContract(out string) error {
	file, decls, err := factoryParse(out)
	if err != nil {
		return err
	}
	ownedTypes := true
	ast.Inspect(file, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok {
			switch id.Name {
			case "Factory", "FactoryJava2goExecution", "FactoryFuncAdapter", "Box":
				if id.Obj == nil || id.Obj.Decl != decls.types[id.Name] {
					ownedTypes = false
				}
			}
		}
		return true
	})
	if !ownedTypes {
		return factoryReject("factory or Box type borrowed another declaration")
	}
	if !factoryInterface(decls.types["Factory"], false) || !factoryInterface(decls.types["FactoryJava2goExecution"], true) {
		return factoryReject("Factory owned generic signature changed")
	}
	adapter := decls.types["FactoryFuncAdapter"]
	if adapter == nil {
		return factoryReject("adapter missing")
	}
	binder := factoryBinder(adapter.TypeParams)
	structure, ok := adapter.Type.(*ast.StructType)
	if binder == nil || !ok || len(structure.Fields.List) != 1 {
		return factoryReject("adapter generic storage changed")
	}
	field := structure.Fields.List[0]
	callbackType, ok := field.Type.(*ast.FuncType)
	if !ok || len(field.Names) != 1 || field.Names[0].Name != "fn" || !factoryTypes(callbackType.Params, "*stdjava.Execution", "*stdjava.JavaString") || !factoryTypes(callbackType.Results, binder.Name) || !factoryBoundTypes(adapter.Type, binder) {
		return factoryReject("adapter callback binder changed")
	}
	constructor := decls.functions["NewFactoryFuncAdapterJava2goExecution"]
	if constructor == nil || constructor.Recv != nil {
		return factoryReject("actual adapter constructor missing")
	}
	binder = factoryBinder(constructor.Type.TypeParams)
	if binder == nil || constructor.Type.Params == nil || len(constructor.Type.Params.List) != 1 || !factoryTypes(constructor.Type.Results, "Factory["+binder.Name+"]") || !factoryBoundTypes(constructor.Type, binder) {
		return factoryReject("adapter constructor type binding changed")
	}
	// The callback's parameter names are irrelevant; only their types matter.
	callback, ok := constructor.Type.Params.List[0].Type.(*ast.FuncType)
	if !ok || !factoryTypes(callback.Params, "*stdjava.Execution", "*stdjava.JavaString") || !factoryTypes(callback.Results, binder.Name) {
		return factoryReject("adapter callback signature changed")
	}
	address, ok := factoryReturn(constructor.Body).(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return factoryReject("adapter must allocate a fresh identity without invoking callback")
	}
	literal, ok := address.X.(*ast.CompositeLit)
	if !ok || len(literal.Elts) != 1 {
		return factoryReject("adapter callback storage changed")
	}
	adapterType := factoryGenericBase(literal.Type, binder.Name)
	pair, ok := literal.Elts[0].(*ast.KeyValueExpr)
	if adapterType == nil || adapterType.Obj == nil || adapterType.Obj.Decl != adapter || !ok || factoryText(pair.Key) != "fn" || !factorySame(pair.Value, factoryParam(constructor.Type.Params, 0)) {
		return factoryReject("actual adapter must retain original callback")
	}
	invoke := decls.functions["FactoryFuncAdapter.CreateJava2goExecution"]
	if invoke == nil || !factoryTypes(invoke.Type.Params, "*stdjava.Execution", "*stdjava.JavaString") {
		return factoryReject("adapter invocation missing")
	}
	receiverType, ok := invoke.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return factoryReject("adapter receiver type changed")
	}
	receiverIndex, ok := receiverType.X.(*ast.IndexExpr)
	if !ok {
		return factoryReject("adapter receiver generic binder missing")
	}
	receiverBinder := factoryIdent(receiverIndex.Index)
	if receiverBinder == nil || !factoryTypes(invoke.Type.Results, receiverBinder.Name) {
		return factoryReject("adapter result must use its own receiver binder")
	}
	if _, _, err := factoryReceiverBinding(decls, invoke, receiverBinder); err != nil {
		return err
	}
	invokeCall := factoryCall(factoryReturn(invoke.Body))
	if invokeCall == nil {
		return factoryReject("adapter invokes callback exactly once")
	}
	fnSelector, ok := invokeCall.Fun.(*ast.SelectorExpr)
	if !ok || fnSelector.Sel.Name != "fn" || !factorySame(fnSelector.X, factoryParam(invoke.Recv, 0)) || len(invokeCall.Args) != 2 || !factorySame(invokeCall.Args[0], factoryParam(invoke.Type.Params, 0)) || !factorySame(invokeCall.Args[1], factoryParam(invoke.Type.Params, 1)) {
		return factoryReject("adapter must use invocation execution and seed")
	}
	const boxType = "*Box[*stdjava.JavaString]"
	build := decls.functions["Build"]
	buildExecution := decls.functions["BuildJava2goExecution"]
	if build == nil || buildExecution == nil || build.Recv != nil || buildExecution.Recv != nil || build.Type.TypeParams != nil || buildExecution.Type.TypeParams != nil || !factoryTypes(build.Type.Params, "Factory["+boxType+"]") || !factoryTypes(build.Type.Results, boxType) || !factoryTypes(buildExecution.Type.Params, "*stdjava.Execution", "Factory["+boxType+"]") || !factoryTypes(buildExecution.Type.Results, boxType) {
		return factoryReject("Build nested generic ABI changed")
	}
	boundary := factoryCall(factoryReturn(build.Body))
	if !factoryOwnCall(boundary, buildExecution, "") || len(boundary.Args) != 2 || factoryText(boundary.Args[0]) != "stdjava.NewExecution()" || !factorySame(boundary.Args[1], factoryParam(build.Type.Params, 0)) {
		return factoryReject("Build boundary lost factory binding")
	}
	if err := factoryBuildFlow(buildExecution); err != nil {
		return err
	}
	main := decls.functions["MainJava2goExecution"]
	if main == nil || main.Recv != nil || main.Type.TypeParams != nil || !factoryTypes(main.Type.Params, "*stdjava.Execution") {
		return factoryReject("actual Main missing")
	}
	var call *ast.CallExpr
	var box *ast.Ident
	for _, statement := range main.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) != 1 || len(assignment.Lhs) != 1 {
			continue
		}
		candidate := factoryCall(assignment.Rhs[0])
		if !factoryOwnCall(candidate, buildExecution, "") {
			continue
		}
		if call != nil {
			return factoryReject("Build result duplicated")
		}
		call, box = candidate, factoryIdent(assignment.Lhs[0])
	}
	if call == nil || box == nil || len(call.Args) != 2 || !factorySame(call.Args[0], factoryParam(main.Type.Params, 0)) {
		return factoryReject("Main must call actual Build with current execution")
	}
	adapt := factoryCall(call.Args[1])
	if !factoryOwnCall(adapt, constructor, boxType) || len(adapt.Args) != 1 {
		return factoryReject("actual Factory adapter generic argument changed")
	}
	lambda, ok := adapt.Args[0].(*ast.FuncLit)
	if !ok || !factoryTypes(lambda.Type.Params, "*stdjava.Execution", "*stdjava.JavaString") || !factoryTypes(lambda.Type.Results, boxType) {
		return factoryReject("lambda SAM type changed")
	}
	newBox := decls.functions["NewBoxJava2goExecution"]
	creation := factoryCall(factoryReturn(lambda.Body))
	if !factoryOwnCall(creation, newBox, "*stdjava.JavaString") || len(creation.Args) != 2 || !factorySame(creation.Args[0], factoryParam(lambda.Type.Params, 0)) || !factorySame(creation.Args[1], factoryParam(lambda.Type.Params, 1)) {
		return factoryReject("lazy diamond constructor must use callback execution and bound seed")
	}
	count := 0
	ast.Inspect(main.Body, func(node ast.Node) bool {
		if c, yes := node.(*ast.CallExpr); yes && factoryOwnCall(c, newBox, "*stdjava.JavaString") {
			count++
		}
		return true
	})
	if count != 1 {
		return factoryReject("constructor must exist only inside callback")
	}
	if newBox == nil || factoryBinder(newBox.Type.TypeParams) == nil {
		return factoryReject("generic Box constructor missing")
	}
	if boxDeclaration := decls.types["Box"]; boxDeclaration == nil || factoryBinder(boxDeclaration.TypeParams) == nil {
		return factoryReject("actual Box generic declaration missing")
	}
	boxBinder := factoryBinder(newBox.Type.TypeParams)
	if !factoryTypes(newBox.Type.Params, "*stdjava.Execution", boxBinder.Name) || !factoryTypes(newBox.Type.Results, "*Box["+boxBinder.Name+"]") || !factoryBoundTypes(newBox.Type, boxBinder) {
		return factoryReject("Box constructor generic binder changed")
	}
	// Preserve the observable consumer of the actual returned Box.
	uses := 0
	ast.Inspect(main.Body, func(node ast.Node) bool {
		c, yes := node.(*ast.CallExpr)
		if !yes || factoryText(c.Fun) != "stdjava.JavaPrintlnStrings" || len(c.Args) != 1 {
			return true
		}
		text := factoryCall(c.Args[0])
		if text == nil || factoryText(text.Fun) != "stdjava.JavaStringTextOperandExecution" || len(text.Args) != 2 || !factorySame(text.Args[0], factoryParam(main.Type.Params, 0)) {
			return true
		}
		if factorySelector(factoryCall(text.Args[1]), box, "GetJava2goExecution", factoryParam(main.Type.Params, 0)) {
			uses++
		}
		return true
	})
	if uses != 1 {
		return factoryReject("actual factory result must reach get and print")
	}
	return nil
}

func factoryBuildFlow(fn *ast.FuncDecl) error {
	outer := factoryCall(factoryReturn(fn.Body))
	if outer == nil || len(outer.Args) != 0 {
		return factoryReject("Build must stage invocation")
	}
	closure, ok := outer.Fun.(*ast.FuncLit)
	if !ok || len(closure.Body.List) != 6 {
		return factoryReject("Build staging flow changed")
	}
	statements := closure.Body.List
	stage := func(statement ast.Stmt) (*ast.Ident, ast.Expr) {
		a, ok := statement.(*ast.AssignStmt)
		if !ok || a.Tok != token.DEFINE || len(a.Lhs) != 1 || len(a.Rhs) != 1 {
			return nil, nil
		}
		return factoryIdent(a.Lhs[0]), a.Rhs[0]
	}
	receiver, value := stage(statements[0])
	seed, seedValue := stage(statements[1])
	if receiver == nil || seed == nil || !factorySame(value, factoryParam(fn.Type.Params, 1)) || factoryText(seedValue) != "stdjava.JavaStringLiteralUTF16([]uint16{115, 101, 101, 100})" {
		return factoryReject("Build must stage actual factory and original UTF16 seed once")
	}
	check, ok := statements[2].(*ast.ExprStmt)
	if !ok {
		return factoryReject("factory null check missing")
	}
	require := factoryCall(check.X)
	if require == nil || factoryText(require.Fun) != "stdjava.RequireInvocationReceiver" || len(require.Args) != 2 || !factorySame(require.Args[0], receiver) {
		return factoryReject("null check must use staged factory")
	}
	assertion, ok := statements[3].(*ast.AssignStmt)
	if !ok || assertion.Tok != token.DEFINE || len(assertion.Lhs) != 2 || len(assertion.Rhs) != 1 {
		return factoryReject("execution dispatch proof missing")
	}
	asserted, ok := assertion.Rhs[0].(*ast.TypeAssertExpr)
	if !ok || factoryText(asserted.Type) != "FactoryJava2goExecution[*Box[*stdjava.JavaString]]" {
		return factoryReject("execution dispatch generic owner changed")
	}
	conversion := factoryCall(asserted.X)
	if conversion == nil || len(conversion.Args) != 1 || !factorySame(conversion.Args[0], receiver) {
		return factoryReject("execution dispatch must use actual staged factory")
	}
	view, ok := conversion.Fun.(*ast.InterfaceType)
	if !ok || view.Methods.NumFields() != 0 {
		return factoryReject("execution dispatch must inspect actual factory value")
	}
	branch, ok := statements[4].(*ast.IfStmt)
	if !ok || branch.Else != nil || !factorySame(branch.Cond, factoryIdent(assertion.Lhs[1])) || !factorySelector(factoryCall(factoryReturn(branch.Body)), factoryIdent(assertion.Lhs[0]), "CreateJava2goExecution", factoryParam(fn.Type.Params, 0), seed) {
		return factoryReject("factory callback must receive invoking execution and seed")
	}
	fallback, ok := statements[5].(*ast.ReturnStmt)
	if !ok || len(fallback.Results) != 1 || !factorySelector(factoryCall(fallback.Results[0]), receiver, "Create", seed) {
		return factoryReject("fallback must invoke same factory and seed")
	}
	return nil
}
func assertComplexFactoryContract(t *testing.T, out string) {
	t.Helper()
	if err := complexFactoryContract(out); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestComplexFactoryContractAcceptsRenamedLocalAndReceiverBindings(t *testing.T) {
	original := renderGoFileFromJava(t, factorySourceJava(t))
	file, declarations, err := factoryParse(original)
	if err != nil {
		t.Fatal(err)
	}
	rename := func(binding *ast.Ident, name string) {
		if binding == nil || binding.Obj == nil {
			t.Fatal("positive binding missing")
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok && id.Obj == binding.Obj {
				id.Name = name
			}
			return true
		})
	}
	main := declarations.functions["MainJava2goExecution"]
	rename(factoryParam(main.Type.Params, 0), "creationContext")
	build := declarations.functions["BuildJava2goExecution"]
	rename(factoryParam(build.Type.Params, 0), "builderContext")
	rename(factoryParam(build.Type.Params, 1), "chosenFactory")
	invoke := declarations.functions["FactoryFuncAdapter.CreateJava2goExecution"]
	receiverBinder := factoryReceiverBinder(invoke)
	info, binding, err := factoryReceiverBinding(declarations, invoke, receiverBinder)
	if err != nil {
		t.Fatal(err)
	}
	result := factoryIdent(invoke.Type.Results.List[0].Type)
	if result == nil || info.Uses[result] != binding || info.Defs[receiverBinder] != binding {
		t.Fatal("genuine result node did not resolve to actual receiver declaration before rename")
	}
	for id, object := range info.Uses {
		if object == binding {
			id.Name = "InvokedResult"
		}
	}
	receiverBinder.Name = "InvokedResult"
	if result.Name != "InvokedResult" || result != factoryIdent(invoke.Type.Results.List[0].Type) {
		t.Fatal("alpha fixture failed to rename the actual original result node")
	}
	ast.Inspect(main.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !factoryOwnCall(call, declarations.functions["NewFactoryFuncAdapterJava2goExecution"], "*Box[*stdjava.JavaString]") {
			return true
		}
		lambda := call.Args[0].(*ast.FuncLit)
		rename(factoryParam(lambda.Type.Params, 0), "invocationContext")
		rename(factoryParam(lambda.Type.Params, 1), "inputReference")
		return false
	})
	var buffer bytes.Buffer
	if err := format.Node(&buffer, token.NewFileSet(), file); err != nil {
		t.Fatal(err)
	}
	assertComplexFactoryContract(t, buffer.String())
}

// Both method receivers use the same spelling T, but own distinct declaration
// identifiers. A same-spelling binder from the sibling method cannot attest
// CreateJava2goExecution's result. This control stays in a detached parsed AST.
func TestComplexFactoryReceiverBinderRejectsForeignDeclaration(t *testing.T) {
	_, declarations, err := factoryParse(renderGoFileFromJava(t, factorySourceJava(t)))
	if err != nil {
		t.Fatal(err)
	}
	invoke := declarations.functions["FactoryFuncAdapter.CreateJava2goExecution"]
	plain := declarations.functions["FactoryFuncAdapter.Create"]
	binder := func(fn *ast.FuncDecl) *ast.Ident {
		return fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.IndexExpr).Index.(*ast.Ident)
	}
	actual, foreign := binder(invoke), binder(plain)
	if actual == foreign || actual.Name != foreign.Name {
		t.Fatal("same-spelling independent receiver declarations missing")
	}
	info, binding, err := factoryReceiverBinding(declarations, invoke, actual)
	if err != nil {
		t.Fatal(err)
	}
	result := factoryIdent(invoke.Type.Results.List[0].Type)
	if result == nil || info.Uses[result] != binding || info.Defs[actual] != binding {
		t.Fatal("actual result/declaration identity lost")
	}
	var parserDecl any
	if result.Obj != nil {
		parserDecl = result.Obj.Decl
	}
	t.Logf("actual printed AST parser receiverObjNil=%t resultObjNil=%t resultDecl=%T resultDeclIsReceiver=%t; go/types actual declaration/result identity proved", actual.Obj == nil, result.Obj == nil, parserDecl, parserDecl == actual)
	if _, _, err := factoryReceiverBinding(declarations, invoke, foreign); err == nil {
		t.Fatal("foreign same-spelling receiver declaration attested actual result")
	}
}

// Mutate detached ASTs only. None of these strings is built or used as runtime
// evidence, and every rejection must be a contract error after successful parse.
func TestComplexFactoryContractRejectsBrokenBindings(t *testing.T) {
	original := renderGoFileFromJava(t, factorySourceJava(t))
	assertComplexFactoryContract(t, original)
	mutations := []struct {
		name   string
		change func(*ast.File, factoryDeclarations)
	}{
		{"host-string-carrier", func(_ *ast.File, d factoryDeclarations) {
			d.functions["Build"].Type.Results.List[0].Type = ast.NewIdent("string")
		}},
		{"wrong-nested-generic", func(_ *ast.File, d factoryDeclarations) {
			d.functions["BuildJava2goExecution"].Type.Params.List[1].Type = ast.NewIdent("any")
		}},
		{"foreign-adapter-callback", func(_ *ast.File, d factoryDeclarations) {
			fn := d.functions["NewFactoryFuncAdapterJava2goExecution"]
			literal := factoryReturn(fn.Body).(*ast.UnaryExpr).X.(*ast.CompositeLit)
			literal.Elts[0].(*ast.KeyValueExpr).Value = ast.NewIdent("nil")
		}},
		{"fresh-callback-execution", func(_ *ast.File, d factoryDeclarations) {
			fn := d.functions["FactoryFuncAdapter.CreateJava2goExecution"]
			factoryCall(factoryReturn(fn.Body)).Args[0] = &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("stdjava"), Sel: ast.NewIdent("NewExecution")}}
		}},
		{"shadow-receiver-binder", func(_ *ast.File, d factoryDeclarations) {
			fn := d.functions["FactoryFuncAdapter.CreateJava2goExecution"]
			binder := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.IndexExpr).Index.(*ast.Ident)
			fn.Type.Params.List[0].Names[0] = ast.NewIdent(binder.Name)
			factoryCall(factoryReturn(fn.Body)).Args[0] = ast.NewIdent(binder.Name)
		}},
		{"foreign-factory-receiver", func(_ *ast.File, d factoryDeclarations) {
			closure := factoryCall(factoryReturn(d.functions["BuildJava2goExecution"].Body)).Fun.(*ast.FuncLit)
			closure.Body.List[0].(*ast.AssignStmt).Rhs[0] = ast.NewIdent("nil")
		}},
		{"wrong-seed-literal", func(_ *ast.File, d factoryDeclarations) {
			closure := factoryCall(factoryReturn(d.functions["BuildJava2goExecution"].Body)).Fun.(*ast.FuncLit)
			closure.Body.List[1].(*ast.AssignStmt).Rhs[0] = ast.NewIdent("nil")
		}},
		{"dispatch-fresh-execution", func(_ *ast.File, d factoryDeclarations) {
			closure := factoryCall(factoryReturn(d.functions["BuildJava2goExecution"].Body)).Fun.(*ast.FuncLit)
			branch := closure.Body.List[4].(*ast.IfStmt)
			factoryCall(factoryReturn(branch.Body)).Args[0] = &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("stdjava"), Sel: ast.NewIdent("NewExecution")}}
		}},
		{"runtime-import-spoof", func(f *ast.File, _ factoryDeclarations) {
			for _, s := range f.Imports {
				if path, _ := strconv.Unquote(s.Path.Value); path == "github.com/NickyBoy89/java2go/stdjava" {
					s.Path.Value = strconv.Quote("example.invalid/stdjava")
					s.Name = ast.NewIdent("stdjava")
				}
			}
		}},
		{"donated-correct-body", func(f *ast.File, d factoryDeclarations) {
			fn := d.functions["BuildJava2goExecution"]
			copy := *fn
			copy.Name = ast.NewIdent("DonatedCorrectBuild")
			f.Decls = append(f.Decls, &copy)
			fn.Body = &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}
		}},
		{"eager-constructor-before-build", func(_ *ast.File, d factoryDeclarations) {
			fn := d.functions["MainJava2goExecution"]
			var eager *ast.CallExpr
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if ok && factoryOwnCall(c, d.functions["NewBoxJava2goExecution"], "*stdjava.JavaString") {
					copy := *c
					copy.Args = []ast.Expr{ast.NewIdent("nil"), ast.NewIdent("nil")}
					eager = &copy
					return false
				}
				return true
			})
			if eager == nil {
				panic("positive constructor missing")
			}
			fn.Body.List = append([]ast.Stmt{&ast.ExprStmt{X: eager}}, fn.Body.List...)
		}},
		{"runtime-shadow", func(_ *ast.File, d factoryDeclarations) {
			fn := d.functions["MainJava2goExecution"]
			fn.Body.List = append([]ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("stdjava")}, Tok: token.DEFINE, Rhs: []ast.Expr{ast.NewIdent("nil")}}}, fn.Body.List...)
		}},
	}
	for _, item := range []struct {
		name   string
		change func(*ast.CallExpr, *ast.FuncLit)
	}{
		{"callback-seed-replaced", func(c *ast.CallExpr, _ *ast.FuncLit) { c.Args[1] = ast.NewIdent("nil") }},
		{"callback-current-execution-replaced", func(c *ast.CallExpr, _ *ast.FuncLit) {
			c.Args[0] = &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("stdjava"), Sel: ast.NewIdent("NewExecution")}}
		}},
		{"diamond-erased-to-any", func(c *ast.CallExpr, _ *ast.FuncLit) { c.Fun.(*ast.IndexExpr).Index = ast.NewIdent("any") }},
		{"foreign-constructor-owner", func(c *ast.CallExpr, _ *ast.FuncLit) { c.Fun.(*ast.IndexExpr).X = ast.NewIdent("NewBox") }},
		{"duplicate-constructor-inside-callback", func(c *ast.CallExpr, l *ast.FuncLit) {
			l.Body.List = append([]ast.Stmt{&ast.ExprStmt{X: c}}, l.Body.List...)
		}},
		{"captures-creation-execution", func(_ *ast.CallExpr, l *ast.FuncLit) {
			l.Type.Params.List[0].Names[0] = ast.NewIdent("callbackExecutionUnused")
		}},
	} {
		item := item
		mutations = append(mutations, struct {
			name   string
			change func(*ast.File, factoryDeclarations)
		}{item.name, func(_ *ast.File, d factoryDeclarations) {
			fn := d.functions["MainJava2goExecution"]
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok || !factoryOwnCall(c, d.functions["NewFactoryFuncAdapterJava2goExecution"], "*Box[*stdjava.JavaString]") {
					return true
				}
				l := c.Args[0].(*ast.FuncLit)
				item.change(factoryCall(factoryReturn(l.Body)), l)
				return false
			})
		}})
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			file, declarations, err := factoryParse(original)
			if err != nil {
				t.Fatal(err)
			}
			mutation.change(file, declarations)
			var buffer bytes.Buffer
			if err := format.Node(&buffer, token.NewFileSet(), file); err != nil {
				t.Fatal(err)
			}
			mutant := buffer.String()
			if mutant == original {
				t.Fatal("mutation did not change detached output")
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "detached-negative.go", mutant, 0); err != nil {
				t.Fatalf("invalid detached mutant: %v", err)
			}
			var rejection *complexFactoryContractError
			if err := complexFactoryContract(mutant); !errors.As(err, &rejection) {
				t.Fatalf("mutant escaped contract-specific rejection: %v", err)
			}
		})
	}
}
