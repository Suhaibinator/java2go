package transpiler

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func codecOriginReadContext(t *testing.T, sources map[string]string, expression string) (Ctx, *sitter.Node) {
	t.Helper()
	ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
	ctx.currentClass = ctx.currentFile.FindClassScope("Main")
	if ctx.currentClass == nil {
		t.Fatal("actual Main declaration missing")
	}
	ctx.className = ctx.currentClass.Class.Name
	methods := ctx.currentClass.FindMethod().ByOriginalName("read")
	if len(methods) != 1 {
		t.Fatal("actual read declaration missing")
	}
	ctx.localScope = methods[0]
	var selected *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n.Content(ctx.currentFile.Source) == expression {
			selected = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if selected == nil {
		t.Fatalf("actual expression %q missing", expression)
	}
	return ctx, selected
}

func TestCodecIntrinsicPrimitiveResultOrigins(t *testing.T) {
	for _, row := range []struct {
		method, typ string
		rank        int
	}{
		{"toCharArray", "char[]", 1},
		{"getBytes", "byte[]", 1},
		{"length", "int", 0},
	} {
		t.Run(row.method, func(t *testing.T) {
			expression := "text." + row.method + "()"
			ctx, call := codecOriginReadContext(t, map[string]string{"app/Main.java": "package app; class Main {static " + row.typ + " read(java.lang.String text," + row.typ + " expected){return " + expression + ";}}"}, expression)
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(call, ctx, ctx.currentFile.Source, &actual)
			if !known || typ != row.typ || actual.javaType != row.typ || actual.arrayRank != row.rank || actual.unresolved || actual.externalName != "" || actual.nominalScope != nil || actual.parameter != nil {
				t.Errorf("selected intrinsic lost primitive origin: type=%q known=%t origin=%+v", typ, known, actual)
			}
			actual.javaType = typ // The ordinary overload scorer performs this step.
			formal := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[1]), ctx)
			_, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
			if row.rank != 0 && (!handled || !applies || !exact) {
				t.Errorf("array invocation rejected: actual=%+v formal=%+v exact/applies/handled=%t/%t/%t", actual, formal, exact, applies, handled)
			}
		})
	}
}

func TestCodecCanonicalThrowableConversions(t *testing.T) {
	for _, row := range []struct {
		name, actual, formal, declaration string
		applies                           bool
	}{
		{"CCE_to_Throwable", "java.lang.ClassCastException", "java.lang.Throwable", "", true},
		{"CCE_to_Exception", "java.lang.ClassCastException", "java.lang.Exception", "", true},
		{"CCE_to_RuntimeException", "java.lang.ClassCastException", "java.lang.RuntimeException", "", true},
		{"reverse", "java.lang.Throwable", "java.lang.ClassCastException", "", false},
		{"array", "java.lang.ClassCastException[]", "java.lang.Throwable", "", false},
		{"foreign_actual", "foreign.ClassCastException", "java.lang.Throwable", "", false},
		{"foreign_formal", "java.lang.ClassCastException", "foreign.Throwable", "", false},
		{"caller_binder", "ClassCastException", "java.lang.Throwable", "<ClassCastException>", false},
		// Unresolved names are parser rejection controls, not JVM fixtures.
		{"unknown_actual", "java.lang.UnknownCodecFailure", "java.lang.Throwable", "", false},
		{"unknown_formal", "java.lang.ClassCastException", "java.lang.UnknownCodecFailure", "", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, _ := codecOriginReadContext(t, map[string]string{
				"app/Main.java":                   "package app; class Main" + row.declaration + " {void read(" + row.actual + " actual," + row.formal + " formal){}}",
				"foreign/ClassCastException.java": "package foreign; public class ClassCastException {}",
				"foreign/Throwable.java":          "package foreign; public class Throwable {}",
			}, "read")
			actual := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			formal := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[1]), ctx)
			_, _, applies, _ := declaredOriginConversion(actual, formal, ctx)
			if applies != row.applies {
				t.Errorf("canonical conversion=%t want=%t actual=%+v formal=%+v", applies, row.applies, actual, formal)
			}
			if row.name == "caller_binder" && (actual.parameter == nil || actual.externalName != "") {
				t.Fatal("caller binder lost declaration identity")
			}
		})
	}
}

func TestCodecConstructorCanonicalThrowableSelection(t *testing.T) {
	const expression = "new Wrapper(cause.getMessage(),cause)"
	ctx, creation := codecOriginReadContext(t, map[string]string{
		"app/Main.java": "package app; class Wrapper extends java.lang.Exception {Wrapper(){} Wrapper(java.lang.String text,java.lang.Throwable cause){super(text,cause);}} class Main {static Wrapper read(java.lang.ClassCastException cause){return " + expression + ";}}",
	}, expression)
	target := resolveObjectCreationTargetScope(creation, creation.ChildByFieldName("type"), "Wrapper", ctx.currentFile.Source, ctx)
	selected := findBestConstructor(target, creation.ChildByFieldName("arguments"), ctx, ctx.currentFile.Source)
	if selected == nil || selected.def == nil || len(selected.def.Parameters) != 2 {
		t.Fatalf("real canonical Throwable overload was not selected: %+v", selected)
	}
}

func TestCodecOriginSourceAndBinderShields(t *testing.T) {
	for _, row := range []struct {
		name, parameter string
		binder          bool
	}{
		{"source_homonym", "model.String", false},
		{"captured_binder", "Box<String>", true},
	} {
		t.Run(row.name, func(t *testing.T) {
			expression := "text.toCharArray()"
			main := "package app; class Main {static char[] read(" + row.parameter + " text){return " + expression + ";}}"
			if row.binder {
				expression = "box.field.toCharArray()"
				main = "package app; interface Textish {char[] toCharArray();} class Box<String extends Textish> {String field;} class Main<String extends Textish> {char[] read(Box<String> box){return " + expression + ";}}"
			}
			ctx, call := codecOriginReadContext(t, map[string]string{
				"app/Main.java":     main,
				"model/String.java": "package model; public class String {public char[] toCharArray(){return new char[]{'x'};}}",
			}, expression)
			if result, native := inferIntrinsicMethodResultType(call, ctx, ctx.currentFile.Source); native {
				t.Fatalf("source/binder invocation acquired native signature %q", result)
			}
			var receiver inferredJavaTypeOrigin
			inferExprJavaType(call.ChildByFieldName("object"), ctx, ctx.currentFile.Source, &receiver)
			if receiver.externalName != "" || row.binder && receiver.parameter == nil {
				t.Fatalf("source/binder receiver lost ownership: %+v", receiver)
			}
		})
	}
}
