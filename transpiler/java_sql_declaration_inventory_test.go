package transpiler

import (
	"reflect"
	"testing"
)

func TestJavaSQLDeclarationInventoryNativeAbsence(t *testing.T) {
	for _, name := range []string{"StringBuilder", "StringBuffer", "Class", "Object", "String", "Integer"} {
		t.Run(name, func(t *testing.T) {
			ctx := nativeExternalOrigin3930Owner(t, `import java.sql.*;class Guard{void value(`+name+` value){}}`, "Guard")
			before := ctx.Clone()
			imports := map[string]string{}
			for key, value := range ctx.currentFile.Imports {
				imports[key] = value
			}
			for range 3 {
				got := nativeExternalOrigin3930Parameter(t, ctx, "value")
				if got.externalName != "java.lang."+name || got.unresolved || got.parameter != nil || got.nominalScope != nil {
					t.Fatalf("known SQL absence lost native declaration: %+v", got)
				}
			}
			if !reflect.DeepEqual(before, ctx) || !reflect.DeepEqual(imports, ctx.currentFile.Imports) {
				t.Fatal("package proof mutated use context")
			}
		})
	}
}

func TestJavaSQLDeclarationInventoryPresenceAndAmbiguity(t *testing.T) {
	for _, name := range []string{"Date", "Time", "Timestamp"} {
		ctx := nativeExternalOrigin3930Owner(t, `import java.sql.*;class Guard{void value(`+name+` value){}}`, "Guard")
		got := nativeExternalOrigin3930Parameter(t, ctx, "value")
		if got.externalName != "java.sql."+name || got.unresolved {
			t.Fatalf("supported SQL presence lost: %+v", got)
		}
	}
	for _, source := range []string{
		`import java.sql.*;import java.util.*;class Guard{void value(Date value){}}`,
		`import java.util.*;import java.sql.*;class Guard{void value(Date value){}}`,
		`import java.sql.*;import unknown.*;class Guard{void value(StringBuilder value){}}`,
		`import unknown.*;import java.sql.*;class Guard{void value(Timestamp value){}}`,
		`import java.sql.*;class Guard{void value(Array value){}}`,
		`import java.sql.*;class Guard{void value(SQLType value){}}`,
		`import java.sql.*;class Guard{void value(DriverInfo value){}}`,
		`import java.sql.*;class Guard{void value(Missing value){}}`,
		`import foreign.StringBuilder;import java.sql.*;class Guard{void value(StringBuilder value){}}`,
	} {
		ctx := nativeExternalOrigin3930Owner(t, source, "Guard")
		got := nativeExternalOrigin3930Parameter(t, ctx, "value")
		if got.externalName != "" || !got.unresolved {
			t.Fatalf("presence/unknown/ambiguity became permissive API admission: %+v", got)
		}
	}
}

func TestJavaSQLDeclarationInventoryNamespaceAndFileShields(t *testing.T) {
	for _, source := range []string{
		`import java.sql.*;class StringBuilder{}class Guard{void value(StringBuilder value){}}`,
		`import java.sql.*;class Timestamp{}class Guard{void value(Timestamp value){}}`,
		`import java.sql.*;class Guard{static class java{static class sql{static class Timestamp{}}}void value(java.sql.Timestamp value){}}`,
	} {
		ctx := nativeExternalOrigin3930Owner(t, source, "Guard")
		got := nativeExternalOrigin3930Parameter(t, ctx, "value")
		if got.nominalScope == nil || got.externalName != "" {
			t.Fatalf("source declaration was replaced by native inventory: %+v", got)
		}
	}
	for _, source := range []string{
		`import java.sql.*;class Guard<StringBuilder>{void value(StringBuilder value){}}`,
		`import java.sql.*;class Guard{<Timestamp>void value(Timestamp value){}}`,
	} {
		ctx := nativeExternalOrigin3930Owner(t, source, "Guard")
		got := nativeExternalOrigin3930Parameter(t, ctx, "value")
		if got.parameter == nil || got.externalName != "" {
			t.Fatalf("binder became inventory member: %+v", got)
		}
	}
	ctx := nativeExternalOrigin3930Owner(t, `import java.sql.*;class Guard{}`, "Guard")
	noFile := ctx.Clone()
	noFile.currentFile = nil
	noOwner := ctx.Clone()
	noOwner.currentClass = nil
	stale := ctx.Clone()
	copyFile := *ctx.currentFile
	stale.currentFile = &copyFile
	for _, bad := range []Ctx{noFile, noOwner, stale} {
		for _, name := range []string{"StringBuilder", "Timestamp"} {
			if got := declaredExternalNominalOwner(name, nil, bad); got != "" {
				t.Fatalf("native proof gained missing/stale owner certificate: %s", got)
			}
		}
	}
}
