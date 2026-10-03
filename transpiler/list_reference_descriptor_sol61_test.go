package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"testing"
)

func TestListReferenceDescriptorSourceIdentitySol61(t *testing.T) {
	for _, test := range []struct { name, source, written, want, method string }{
		{"wildcard-import", `package p; import java.util.*; class Use {}`, "List<String>", `stdjava.TypeID("java.util.List")`, ""},
		{"single-import", `package p; import java.util.List; class Use {}`, "List<String>", `stdjava.TypeID("java.util.List")`, ""},
		{"qualified", `package p; class Use {}`, "java.util.List<String>", `stdjava.TypeID("java.util.List")`, ""},
		{"nested-array", `package p; import java.util.List; class Use {}`, "List<String>[][]", `stdjava.ArrayTypeID(stdjava.ArrayTypeID(stdjava.TypeID("java.util.List")))`, ""},
		{"same-package-source", `package p; class Use {} class List<T> {}`, "List<String>", `stdjava.TypeID("p.List")`, ""},
		{"nested-source", `package p; import java.util.*; class Use { static class List<T> {} }`, "List<String>", `stdjava.TypeID("p.Use$List")`, ""},
		{"other-qualified-external", `package p; import java.util.List; class Use {}`, "other.List<String>", `stdjava.TypeID("other.List")`, ""},
		{"other-single-import", `package p; import other.List; class Use {}`, "List<String>", `stdjava.TypeID("other.List")`, ""},
		{"class-binder", `package p; import java.util.*; class Use<List> {}`, "List", `stdjava.ObjectTypeID`, ""},
		{"method-binder", `package p; import java.util.*; class Use { <List extends Number> void f() {} }`, "List", `stdjava.NumberTypeID`, "f"},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			ctx := helper.Ctx
			if test.method != "" { ctx.localScope = ctx.currentClass.FindMethodByName(test.method, nil) }
			expr, ok := javaTypeDescriptorExpr(test.written, ctx)
			if !ok { t.Fatal("missing descriptor") }
			var text bytes.Buffer
			if err := printer.Fprint(&text, token.NewFileSet(), expr); err != nil { t.Fatal(err) }
			if text.String() != test.want { t.Fatalf("descriptor(%s) = %s, want %s", test.written, text.String(), test.want) }
		})
	}
}
