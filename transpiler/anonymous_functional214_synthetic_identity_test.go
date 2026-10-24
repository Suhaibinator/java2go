package transpiler

import (
	"go/ast"
	"strconv"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

type ci214CanonicalRow struct{ name, family, source, expected string }

func ci214CanonicalRows() []ci214CanonicalRow {
	return []ci214CanonicalRow{
		{"Function_explicit", "Function", `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`, "Function<String,String>"},
		{"Function_diamond_target", "Function", `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<>(){public String apply(String value){return value;}};}}`, "Function<String,String>"},
		{"Function_method_binder", "Function", `import java.util.function.Function;class Main{static<T>Function<T,T> make(T token){return new Function<T,T>(){public T apply(T value){return token;}};}}`, "Function<T,T>"},
		{"Supplier_explicit", "Supplier", `import java.util.function.Supplier;class Main{static Supplier<String> make(){return new Supplier<String>(){public String get(){return "value";}};}}`, "Supplier<String>"},
		{"Supplier_method_binder", "Supplier", `import java.util.function.Supplier;class Main{static<T>Supplier<T> make(T token){return new Supplier<T>(){public T get(){return token;}};}}`, "Supplier<T>"},
		{"Supplier_before_initializer", "Supplier", `import java.util.function.Supplier;class Main{static Supplier<String> make(){return new Supplier<String>(){String initialized=((Supplier<String>)(Object)this).get();public String get(){return "ready";}};}}`, "Supplier<String>"},
	}
}
func ci214Identity(scope *symbol.ClassScope, name string, ctx Ctx) []ast.Decl {
	return syntheticHierarchicalReferenceIdentityDecls(name, "review.identity."+name, scope, nil, nil, scope.GoTypeParameterNames(), ctx)
}
func ci214Factories(declarations []ast.Decl, family string) int {
	return anonymousFunctional209Calls(&ast.File{Name: ast.NewIdent("snapshot"), Decls: declarations}, "New"+family+"SourceView")
}
func ci214Cases(declarations []ast.Decl, family string) int {
	count := 0
	ast.Inspect(&ast.File{Name: ast.NewIdent("snapshot"), Decls: declarations}, func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, value := range clause.List {
			ast.Inspect(value, func(node ast.Node) bool {
				if literal, ok := node.(*ast.BasicLit); ok {
					s, err := strconv.Unquote(literal.Value)
					if err == nil && s == "java.util.function."+family {
						count++
					}
				}
				return true
			})
		}
		return true
	})
	return count
}
func ci214RequireIdentity(t *testing.T, declarations []ast.Decl, family string, wanted int) {
	t.Helper()
	if ci214Factories(declarations, family) != wanted || ci214Cases(declarations, family) != wanted {
		t.Fatalf("identity family composition factories=%d cases=%d want=%d", ci214Factories(declarations, family), ci214Cases(declarations, family), wanted)
	}
}
func TestCI214SyntheticFunctionalIdentityCanonicalComposition(t *testing.T) {
	for _, row := range ci214CanonicalRows() {
		t.Run(row.name, func(t *testing.T) {
			ctx, _, _, info := anonymousFunctional209Lower(t, row.source, row.expected)
			ci214RequireIdentity(t, ci214Identity(info.scope, info.structName, ctx), row.family, 0)
			owned := anonymousFunctionalIdentity(info.functional, ctx)
			if len(owned) != 3 {
				t.Fatal("complete owned anonymous identity missing")
			}
			ci214RequireIdentity(t, owned, row.family, 1)
			if anonymousFunctional209Calls(&ast.File{Name: ast.NewIdent("snapshot"), Decls: owned}, "RegisterJavaType") != 1 {
				t.Fatal("owned registration composition changed")
			}
		})
	}
}
func TestCI214SyntheticFunctionalIdentityCanonicalRefusal(t *testing.T) {
	modes := []string{"Body", "Failed", "copied_info", "copied_scope", "copied_class", "copied_certificate", "removed_certificate", "removed_anonymous_map", "foreign_source_bytes", "foreign_hoist", "foreign_counter", "foreign_physical_ABI"}
	for _, row := range []ci214CanonicalRow{ci214CanonicalRows()[0], ci214CanonicalRows()[3]} {
		for _, mode := range modes {
			t.Run(row.family+"/"+mode, func(t *testing.T) {
				ctx, _, _, info := anonymousFunctional209Lower(t, row.source, row.expected)
				certificate := info.functional
				key, _ := anonymousClassSourceKey(certificate.allocation)
				switch mode {
				case "Body":
					certificate.phase = anonymousFunctionalBody
				case "Failed":
					certificate.phase = anonymousFunctionalFailed
				case "copied_info":
					copy := *info
					ctx.anonymousClasses[key] = &copy
				case "copied_scope":
					copy := *info.scope
					info.scope = &copy
				case "copied_class":
					copy := *info.scope.Class
					info.scope.Class = &copy
				case "copied_certificate":
					copy := *certificate
					info.functional = &copy
				case "removed_certificate":
					info.functional = nil
				case "removed_anonymous_map":
					delete(ctx.anonymousClasses, key)
				case "foreign_source_bytes":
					ctx.currentFile.Source = append([]byte(nil), ctx.currentFile.Source...)
				case "foreign_hoist":
					copy := append([]ast.Decl(nil), (*ctx.hoistedDecls)...)
					ctx.hoistedDecls = &copy
				case "foreign_counter":
					copy := *ctx.anonClassCounter
					ctx.anonClassCounter = &copy
				case "foreign_physical_ABI":
					ctx.rawGenericParameterTypes = map[string]string{"Donated": "Object"}
				}
				ci214RequireIdentity(t, ci214Identity(info.scope, info.structName, ctx), row.family, 0)
				if anonymousFunctionalIdentity(certificate, ctx) != nil {
					t.Fatal("incomplete copied retired or foreign anonymous authority exported")
				}
			})
		}
	}
}
func TestCI214SyntheticFunctionalIdentityNamedLocalRouting(t *testing.T) {
	for _, parent := range []string{"Function", "Supplier", "Runnable"} {
		for _, child := range []string{"Function", "Supplier"} {
			t.Run(parent+"/"+child, func(t *testing.T) {
				ctx, _, local := anonymousFunctional214NestedContext(t, parent, child)
				query := classScopeCtx(local.scope, ctx)
				contracts := sourceNativeFunctionalContracts(local.scope, query)
				wanted := 0
				if child == "Function" {
					if len(contracts) != 1 || !anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
						t.Fatal("actual named local Function eligibility missing")
					}
					wanted = 1
				} else {
					if !anonymousFunctionalActualNamed(local.scope, query) || len(contracts) != 0 {
						t.Fatal("actual named local Supplier role or contract refusal changed")
					}
					if _, owned := supplierSourceDeclarationCtx(local.scope, query); owned {
						t.Fatal("local Supplier gained registered-file authority")
					}
				}
				ci214RequireIdentity(t, ci214Identity(local.scope, local.structName, query), child, wanted)
			})
		}
	}
}
func ci214MutateParent(t *testing.T, mode, source string, ctx *Ctx, owner *anonymousClassInfo) {
	t.Helper()
	role := owner.lexicalOwner
	key, _ := anonymousClassSourceKey(role.allocation)
	foreign := parsing.SourceFile{Name: "Other.java", Source: []byte(source)}
	if err := foreign.ParseAST(); err != nil {
		t.Fatal(err)
	}
	allocation := findNode(foreign.Ast, "object_creation_expression")
	if allocation == nil || allocation.Equal(role.allocation) {
		t.Fatal("independent parent AST missing")
	}
	switch mode {
	case "missing_owner_info":
		delete(ctx.anonymousClasses, key)
	case "copied_owner_info":
		copy := *owner
		ctx.anonymousClasses[key] = &copy
	case "copied_owner_role":
		copy := *role
		owner.lexicalOwner = &copy
	case "named_owner_node":
		owner.scope.Class.DeclarationNode = ctx.currentClass.Class.DeclarationNode
	case "foreign_owner_body":
		role.body = objectCreationClassBody(allocation)
	case "foreign_owner_method":
		owner.scope.Methods[0].DeclarationNode = findNode(allocation, "method_declaration")
	case "owner_binder_donation":
		owner.scope.TypeParameters = append(owner.scope.TypeParameters, symbol.NewTypeParam("Donated", nil))
	case "owner_failed_phase":
		role.phase = anonymousFunctionalFailed
	case "owner_local_donation":
		delete(ctx.anonymousClasses, key)
		owner.scope.Class.DeclarationNode = ctx.currentClass.Class.DeclarationNode
		ctx.localClasses["donated"] = &localClassInfo{scope: owner.scope}
	case "owner_graph_donation":
		delete(ctx.anonymousClasses, key)
		owner.scope.Class.DeclarationNode = ctx.currentClass.Class.DeclarationNode
		ctx.currentFile.TopLevelClasses = append(ctx.currentFile.TopLevelClasses, owner.scope)
	case "foreign_hoist":
		copy := append([]ast.Decl(nil), (*ctx.hoistedDecls)...)
		ctx.hoistedDecls = &copy
	case "foreign_counter":
		copy := *ctx.anonClassCounter
		ctx.anonClassCounter = &copy
	case "stale_input":
		ctx.currentFile.SourceFilename = "Other.java"
	default:
		t.Fatal("unlisted parent mutant")
	}
}
func TestCI214SyntheticFunctionalIdentityParentRefusal(t *testing.T) {
	modes := []string{"missing_owner_info", "copied_owner_info", "copied_owner_role", "named_owner_node", "foreign_owner_body", "foreign_owner_method", "owner_binder_donation", "owner_failed_phase", "owner_local_donation", "owner_graph_donation", "foreign_hoist", "foreign_counter", "stale_input"}
	for _, parent := range []string{"Function", "Supplier", "Runnable"} {
		for _, child := range []string{"Function", "Supplier"} {
			for _, mode := range modes {
				t.Run(parent+"/"+child+"/"+mode, func(t *testing.T) {
					ctx, owner, local := anonymousFunctional214NestedContext(t, parent, child)
					ci214MutateParent(t, mode, string(ctx.currentFile.Source), &ctx, owner)
					query := classScopeCtx(local.scope, ctx)
					ci214RequireIdentity(t, ci214Identity(local.scope, local.structName, query), child, 0)
				})
			}
		}
	}
}

