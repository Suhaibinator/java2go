package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func boundResultBridgeFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/generic_date_bound/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func boundResultBridgeMethod(t *testing.T, owner *symbol.ClassScope, name string) *symbol.Definition {
	t.Helper()
	for _, method := range owner.Methods {
		if method.OriginalName == name {
			return method
		}
	}
	t.Fatal("missing exact source method")
	return nil
}
func boundResultBridgeGoType(t *testing.T, expression ast.Expr) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), expression); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestGenericFamilyBoundResultDispatchABI(t *testing.T) {
	t.Run("getter_family", func(t *testing.T) {
		helper := setupParseHelper(t, boundResultBridgeFixture(t))
		base := helper.File.Symbols.FindClassScope("Base")
		child := helper.File.Symbols.FindClassScope("DateChild")
		method := boundResultBridgeMethod(t, base, "read")
		family, ok := genericFamilyMethodPlan(base, method, helper.Ctx)
		if !ok || family == nil {
			t.Fatal("Date-bound getter lacks a proved erased ancestor result bridge family")
		}
		selected, ok := genericFamilySpecializedMethod(child, boundResultBridgeMethod(t, child, "read"), helper.Ctx)
		if !ok || selected.family == nil || selected.family.owner != base || selected.bridge.method != boundResultBridgeMethod(t, child, "read") {
			t.Fatal("Date-bound getter did not select its actual ancestor bridge")
		}
		if !selected.bridge.result.requiresWidening {
			t.Fatal("narrow Date result did not retain bridge widening")
		}
		ctx := classScopeCtx(base, helper.Ctx)
		typ := javaTypeStringToGoTypeExpr(family.erasedResult, nil, ctx)
		if boundResultBridgeGoType(t, typ) != "any" {
			t.Fatal("parent result descriptor is not the physical Object carrier")
		}
	})
	t.Run("emitted_dispatch", func(t *testing.T) {
		text := renderGoFileFromJava(t, boundResultBridgeFixture(t)+`class BoundReadProbe { static java.util.Date normal(DateChild<java.util.Date> value){return value.read();} static java.util.Date through(Base<java.util.Date> value){return value.read();} }`)
		file, err := parser.ParseFile(token.NewFileSet(), "generated.go", text, 0)
		if err != nil {
			t.Fatal(err)
		}
		results := map[string]string{}
		storedDateValue := false
		constructorForwarded := false
		registrationAsserted := false
		for _, declaration := range file.Decls {
			if group, ok := declaration.(*ast.GenDecl); ok {
				for _, spec := range group.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.Name == "dateChildJava2goErased" {
						structure, ok := typ.Type.(*ast.StructType)
						if ok {
							for _, field := range structure.Fields.List {
								for _, name := range field.Names {
									if name.Name == "stored" && strings.HasSuffix(boundResultBridgeGoType(t, field.Type), ".DateValue") {
										storedDateValue = true
									}
								}
							}
						}
					}
				}
			}
			method, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if method.Recv != nil && len(method.Recv.List) == 1 {
				pointer, ok := method.Recv.List[0].Type.(*ast.StarExpr)
				if ok {
					owner, ok := pointer.X.(*ast.Ident)
					if ok && owner.Name == "dateChildJava2goErased" && method.Type.Results != nil && len(method.Type.Results.List) == 1 {
						results[method.Name.Name] = boundResultBridgeGoType(t, method.Type.Results.List[0].Type)
					}
				}
			}
			if method.Name.Name == "java2goSetBaseSelf" {
				ast.Inspect(method, func(node ast.Node) bool {
					if assertion, ok := node.(*ast.TypeAssertExpr); ok && assertion.Type != nil {
						registrationAsserted = true
					}
					return true
				})
			}
			if method.Name.Name == "newDateChildJava2goWithSelfJava2goExecution" {
				ast.Inspect(method, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok || len(call.Args) != 2 {
						return true
					}
					generic, ok := call.Fun.(*ast.IndexExpr)
					if !ok {
						return true
					}
					name, ok := generic.X.(*ast.Ident)
					if ok && name.Name == "newBaseJava2goWithSelfJava2goExecution" {
						if forwarded, ok := call.Args[1].(*ast.Ident); ok && forwarded.Name == "__java2goMostDerived" {
							constructorForwarded = true
						}
					}
					return true
				})
			}
		}
		if !storedDateValue {
			t.Fatal("Date storage was erased instead of preserving DateValue")
		}
		if !constructorForwarded || !registrationAsserted {
			t.Fatal("constructor dispatch registration was removed or bypassed")
		}
		if results["read"] != "any" || results["readJava2goExecution"] != "any" {
			t.Fatal("Date implementation does not expose the actual parent read dispatch signatures")
		}
		if !strings.HasSuffix(results["readJava2goExactExecution"], ".DateValue") {
			t.Fatal("typed Date getter body did not retain its exact native result")
		}
	})
}

func TestGenericFamilyBoundResultOriginControls(t *testing.T) {
	for _, test := range []struct {
		name, source, actual, expected string
		want                           bool
	}{
		{"canonical_Date_Object", `class Owner{}`, "java.util.Date", "java.lang.Object", true},
		{"reverse_Object_Date", `class Owner{}`, "java.lang.Object", "java.util.Date", false},
		{"foreign_Date", `class Owner{}`, "foreign.Date", "java.lang.Object", false},
		{"source_Date_not_native", `class Date{}class Owner{}`, "Date", "java.util.Date", false},
		{"source_Object_not_native", `class Object{}class Owner{}`, "java.util.Date", "Object", false},
		{"binder_Date_not_native", `class Owner<Date>{}`, "Date", "java.util.Date", false},
		{"binder_Object_not_native", `class Owner<Object>{}`, "java.util.Date", "Object", false},
		{"source_owned_Object_edge", `class Date{}class Owner{}`, "Date", "java.lang.Object", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			if got := overrideBridgeResultCompatible(test.actual, owner, test.expected, owner, helper.Ctx); got != test.want {
				t.Fatalf("declared result relation = %v, want %v", got, test.want)
			}
		})
	}
	t.Run("setter_bridge_preserved", func(t *testing.T) {
		helper := setupParseHelper(t, boundResultBridgeFixture(t))
		base := helper.File.Symbols.FindClassScope("Base")
		child := helper.File.Symbols.FindClassScope("DateChild")
		selected, ok := genericFamilySpecializedMethod(child, boundResultBridgeMethod(t, child, "write"), helper.Ctx)
		if !ok || selected.family == nil || selected.family.owner != base {
			t.Fatal("existing checked setter bridge lost")
		}
	})
}
