package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestRecordComponentSourceOriginsAndCopies(t *testing.T) {
	for _, mutation := range []string{"scope-copy", "field-copy", "field-donation", "accessor-copy", "accessor-donation", "header-donation", "field-type", "accessor-type", "accessor-parameter", "static-role", "source"} {
		t.Run(mutation, func(t *testing.T) {
			h := setupParseHelper(t, `class A {record R(int count,String label){public String label(){return label;}}} class B {record R(int count,String label){}}`)
			owner := h.File.Symbols.FindClassScope("A").FindClassScope("R")
			other := h.File.Symbols.FindClassScope("B").FindClassScope("R")
			ctx := classScopeCtx(owner, h.Ctx)
			plans, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if !known || len(plans) != 2 || plans[0].origin.Accessor == nil || plans[1].origin.Accessor != nil {
				t.Fatal("owned implicit/explicit profile rejected")
			}
			origins, ok := other.RecordComponentOrigins()
			if !ok {
				t.Fatal("other record origin missing")
			}
			field, accessor := plans[0].origin.Field, plans[0].origin.Accessor
			switch mutation {
			case "scope-copy":
				copy := *owner
				parent := owner.Enclosing
				for i, c := range parent.Subclasses {
					if c == owner {
						parent.Subclasses[i] = &copy
						defer func() { parent.Subclasses[i] = owner }()
						break
					}
				}
				owner = &copy
			case "field-copy", "field-donation":
				copied := *field
				replacement := &copied
				if mutation == "field-donation" {
					replacement = origins[0].Field
				}
				for i, f := range owner.Fields {
					if f == field {
						owner.Fields[i] = replacement
						break
					}
				}
			case "accessor-copy", "accessor-donation":
				copied := *accessor
				replacement := &copied
				if mutation == "accessor-donation" {
					replacement = origins[0].Accessor
				}
				for i, m := range owner.Methods {
					if m == accessor {
						owner.Methods[i] = replacement
						break
					}
				}
			case "header-donation":
				owner.Class.DeclarationNode = other.Class.DeclarationNode
			case "field-type":
				field.OriginalType = "long"
			case "accessor-type":
				accessor.OriginalType = "long"
			case "accessor-parameter":
				accessor.Parameters = []*symbol.Definition{{OriginalName: "extra", OriginalType: "int"}}
			case "static-role":
				accessor.IsStatic = true
			case "source":
				h.File.Symbols.Source[0] ^= 1
			}
			if _, known := sourceReflectionRecordComponentPlan(owner, ctx); known {
				t.Fatal("borrowed/copy/stale record capability accepted")
			}
		})
	}
}
func TestRecordComponentOriginCopyAndGoRename(t *testing.T) {
	h := setupParseHelper(t, `class Host {record R(int count){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	origins, known := owner.RecordComponentOrigins()
	if !known {
		t.Fatal("origin missing")
	}
	originals := origins[0]
	origins[0].Field = nil
	origins[0].Accessor = nil
	origins[0].Node = nil
	again, known := owner.RecordComponentOrigins()
	if !known || again[0].Field != originals.Field || again[0].Accessor != originals.Accessor {
		t.Fatal("returned copy mutated retained certificate")
	}
	originals.Field.Name = "renamedCountStorage"
	originals.Accessor.Name = "RenamedCountGetter"
	plans, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known || plans[0].origin.Accessor.Name != "RenamedCountGetter" {
		t.Fatal("legitimate Go rename invalidated Java identity")
	}
	declarations := sourceReflectionRecordAccessorBridges(owner, ctx)
	if len(declarations) != 1 {
		t.Fatal("bridge missing")
	}
	var output bytes.Buffer
	if err := printer.Fprint(&output, token.NewFileSet(), declarations[0]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), ".RenamedCountGetter()") {
		t.Fatalf("bridge did not invoke actual resolved getter once: %s", output.String())
	}
	count := 0
	ast.Inspect(declarations[0], func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "RenamedCountGetter" {
				count++
			}
		}
		return true
	})
	if count != 1 {
		t.Fatalf("getter calls=%d", count)
	}
}
func TestRecordComponentSourceHomonymAndBinder(t *testing.T) {
	h := setupParseHelper(t, `class Host {class RecordComponent{} record R(RecordComponent component){} record Generic<RecordComponent>(RecordComponent component){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	plans, known := sourceReflectionRecordComponentPlan(owner, classScopeCtx(owner, h.Ctx))
	if !known {
		t.Fatal("source class component rejected")
	}
	var output bytes.Buffer
	if err := printer.Fprint(&output, token.NewFileSet(), plans[0].typ); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Host$RecordComponent") || strings.Contains(output.String(), "java.lang.reflect.RecordComponent") {
		t.Fatalf("source homonym borrowed platform identity: %s", output.String())
	}
	generic := h.File.Symbols.FindClassScope("Generic")
	plans, known = sourceReflectionRecordComponentPlan(generic, classScopeCtx(generic, h.Ctx))
	if !known {
		t.Fatal("owned binder erasure rejected")
	}
	output.Reset()
	if err := printer.Fprint(&output, token.NewFileSet(), plans[0].typ); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "ObjectTypeID") || strings.Contains(output.String(), "java.lang.reflect.RecordComponent") {
		t.Fatalf("binder borrowed platform identity: %s", output.String())
	}
	nonrecord := h.File.Symbols.FindClassScope("RecordComponent")
	if _, known := sourceReflectionRecordComponentPlan(nonrecord, classScopeCtx(nonrecord, h.Ctx)); known {
		t.Fatal("ordinary source homonym became record")
	}
}
func TestRecordComponentEmptyRequiresSourceAndOrigin(t *testing.T) {
	h := setupParseHelper(t, `class Host {record Empty(){}}`)
	owner := h.File.Symbols.FindClassScope("Empty")
	ctx := classScopeCtx(owner, h.Ctx)
	if plans, known := sourceReflectionRecordComponentPlan(owner, ctx); !known || len(plans) != 0 {
		t.Fatal("genuine empty record rejected")
	}
	cloned := *owner
	if _, known := cloned.RecordComponentOrigins(); known {
		t.Fatal("empty origin copied into a different owner")
	}
	before := append([]byte(nil), h.File.Symbols.Source...)
	h.File.Symbols.Source[0] ^= 1
	defer func() { copy(h.File.Symbols.Source, before) }()
	if _, known := sourceReflectionRecordComponentPlan(owner, ctx); known {
		t.Fatal("empty record borrowed stale source")
	}
}

func TestRecordComponentUnprovedOriginCannotEmitSuccess(t *testing.T) {
	h := setupParseHelper(t, `class Host {record Empty(){}}`)
	owner := h.File.Symbols.FindClassScope("Empty")
	cloned := *owner
	descriptor := &ast.CompositeLit{}
	var failure any
	func() {
		defer func() { failure = recover() }()
		appendSourceReflectionRecordMetadata(descriptor, &cloned, h.Ctx)
	}()
	refused, known := failure.(strictModeError)
	if !known || refused.diagnostic.Kind != "record reflection member origin" || len(descriptor.Elts) != 0 {
		t.Fatalf("unproved origin became successful metadata: %T %v %+v", failure, failure, descriptor.Elts)
	}
}

func TestRecordComponentJavaTypeParameterOriginAndGoNames(t *testing.T) {
	for _, mutation := range []string{"java-name", "bound", "field-binding", "method-binding"} {
		t.Run(mutation, func(t *testing.T) {
			h := setupParseHelper(t, `class Host {record R<T extends Number>(T count){} record S<T extends Number>(T count){}}`)
			owner, other := h.File.Symbols.FindClassScope("R"), h.File.Symbols.FindClassScope("S")
			origins, known := owner.RecordComponentOrigins()
			if !known {
				t.Fatal("parameter origin missing")
			}
			// A Go-only alpha rename preserves the Java parameter declaration identity.
			owner.DeclaredTypeParameters[0].Declaration.GoName = "RenamedGoT"
			if _, known := owner.RecordComponentOrigins(); !known {
				t.Fatal("Go binder rename invalidated Java origin")
			}
			switch mutation {
			case "java-name":
				owner.DeclaredTypeParameters[0].Declaration.SourceName = "OtherT"
			case "bound":
				owner.DeclaredTypeParameters[0].Bounds[0].Original = "Object"
			case "field-binding":
				origins[0].Field.TypeParameterBindings["T"] = other.DeclaredTypeParameters[0].Declaration
			case "method-binding":
				origins[0].Accessor.TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"T": other.DeclaredTypeParameters[0].Declaration}
			}
			if _, known := owner.RecordComponentOrigins(); known {
				t.Fatal("changed Java parameter/bound origin accepted")
			}
		})
	}
}