// The retention allocation is lowered in the normal source pipeline exactly
// once. Its live lexical record retains the actual owning Local block map.
func ci214NamedFixture(t *testing.T, source, expected string) (Ctx, *anonymousClassInfo, *localClassInfo) {
	t.Helper()
	ctx, _, _, owner := anonymousFunctional209Lower(t, source, expected)
	localNode := findNode(owner.scope.Class.DeclarationNode, "class_declaration")
	if localNode == nil {
		t.Fatal("actual Local node missing")
	}
	witness := findNode(localNode, "object_creation_expression")
	key, ok := anonymousClassSourceKey(witness)
	info := ctx.anonymousClasses[key]
	if !ok || info == nil || info.lexicalOwner == nil || info.lexicalOwner.phase != anonymousFunctionalComplete {
		t.Fatal("actual retention role missing")
	}
	role := info.lexicalOwner
	local := role.lexical.localClasses["Local"]
	if local == nil || local.scope == nil || role.lexical.currentClass != local.scope || !anonymousFunctionalEqual(local.scope.Class.DeclarationNode, localNode) || local.scope.Enclosing != owner.scope || !anonymousFunctionalEqual(role.allocation, witness) {
		t.Fatal("actual owning named block missing")
	}
	ctx.localClasses = role.lexical.localClasses
	if !anonymousLexicalOwnerValid(role, ctx, map[*symbol.ClassScope]bool{}) {
		t.Fatal("actual retention role/stamp invalid")
	}
	return ctx, owner, local
}

