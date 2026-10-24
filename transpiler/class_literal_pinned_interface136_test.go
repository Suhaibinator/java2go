package transpiler

import (
	"encoding/json"
	"go/ast"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestClassLiteralPinnedInterface136NativeDemandIdentity(t *testing.T) {
	for _, row := range []struct {
		name, source string
		array        bool
	}{
		{"wildcard", `import java.util.*; class Main { Object check(){return Comparator.class;} }`, false},
		{"qualified", `class Main { Object check(){return java.util.Comparator.class;} }`, false},
		{"explicit", `import java.util.Comparator; class Main { Object check(){return Comparator.class;} }`, false},
		{"values remain values", `import java.util.*; class Main { Object Comparator; Object java; Object check(){Object Comparator=null; Object java=null; return Comparator.class;} }`, false},
		{"method binder unrelated to prefix", `class Main { <Comparator> Object check(){return java.util.Comparator.class;} }`, false},
		{"array immediate component", `import java.util.*; class Main { Object check(){return Comparator[].class;} }`, true},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := h.Ctx.Clone()
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("check")[0]
			literal := findNode(h.File.Ast, "class_literal")
			typ, ok := classLiteralJavaType(literal, h.File.Source)
			if !ok {
				t.Fatal("missing literal type")
			}
			expression, ok := classLiteralTypeDescriptorExpr(typ, ctx)
			if !ok {
				t.Fatal("literal descriptor missing")
			}
			if row.array {
				call, valid := expression.(*ast.CallExpr)
				if !valid || volatileTDDCallName(call) != "ArrayTypeID" || len(call.Args) != 1 {
					t.Fatalf("array descriptor=%#v", expression)
				}
				expression = call.Args[0]
			}
			call, valid := expression.(*ast.CallExpr)
			if !valid || volatileTDDCallName(call) != "TypeID" || len(call.Args) != 1 {
				t.Fatalf("nominal descriptor=%#v", expression)
			}
			text, valid := call.Args[0].(*ast.BasicLit)
			if !valid {
				t.Fatal("descriptor missing exact Java identity")
			}
			id, err := strconv.Unquote(text.Value)
			if err != nil || id != "java.util.Comparator" {
				t.Fatalf("descriptor=%q/%v, want java.util.Comparator", id, err)
			}
		})
	}
}

func TestClassLiteralPinnedInterface136BinderDiagnostic(t *testing.T) {
	diagnostics.mu.Lock()
	savedStrict := diagnostics.strict
	savedItems := append([]Diagnostic(nil), diagnostics.items...)
	diagnostics.mu.Unlock()
	defer func() {
		diagnostics.mu.Lock()
		diagnostics.strict = savedStrict
		diagnostics.items = savedItems
		diagnostics.mu.Unlock()
	}()
	for _, row := range []struct {
		name, source, first string
		synthetic, fileless bool
	}{
		{"class binder", `class Main<Comparator>{Object check(){return Comparator.class;}}`, "Comparator", false, false},
		{"method binder", `class Main{<Comparator>Object check(){return Comparator.class;}}`, "Comparator", false, false},
		{"type prefix binder", `class Main{<java>Object check(){return java.util.Comparator.class;}}`, "java", false, false},
		{"synthetic binder", `class Main{Object check(){return Comparator.class;}}`, "Comparator", true, false},
		{"fileless actual binder", `class Main{<Comparator>Object check(){return Comparator.class;}}`, "Comparator", false, true},
	} {
		for _, strict := range []bool{false, true} {
			t.Run(row.name+"/strict="+strconv.FormatBool(strict), func(t *testing.T) {
				h := setupParseHelper(t, row.source)
				ctx := h.Ctx.Clone()
				ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("check")[0]
				if row.synthetic {
					ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam(row.first, nil)}
				}
				literal := findNode(h.File.Ast, "class_literal")
				typ, ok := classLiteralJavaType(literal, h.File.Source)
				if !ok {
					t.Fatal("missing literal")
				}
				if row.fileless {
					ctx.currentFile = nil
				}
				resetDiagnostics()
				setStrictMode(strict)
				var descriptor ast.Expr
				var described bool
				var observed any
				func() {
					defer func() { observed = recover() }()
					descriptor, described = classLiteralTypeDescriptorExpr(typ, ctx)
				}()
				want := Diagnostic{Kind: "class literal of type parameter " + row.first, ClassName: ctx.className}
				if strict {
					failure, ok := observed.(strictModeError)
					if !ok || failure.diagnostic != want {
						t.Fatalf("panic=%#v, want strictModeError %#v", observed, want)
					}
				} else if observed != nil || descriptor != nil || described {
					t.Fatalf("permissive binder admitted: descriptor=%#v described=%v panic=%#v", descriptor, described, observed)
				}
				if got := collectedDiagnostics(); !reflect.DeepEqual(got, []Diagnostic{want}) {
					t.Fatalf("diagnostics=%#v, want %#v", got, want)
				}
			})
		}
	}
}

