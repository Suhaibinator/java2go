package transpiler

import (
	"go/ast"
	"strconv"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func anonymousFunctional209Call(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.IndexExpr:
		return anonymousFunctional209Call(value.X)
	case *ast.IndexListExpr:
		return anonymousFunctional209Call(value.X)
	case *ast.SelectorExpr:
		return value.Sel.Name
	case *ast.Ident:
		return value.Name
	}
	return ""
}

func anonymousFunctional209Calls(node ast.Node, name string) int {
	count := 0
	ast.Inspect(node, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && anonymousFunctional209Call(call.Fun) == name {
			count++
		}
		return true
	})
	return count
}

func anonymousFunctional209Lower(t *testing.T, source, expected string) (Ctx, ast.Expr, []ast.Decl, *anonymousClassInfo) {
	t.Helper()
	h := setupParseHelper(t, source)
	ctx := h.Ctx.Clone()
	ctx.currentClass = ctx.currentFile.FindClassScope("Main")
	if ctx.currentClass == nil {
		t.Fatal("actual Main declaration missing")
	}
	methods := ctx.currentClass.FindMethod().ByOriginalName("make")
	if len(methods) != 1 {
		t.Fatal("actual make declaration missing")
	}
	ctx.localScope = methods[0]
	ctx.className = ctx.currentClass.Class.Name
	var declarations []ast.Decl
	var counter int
	ctx.hoistedDecls = &declarations
	ctx.anonClassCounter = &counter
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.executionContextName = "execution"
	creation := findNode(h.File.Ast, "object_creation_expression")
	if creation == nil {
		t.Fatal("actual allocation missing")
	}
	ctx.expectedType, ctx.expectedTypeRoot = expected, creation
	originalClass, originalMethod := ctx.currentClass, ctx.localScope
	expression := lowerAnonymousClassToStruct(creation, creation.ChildByFieldName("type"), objectCreationClassBody(creation), h.File.Source, ctx)
	key, _ := anonymousClassSourceKey(creation)
	info := ctx.anonymousClasses[key]
	if expression == nil || info == nil || info.scope == nil || info.scope.Class == nil || info.scope.Class.DeclarationNode == nil || !info.scope.Class.DeclarationNode.Equal(creation) {
		t.Fatal("actual allocation's synthesized source scope missing")
	}
	if ctx.currentClass != originalClass || ctx.localScope != originalMethod {
		t.Fatal("lowering changed lexical class or executable")
	}
	samIdentity205RequireAdapter(t, &ast.File{Decls: declarations}, info.structName)
	return ctx, expression, declarations, info
}