const ci214GenericLocalSource = `class Main{static<A,B>java.util.function.Function<A,B> make(B token){return new java.util.function.Function<A,B>(){public B apply(A value){class Local<X,Y> implements java.util.function.Function<A,B>{public B apply(A argument){Object ownershipWitness=new Object(){};return token;}}return new Local<String,String>().apply(value);}};}}`

func TestCI214SyntheticFunctionalIdentityMethodRefusal(t *testing.T) {
	modes := []string{"foreign_method_AST", "selected_constructor", "selected_helper_flag", "selected_helper_name", "selected_platform_variadic", "selected_runtime_default", "selected_static", "selected_private", "selected_hasbody", "selected_source_name", "selected_result_type", "missing_metadata_formal", "extra_metadata_formal", "selected_formal_type", "selected_formal_origin", "selected_formal_direct_binder", "selected_result_origin", "selected_result_direct_binder", "selected_method_binder", "duplicate_selected_table_identity", "own_declared_binder", "own_carried_binder_identity", "own_carried_order", "empty_retained_selector", "ambiguous_retained_selector"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			ctx, _, local := ci214NamedFixture(t, ci214GenericLocalSource, "java.util.function.Function<A,B>")
			query := classScopeCtx(local.scope, ctx)
			contracts := sourceNativeFunctionalContracts(local.scope, query)
			if len(contracts) != 1 || !anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
				t.Fatal("actual generic named baseline missing")
			}
			selected, owner, _ := resolveSourceNativeSAM(local.scope, contracts[0], query)
			if selected == nil || owner != local.scope || len(selected.Parameters) != 1 || len(local.scope.DeclaredTypeParameters) != 2 || len(local.scope.TypeParameters) < 4 {
				t.Fatal("actual method/binder mutant premise missing")
			}
			donor := symbol.NewTypeParam("A", nil).Declaration
			switch mode {
			case "foreign_method_AST":
				foreign := parsing.SourceFile{Name: "Other.java", Source: append([]byte(nil), ctx.currentFile.Source...)}
				if err := foreign.ParseAST(); err != nil {
					t.Fatal(err)
				}
				selected.DeclarationNode = findNode(findNode(findNode(foreign.Ast, "object_creation_expression"), "class_declaration"), "method_declaration")
			case "selected_constructor":
				selected.Constructor = true
			case "selected_helper_flag":
				selected.RequiresHelper = true
			case "selected_helper_name":
				selected.HelperName = "donated"
			case "selected_platform_variadic":
				selected.IsVariadic = true
			case "selected_runtime_default":
				selected.RuntimeDefault = true
			case "selected_static":
				selected.IsStatic = true
			case "selected_private":
				selected.IsPrivate = true
			case "selected_hasbody":
				selected.HasBody = false
			case "selected_source_name":
				selected.OriginalName = "donated"
			case "selected_result_type":
				selected.OriginalType = "A"
			case "missing_metadata_formal":
				selected.Parameters = nil
			case "extra_metadata_formal":
				selected.Parameters = append(selected.Parameters, selected.Parameters[0])
			case "selected_formal_type":
				selected.Parameters[0].OriginalType = "B"
			case "selected_formal_origin":
				selected.Parameters[0].TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"A": donor}
			case "selected_formal_direct_binder":
				selected.Parameters[0].DirectTypeParameter = donor
			case "selected_result_origin":
				selected.TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"B": donor}
			case "selected_result_direct_binder":
				selected.DirectTypeParameter = donor
			case "selected_method_binder":
				selected.TypeParameters = []symbol.TypeParam{symbol.NewTypeParam("Donated", nil)}
			case "duplicate_selected_table_identity":
				local.scope.Methods = append(local.scope.Methods, selected)
			case "own_declared_binder":
				local.scope.DeclaredTypeParameters[0] = symbol.NewTypeParam("Donated", nil)
			case "own_carried_binder_identity":
				local.scope.TypeParameters[len(local.scope.TypeParameters)-1] = symbol.NewTypeParam("Y", nil)
			case "own_carried_order":
				local.scope.TypeParameters[0], local.scope.TypeParameters[1] = local.scope.TypeParameters[1], local.scope.TypeParameters[0]
			case "empty_retained_selector":
				selected.Name = ""
			case "ambiguous_retained_selector":
				copy := *selected
				copy.OriginalName = "other"
				local.scope.Methods = append(local.scope.Methods, &copy)
			}
			if anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
				t.Fatal("donated source method/binder/ABI accepted by new route")
			}
			ci214RequireIdentity(t, ci214Identity(local.scope, local.structName, query), "Function", 0)
		})
	}
}

