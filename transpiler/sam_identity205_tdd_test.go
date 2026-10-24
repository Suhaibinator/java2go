package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func samIdentity205ReceiverBase(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.StarExpr:
		return samIdentity205ReceiverBase(value.X)
	case *ast.IndexExpr:
		return samIdentity205ReceiverBase(value.X)
	case *ast.IndexListExpr:
		return samIdentity205ReceiverBase(value.X)
	case *ast.Ident:
		return value.Name
	}
	return ""
}

func samIdentity205RequireAdapter(t *testing.T, file *ast.File, receiver string) {
	t.Helper()
	declarations := 0
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, specification := range group.Specs {
			typ, ok := specification.(*ast.TypeSpec)
			if !ok || typ.Name.Name != receiver {
				continue
			}
			if _, structure := typ.Type.(*ast.StructType); !structure {
				t.Fatal("actual source SAM adapter is not a struct declaration")
			}
			declarations++
		}
	}
	if declarations != 1 {
		t.Fatalf("actual source SAM adapter declaration count for %s: %d", receiver, declarations)
	}
}

func samIdentity205RequireMetadata(t *testing.T, file *ast.File, method *ast.FuncDecl, ownerID string) {
	t.Helper()
	alias, imports := "", 0
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if path == "github.com/NickyBoy89/java2go/stdjava" {
			imports++
			alias = "stdjava"
			if specification.Name != nil {
				alias = specification.Name.Name
			}
		}
	}
	if imports != 1 || alias == "" || alias == "." || alias == "_" {
		t.Fatal("metadata must use the actual runtime TypeID import")
	}
	typeID := func(expression ast.Expr) bool {
		selector, ok := expression.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "TypeID" {
			return false
		}
		owner, ok := selector.X.(*ast.Ident)
		return ok && owner.Name == alias
	}
	if method.Name.Name != "JavaDynamicTypeID" || method.Type.TypeParams != nil || method.Type.Params != nil && len(method.Type.Params.List) != 0 || method.Type.Results == nil || len(method.Type.Results.List) != 1 || !typeID(method.Type.Results.List[0].Type) {
		t.Fatal("metadata selector/signature must be JavaDynamicTypeID() runtime.TypeID")
	}
	if method.Body == nil || len(method.Body.List) != 1 {
		t.Fatal("metadata must return its declaration ID without executing a Java body")
	}
	returned, ok := method.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		t.Fatal("metadata declaration ID return missing")
	}
	conversion, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || !typeID(conversion.Fun) || len(conversion.Args) != 1 || conversion.Ellipsis.IsValid() {
		t.Fatal("metadata must return a constant runtime TypeID")
	}
	literal, ok := conversion.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		t.Fatal("metadata Java declaration ID literal missing")
	}
	actual, err := strconv.Unquote(literal.Value)
	if err != nil || actual != ownerID {
		t.Fatalf("metadata ID %q must belong to actual Java declaration %q", actual, ownerID)
	}
}

// These actual source declarations have no array/reflection/identity-demand
// trigger. Source homonyms must retain their declaration ID, never a native ID.
func TestCI205SourceSAMAdapterIdentityWithoutDemand(t *testing.T) {
	for _, name := range []string{"Action", "Function", "Supplier"} {
		t.Run(name, func(t *testing.T) {
			source := "package samidentity; interface " + name + "<T>{T apply(T value);}"
			h := setupParseHelper(t, source)
			scope := h.File.Symbols.FindClassScope(name)
			if scope == nil || classNeedsReferenceIdentity(scope, h.Ctx) {
				t.Fatal("control requires the actual source interface without identity demand")
			}
			receiver := symbol.GoIdentifier(scope.Class.Name + "FuncAdapter")
			ownerID := javaClassBinaryName(scope)
			if ownerID != "samidentity."+name {
				t.Fatal("actual Java declaration lost its original binary name")
			}
			generated := renderGoFileFromJava(t, source)
			file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
			if err != nil {
				t.Fatal(err)
			}
			samIdentity205RequireAdapter(t, file, receiver)
			identities := 0
			for _, declaration := range file.Decls {
				method, ok := declaration.(*ast.FuncDecl)
				if !ok || method.Name.Name != "JavaDynamicTypeID" || method.Recv == nil || len(method.Recv.List) != 1 || samIdentity205ReceiverBase(method.Recv.List[0].Type) != receiver {
					continue
				}
				identities++
				samIdentity205RequireMetadata(t, file, method, ownerID)
			}
			if identities != 1 {
				t.Fatalf("source SAM adapter needs one declaration-owned identity without demand; receiver=%s got %d", receiver, identities)
			}
		})
	}
}

func TestCI205SourceSAMAdapterEligibilityControls(t *testing.T) {
	for _, source := range []string{
		"package samidentity; interface Multiple{Object first(Object value);Object second(Object value);}",
		"package samidentity; interface Empty{}",
		"package samidentity; class Shape{Object apply(Object value){return value;}}",
	} {
		generated := renderGoFileFromJava(t, source)
		if strings.Contains(generated, "FuncAdapter") {
			t.Fatal("ineligible source declaration acquired a SAM adapter")
		}
	}
}