// The allocation, its source class and its native views must share one carrier;
// a native family TypeID or a method-shaped Go interface is insufficient.
func TestCI209AnonymousFunctionalOwnedCarrier(t *testing.T) {
	rows := []struct{ name, family, source, expected string }{
		{"Function explicit", "Function", `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`, "Function<String,String>"},
		{"Function diamond target", "Function", `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<>(){public String apply(String value){return value;}};}}`, "Function<String,String>"},
		{"Function method binder", "Function", `import java.util.function.Function;class Main{static<T>Function<T,T> make(T token){return new Function<T,T>(){public T apply(T value){return token;}};}}`, "Function<T,T>"},
		{"Supplier explicit", "Supplier", `import java.util.function.Supplier;class Main{static Supplier<String> make(){return new Supplier<String>(){public String get(){return "value";}};}}`, "Supplier<String>"},
		{"Supplier method binder", "Supplier", `import java.util.function.Supplier;class Main{static<T>Supplier<T> make(T token){return new Supplier<T>(){public T get(){return token;}};}}`, "Supplier<T>"},
		{"Supplier before initializer", "Supplier", `import java.util.function.Supplier;class Main{static Supplier<String> make(){return new Supplier<String>(){String initialized=((Supplier<String>)(Object)this).get();public String get(){return "ready";}};}}`, "Supplier<String>"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ctx, expression, declarations, info := anonymousFunctional209Lower(t, row.source, row.expected)
			objectInfos, dynamic, views, registration, sourceRegistration, nativeViews := 0, 0, 0, 0, 0, 0
			ownerID := javaClassBinaryName(ctx.currentClass) + "$" + info.structName
			for _, declaration := range declarations {
				if group, ok := declaration.(*ast.GenDecl); ok {
					for _, specification := range group.Specs {
						typ, ok := specification.(*ast.TypeSpec)
						if !ok || typ.Name.Name != info.structName {
							continue
						}
						structure, ok := typ.Type.(*ast.StructType)
						if !ok {
							t.Fatal("actual adapter is not a struct")
						}
						for _, field := range structure.Fields.List {
							pointer, ok := field.Type.(*ast.StarExpr)
							if !ok {
								continue
							}
							selector, ok := pointer.X.(*ast.SelectorExpr)
							if !ok || selector.Sel.Name != "ObjectInfo" {
								continue
							}
							qualifier, ok := selector.X.(*ast.Ident)
							if !ok || qualifier.Name != markStdjavaUsage(ctx) || len(field.Names) != 0 {
								t.Fatal("ObjectInfo must be the actual runtime carrier")
							}
							objectInfos++
						}
					}
				}
				if method, ok := declaration.(*ast.FuncDecl); ok && method.Recv != nil && len(method.Recv.List) == 1 && samIdentity205ReceiverBase(method.Recv.List[0].Type) == info.structName {
					if method.Name.Name == generatedDynamicTypeMethod {
						dynamic++
						if method.Type.TypeParams != nil || method.Type.Params != nil && len(method.Type.Params.List) != 0 || method.Type.Results == nil || len(method.Type.Results.List) != 1 || method.Body == nil || len(method.Body.List) != 1 {
							t.Fatal("dynamic metadata must return its constant source ID without executing a Java body")
						}
						resultType, ok := method.Type.Results.List[0].Type.(*ast.SelectorExpr)
						if !ok || resultType.Sel.Name != "TypeID" {
							t.Fatal("dynamic metadata result must be runtime TypeID")
						}
						resultOwner, ok := resultType.X.(*ast.Ident)
						if !ok || resultOwner.Name != markStdjavaUsage(ctx) {
							t.Fatal("dynamic metadata result owner changed")
						}
						returned, ok := method.Body.List[0].(*ast.ReturnStmt)
						if !ok || len(returned.Results) != 1 {
							t.Fatal("source dynamic ID return missing")
						}
						conversion, ok := returned.Results[0].(*ast.CallExpr)
						if !ok || len(conversion.Args) != 1 || conversion.Ellipsis.IsValid() {
							t.Fatal("source dynamic ID constant conversion missing")
						}
						selector, ok := conversion.Fun.(*ast.SelectorExpr)
						if !ok || selector.Sel.Name != "TypeID" {
							t.Fatal("source dynamic ID must use runtime TypeID")
						}
						qualifier, ok := selector.X.(*ast.Ident)
						if !ok || qualifier.Name != markStdjavaUsage(ctx) {
							t.Fatal("source dynamic ID runtime owner changed")
						}
						literal, ok := conversion.Args[0].(*ast.BasicLit)
						if !ok {
							t.Fatal("source dynamic ID literal missing")
						}
						actual, err := strconv.Unquote(literal.Value)
						if err != nil || actual != ownerID {
							t.Fatalf("anonymous metadata ID %q must belong to %q", actual, ownerID)
						}
					}
					if method.Name.Name == generatedObjectViewMethod {
						views++
						nativeViews += anonymousFunctional209Calls(method, "New"+row.family+"SourceView")
					}
				}
				ast.Inspect(declaration, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					name := anonymousFunctional209Call(call.Fun)
					if name != "RegisterJavaType" && name != "RegisterJavaSourceType" {
						return true
					}
					ids := map[string]int{}
					for _, argument := range call.Args {
						ast.Inspect(argument, func(node ast.Node) bool {
							if literal, ok := node.(*ast.BasicLit); ok {
								if text, err := strconv.Unquote(literal.Value); err == nil {
									ids[text]++
								}
							}
							return true
						})
					}
					if ids[ownerID] == 1 {
						if name == "RegisterJavaSourceType" {
							sourceRegistration++
						} else {
							registration++
							if ids["java.util.function."+row.family] != 1 {
								t.Fatal("source registration lacks its single canonical allocation edge")
							}
						}
					}
					return true
				})
			}
			initialized := anonymousFunctional209Calls(expression, "NewGeneratedObjectInfo")
			if info.fieldInitializerMethodName != "" && initialized == 1 {
				call, ok := expression.(*ast.CallExpr)
				if !ok {
					t.Fatal("source initializers need an owned construction sequence")
				}
				body, ok := call.Fun.(*ast.FuncLit)
				if !ok {
					t.Fatal("owned construction sequence missing")
				}
				identityAt, javaInitializerAt := -1, -1
				for i, statement := range body.Body.List {
					if anonymousFunctional209Calls(statement, "NewGeneratedObjectInfo") == 1 {
						identityAt = i
					}
					if anonymousFunctional209Calls(statement, info.fieldInitializerMethodName)+anonymousFunctional209Calls(statement, info.fieldInitializerMethodName+executionMethodSuffix) == 1 {
						javaInitializerAt = i
					}
				}
				if identityAt < 0 || javaInitializerAt <= identityAt {
					t.Fatal("ObjectInfo must be initialized once before any Java field initializer")
				}
			}
			if objectInfos != 1 || dynamic != 1 || views != 1 || registration != 1 || sourceRegistration != 1 || nativeViews != 1 || initialized != 1 {
				t.Fatalf("anonymous functional allocation needs one owned carrier lifecycle: family=%s struct=%s carrier=%d dynamic=%d view=%d registration=%d source=%d nativeView=%d initialize=%d", row.family, info.structName, objectInfos, dynamic, views, registration, sourceRegistration, nativeViews, initialized)
			}
		})
	}
}