func ci214InheritedSource(base string) string {
	return base + `class Main{static java.util.function.Function<String,String> make(){return new java.util.function.Function<String,String>(){public String apply(String value){class Local extends Base<String>{public void retain(){Object ownershipWitness=new Object(){};}}new Local().retain();return new Local().apply(value);}};}}`
}

const ci214GenericBase = `class Base<T> implements java.util.function.Function<T,T>{public T apply(T value){return value;}}`

func TestCI214SyntheticFunctionalIdentitySelectorSources(t *testing.T) {
	rows := []struct{ name, source, expected string }{
		{"overload", `class Main{static java.util.function.Function<String,String> make(){return new java.util.function.Function<String,String>(){public String apply(String value){class Local implements java.util.function.Function<String,String>{public int apply(int argument){return argument;}public String apply(String argument){Object ownershipWitness=new Object(){};return argument;}}return new Local().apply(value);}};}}`, "java.util.function.Function<String,String>"},
		{"inherited", ci214InheritedSource(`class Base<T> implements java.util.function.Function<String,String>{public String apply(String value){return value;}}`), "java.util.function.Function<String,String>"},
		{"specialized", ci214InheritedSource(ci214GenericBase), "java.util.function.Function<String,String>"},
		{"own_generic", `class Main{static java.util.function.Function<String,String> make(){return new java.util.function.Function<String,String>(){public String apply(String value){class Local<X> implements java.util.function.Function<X,X>{public X apply(X argument){Object ownershipWitness=new Object(){};return argument;}}return new Local<String>().apply(value);}};}}`, "java.util.function.Function<String,String>"},
		{"carried_generic", ci214GenericLocalSource, "java.util.function.Function<A,B>"},
		{"source_spread", `class Main{static java.util.function.Function<String[],String> make(){return new java.util.function.Function<String[],String>(){public String apply(String[] value){class Local implements java.util.function.Function<String[],String>{public String apply(String... argument){Object ownershipWitness=new Object(){};return "spread";}}return new Local().apply(value);}};}}`, "java.util.function.Function<String[],String>"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ctx, _, local := ci214NamedFixture(t, row.source, row.expected)
			query := classScopeCtx(local.scope, ctx)
			contracts := sourceNativeFunctionalContracts(local.scope, query)
			if len(contracts) != 1 || !anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
				t.Fatal("actual allocated named selector/effective signature rejected")
			}
			selected, owner, _ := resolveSourceNativeSAM(local.scope, contracts[0], query)
			if selected == nil || selected.Name == "" || owner == nil {
				t.Fatal("actual selected method missing")
			}
			if (row.name == "inherited" || row.name == "specialized") && owner == local.scope {
				t.Fatal("inherited selected owner not exercised")
			}
			if row.name == "source_spread" && (!executionParameterIsVariadic(selected, 0) || selected.IsVariadic) {
				t.Fatal("actual source spread/platform ABI distinction missing")
			}
			if row.name == "own_generic" && len(local.scope.DeclaredTypeParameters) != 1 {
				t.Fatal("actual own binder missing")
			}
			if row.name == "carried_generic" && len(local.scope.TypeParameters)-len(local.scope.DeclaredTypeParameters) != 2 {
				t.Fatal("actual carried binder order missing")
			}
			ci214RequireIdentity(t, ci214Identity(local.scope, local.structName, query), "Function", 1)
		})
	}
}
func TestCI214SyntheticFunctionalIdentityInheritedRefusal(t *testing.T) {
	for _, mode := range []string{"foreign_selected_owner_AST", "foreign_selected_method_AST", "selectedowner_sourceinput_invalid", "selectedowner_header_invalid", "donated_selected_binding_origin"} {
		t.Run(mode, func(t *testing.T) {
			source := ci214InheritedSource(ci214GenericBase)
			ctx, _, local := ci214NamedFixture(t, source, "java.util.function.Function<String,String>")
			query := classScopeCtx(local.scope, ctx)
			contracts := sourceNativeFunctionalContracts(local.scope, query)
			if len(contracts) != 1 || !anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
				t.Fatal("actual inherited baseline missing")
			}
			selected, owner, _ := resolveSourceNativeSAM(local.scope, contracts[0], query)
			if selected == nil || owner == local.scope {
				t.Fatal("actual inherited owner missing")
			}
			foreign := parsing.SourceFile{Name: "Other.java", Source: []byte(source)}
			if err := foreign.ParseAST(); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "foreign_selected_owner_AST":
				owner.Class.DeclarationNode = findNode(foreign.Ast, "class_declaration")
			case "foreign_selected_method_AST":
				selected.DeclarationNode = findNode(foreign.Ast, "method_declaration")
			case "selectedowner_sourceinput_invalid":
				ctx.currentFile.SourceFilename = "Other.java"
			case "selectedowner_header_invalid":
				owner.ImplementedInterfaces = []string{"java.lang.Runnable"}
			case "donated_selected_binding_origin":
				selected.Parameters[0].TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"T": symbol.NewTypeParam("T", nil).Declaration}
			}
			if anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
				t.Fatal("foreign inherited owner or signature accepted")
			}
			ci214RequireIdentity(t, ci214Identity(local.scope, local.structName, query), "Function", 0)
		})
	}
}
func TestCI214SyntheticFunctionalIdentityKindVisibility(t *testing.T) {
	rows := []string{"class_package_private", "class_as_interface", "public_class", "implicit_public_interface", "interface_as_class", "inherited_class_as_interface", "class_as_enum", "enum_as_class"}
	for _, mode := range rows {
		t.Run(mode, func(t *testing.T) {
			if mode == "inherited_class_as_interface" {
				ctx, _, local := ci214NamedFixture(t, ci214InheritedSource(`class Base<T> implements java.util.function.Function<T,T>{T apply(T value){return value;}}`), "java.util.function.Function<String,String>")
				query := classScopeCtx(local.scope, ctx)
				contracts := sourceNativeFunctionalContracts(local.scope, query)
				if len(contracts) != 1 {
					t.Fatal("actual inherited visibility contract missing")
				}
				selected, owner, _ := resolveSourceNativeSAM(local.scope, contracts[0], query)
				if selected == nil || owner == local.scope || owner.IsInterface || declarationHasModifier(selected.DeclarationNode, "public") {
					t.Fatal("actual inherited class visibility premise missing")
				}
				if anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
					t.Fatal("package-private inherited source method accepted")
				}
				owner.IsInterface = true
				if anonymousFunctionalNamedViewEligible(local.scope, contracts[0], query) {
					t.Fatal("class-kind donation waived inherited source public")
				}
				ci214RequireIdentity(t, ci214Identity(local.scope, local.structName, query), "Function", 0)
				return
			}
			source := `class Owned implements java.util.function.Function<String,String>{public String apply(String value){return value;}}`
			if mode == "class_package_private" || mode == "class_as_interface" {
				source = `class Owned implements java.util.function.Function<String,String>{String apply(String value){return value;}}`
			}
			if mode == "implicit_public_interface" || mode == "interface_as_class" {
				source = `interface Owned extends java.util.function.Function<String,String>{default String apply(String value){return value;}}`
			}
			if mode == "enum_as_class" {
				source = `enum Owned implements java.util.function.Function<String,String>{ONLY;public String apply(String value){return value;}}`
			}
			helper := setupParseHelper(t, source)
			scope := helper.Ctx.currentFile.FindClassScope("Owned")
			if scope == nil {
				t.Fatal("actual visibility source missing")
			}
			query := classScopeCtx(scope, helper.Ctx)
			contract, _ := nativeFunctionalContract("Function", []string{"java.lang.String", "java.lang.String"})
			wanted := mode == "public_class" || mode == "implicit_public_interface"
			switch mode {
			case "class_as_interface":
				if scope.IsInterface {
					t.Fatal("actual class premise missing")
				}
				scope.IsInterface = true
			case "interface_as_class":
				if !scope.IsInterface {
					t.Fatal("actual interface premise missing")
				}
				scope.IsInterface = false
			case "class_as_enum":
				scope.IsEnum = true
			case "enum_as_class":
				if !scope.IsEnum {
					t.Fatal("actual enum premise missing")
				}
				scope.IsEnum = false
			}
			if anonymousFunctionalNamedViewEligible(scope, contract, query) != wanted {
				t.Fatal("actual source kind/public visibility eligibility changed")
			}
			count := 0
			if wanted {
				count = 1
			}
			ci214RequireIdentity(t, ci214Identity(scope, scope.Class.Name, query), "Function", count)
		})
	}
}
func TestCI214SyntheticFunctionalIdentityOrdinaryBinderRefusal(t *testing.T) {
	for _, row := range []struct{ name, source string }{
		{"Function_binder", `class Main{static<Function>Object make(){return new Function(){public Object apply(Object value){return value;}};}}`},
		{"java_binder", `class Main{static<java>Object make(){return new java.util.function.Supplier<String>(){public String get(){return "value";}};}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, _, _, info := anonymousFunctional209Lower(t, row.source, "")
			declarations := ci214Identity(info.scope, info.structName, ctx)
			ci214RequireIdentity(t, declarations, "Function", 0)
			ci214RequireIdentity(t, declarations, "Supplier", 0)
		})
	}
}