func TestClassLiteralPinnedInterface136FullPrograms(t *testing.T) {
	compiler := &campaignCompilerParentCLI{directory: t.TempDir()}
	data, err := os.ReadFile("testdata/class_literal_pinned_interface136/full-controls.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Name  string            `json:"name"`
		Entry string            `json:"entry"`
		Want  string            `json:"want"`
		Files map[string]string `json:"files"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 9 {
		t.Fatalf("full controls=%d, want all9", len(rows))
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) { runCampaignCompilerStrictProjectOracle(t, row.Files, row.Entry, row.Want, compiler) })
	}
}

func TestClassLiteralPinnedInterface136GoAliasIsNotJavaBinder(t *testing.T) {
	for _, row := range []struct {
		name, source, alias string
		method              bool
	}{
		{"class alias simple", `import java.util.*; class Main<T>{Object check(){return Comparator.class;}}`, "Comparator", false},
		{"class alias prefix", `class Main<T>{Object check(){return java.util.Comparator.class;}}`, "java", false},
		{"method alias simple", `import java.util.*; class Main{<T>Object check(){return Comparator.class;}}`, "Comparator", true},
		{"method alias prefix", `class Main{<T>Object check(){return java.util.Comparator.class;}}`, "java", true},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := h.Ctx.Clone()
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("check")[0]
			parameters := ctx.currentClass.TypeParameters
			if row.method {
				parameters = ctx.localScope.TypeParameters
			}
			if len(parameters) != 1 || parameters[0].Declaration == nil {
				t.Fatal("missing actual T declaration")
			}
			parameters[0].Declaration.GoName = row.alias
			literal := findNode(h.File.Ast, "class_literal")
			typ, ok := classLiteralJavaType(literal, h.File.Source)
			if !ok {
				t.Fatal("missing class literal")
			}
			diagnostics.mu.Lock()
			savedStrict := diagnostics.strict
			savedItems := append([]Diagnostic(nil), diagnostics.items...)
			diagnostics.mu.Unlock()
			defer func() {
				diagnostics.mu.Lock()
				diagnostics.strict = savedStrict
				diagnostics.items = savedItems
				diagnostics.mu.Unlock()
			}()
			resetDiagnostics()
			setStrictMode(true)
			var expression ast.Expr
			var mapped bool
			var observed any
			func() {
				defer func() { observed = recover() }()
				expression, mapped = classLiteralTypeDescriptorExpr(typ, ctx)
			}()
			if observed != nil || !mapped {
				t.Fatalf("Go alias captured Java literal: mapped=%v panic=%#v", mapped, observed)
			}
			call, valid := expression.(*ast.CallExpr)
			if !valid || volatileTDDCallName(call) != "TypeID" || len(call.Args) != 1 {
				t.Fatalf("descriptor=%#v", expression)
			}
			text, valid := call.Args[0].(*ast.BasicLit)
			if !valid {
				t.Fatal("missing nominal ID")
			}
			id, err := strconv.Unquote(text.Value)
			if err != nil || id != "java.util.Comparator" {
				t.Fatalf("ID=%q/%v", id, err)
			}
			if got := collectedDiagnostics(); len(got) != 0 {
				t.Fatalf("unexpected diagnostic=%#v", got)
			}
		})
	}
}