func TestCI209AnonymousFunctionalRawHeaderAuthority(t *testing.T) {
	_, _, _, info := anonymousFunctional209Lower(t, `import java.util.function.Function;class Main{static Object make(){return new Function(){public Object apply(Object value){return value;}};}}`, "Function<String,String>")
	if len(info.scope.ImplementedInterfaces) != 1 || info.scope.ImplementedInterfaces[0] != "Function" {
		t.Fatalf("raw allocation header must not borrow target display: interfaces=%v", info.scope.ImplementedInterfaces)
	}
}

func TestCI209AnonymousFunctionalCanonicalOwnerGuards(t *testing.T) {
	rows := []struct{ name, source string }{
		{"source Function class", `class Function<A,B>{public B apply(A value){return null;}}class Main{static Object make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`},
		{"source Supplier interface", `interface Supplier<T>{T get();}class Main{static Object make(){return new Supplier<String>(){public String get(){return "value";}};}}`},
		{"method Function binder", `class Main{static<Function>Object make(){return new Function(){public Object apply(Object value){return value;}};}}`},
		{"qualified java binder", `class Main{static<java>Object make(){return new java.util.function.Supplier<String>(){public String get(){return "value";}};}}`},
		{"unknown prefix", `class Main{static Object make(){return new foreign.Function<String,String>(){public String apply(String value){return value;}};}}`},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, expression, declarations, _ := anonymousFunctional209Lower(t, row.source, "")
			for _, name := range []string{"NewFunctionSourceView", "NewSupplierSourceView"} {
				for _, declaration := range declarations {
					if anonymousFunctional209Calls(declaration, name) != 0 {
						t.Fatal("unproved source spelling acquired a native functional view")
					}
				}
				if anonymousFunctional209Calls(expression, name) != 0 {
					t.Fatal("allocation expression acquired unproved native view")
				}
			}
		})
	}
}

func TestCI209AnonymousFunctionalProtocolSelectorReservations(t *testing.T) {
	for _, family := range []string{"Function", "Supplier"} {
		t.Run(family, func(t *testing.T) {
			header, implementation := "Function<String,String>", `public String apply(String value){return value;}`
			if family == "Supplier" {
				header, implementation = "Supplier<String>", `public String get(){return "value";}`
			}
			text := "import java.util.function." + family + ";class Main{static " + header + " make(){return new " + header + "(){Object ObjectInfo;public int JavaObjectInfo(){return 3;}public int JavaDynamicTypeID(){return 5;}public int Java2goReferenceDynamicType(){return 7;}public int Java2goReferenceView(int value){return value;}" + implementation + "};}}"
			_, _, _, info := anonymousFunctional209Lower(t, text, header)
			for _, field := range info.scope.Fields {
				if field.OriginalName == "ObjectInfo" && field.Name == "ObjectInfo" {
					t.Fatal("source field must not occupy the anonymous ObjectInfo carrier selector")
				}
			}
			for _, name := range []string{"JavaObjectInfo", "JavaDynamicTypeID", generatedDynamicTypeMethod, generatedObjectViewMethod} {
				methods := info.scope.FindMethod().ByOriginalName(name)
				if len(methods) != 1 || methods[0].Name == name {
					t.Fatalf("anonymous source method family must retain a distinct protocol selector: %s", name)
				}
			}
		})
	}
}

// These parser-level mutation controls exercise the same actual lowering entry
// as the positive carrier cases. They are not additional legal-Java fixtures.
// Each must refuse before constructing from an equal-span foreign allocation,
// mutable registry donation, stale source, or a different registered root.
func TestCI209AnonymousFunctionalAllocationOwnership(t *testing.T) {
	for _, name := range []string{"foreign allocation", "foreign type", "foreign body", "wrong current root", "stale source", "copied cached info", "detached cached owner"} {
		t.Run(name, func(t *testing.T) {
			const source = `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`
			ctx, _, _, info := anonymousFunctional209Lower(t, source, "Function<String,String>")
			allocation := info.scope.Class.DeclarationNode
			typ, body := allocation.ChildByFieldName("type"), objectCreationClassBody(allocation)
			bytes := ctx.currentFile.Source
			foreign := parsing.SourceFile{Name: "Other.java", Source: []byte(source)}
			if err := foreign.ParseAST(); err != nil {
				t.Fatal(err)
			}
			other := findNode(foreign.Ast, "object_creation_expression")
			if other == nil || other.Equal(allocation) || other.StartByte() != allocation.StartByte() || other.EndByte() != allocation.EndByte() {
				t.Fatal("equal-span independent allocation setup failed")
			}
			key, ok := anonymousClassSourceKey(allocation)
			if !ok {
				t.Fatal("actual allocation key missing")
			}
			switch name {
			case "foreign allocation":
				allocation = other
			case "foreign type":
				typ = other.ChildByFieldName("type")
			case "foreign body":
				body = objectCreationClassBody(other)
			case "wrong current root":
				ctx.currentFile = foreign.ParseSymbols()
			case "stale source":
				bytes = append([]byte(nil), bytes...)
				bytes[0] = ' '
			case "copied cached info":
				copy := *info
				ctx.anonymousClasses[key] = &copy
			case "detached cached owner":
				copyInfo, copyOwner, copyClass := *info, *info.scope, *info.scope.Class
				copyOwner.Class = &copyClass
				copyInfo.scope = &copyOwner
				ctx.anonymousClasses[key] = &copyInfo
			}
			before := len(*ctx.hoistedDecls)
			if result := lowerAnonymousClassToStruct(allocation, typ, body, bytes, ctx); result != nil || len(*ctx.hoistedDecls) != before {
				t.Fatalf("anonymous functional ownership must refuse %s before construction or emission", name)
			}
		})
	}
}
