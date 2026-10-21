package transpiler

import (
	"reflect"
	"testing"
)

func TestSQLNativeDeclarationOriginCanonicalInventory(t *testing.T) {
	for _, name := range []string{"Date", "Time", "Timestamp"} {
		for _, row := range []struct{ kind, source string }{
			{"qualified", `class Guard{void value(java.sql.` + name + ` value){}}`},
			{"import", `import java.sql.` + name + `;class Guard{void value(` + name + ` value){}}`},
			{"demand", `import java.sql.*;class Guard{void value(` + name + ` value){}}`},
		} {
			t.Run(name+row.kind, func(t *testing.T) {
				ctx := nativeExternalOrigin3930Owner(t, row.source, "Guard")
				before := ctx.Clone()
				imports := map[string]string{}
				for n, pkg := range ctx.currentFile.Imports {
					imports[n] = pkg
				}
				for repeat := 0; repeat < 3; repeat++ {
					got := nativeExternalOrigin3930Parameter(t, ctx, "value")
					if got.externalName != "java.sql."+name || got.unresolved || got.parameter != nil || got.nominalScope != nil {
						t.Fatalf("declared SQL owner missing: %+v", got)
					}
				}
				if !reflect.DeepEqual(before, ctx) || !reflect.DeepEqual(imports, ctx.currentFile.Imports) {
					t.Fatal("declaration proof mutated caller")
				}
			})
		}
	}
}

func TestSQLNativeDeclarationOriginNamespaceShields(t *testing.T) {
	for _, name := range []string{"Date", "Time", "Timestamp"} {
		t.Run(name, func(t *testing.T) {
			source := nativeExternalOrigin3930Owner(t, `class Guard{static class java{static class sql{static class `+name+`{}}}void value(java.sql.`+name+` value){}}`, "Guard")
			if got := nativeExternalOrigin3930Parameter(t, source, "value"); got.nominalScope == nil || got.externalName != "" {
				t.Fatalf("source qualified TYPE root became native: %+v", got)
			}
			homonym := nativeExternalOrigin3930Owner(t, `class `+name+`{}class Guard{void value(`+name+` value){}}`, "Guard")
			if got := nativeExternalOrigin3930Parameter(t, homonym, "value"); got.nominalScope == nil || got.externalName != "" {
				t.Fatalf("source homonym became native: %+v", got)
			}
			valueRoot := nativeExternalOrigin3930Owner(t, `class Guard{int java;void value(java.sql.`+name+` value){}}`, "Guard")
			if got := nativeExternalOrigin3930Parameter(t, valueRoot, "value"); got.externalName != "java.sql."+name {
				t.Fatalf("VALUE root changed TYPE namespace: %+v", got)
			}
			captured := nativeExternalOrigin3930Owner(t, `class Guard<`+name+`>{void value(`+name+` value){}}`, "Guard")
			if got := nativeExternalOrigin3930Parameter(t, captured, "value"); got.parameter != captured.currentClass.TypeParameters[0].Declaration || got.externalName != "" {
				t.Fatalf("captured class declaration recaptured: %+v", got)
			}
			method := nativeExternalOrigin3930Owner(t, `class Guard{<`+name+`>void value(`+name+` value){}}`, "Guard")
			if got := nativeExternalOrigin3930Parameter(t, method, "value"); got.parameter != method.currentClass.FindMethod().ByOriginalName("value")[0].TypeParameters[0].Declaration || got.externalName != "" {
				t.Fatalf("captured method declaration recaptured: %+v", got)
			}
		})
	}
}

func TestSQLNativeDeclarationOriginUnknownAmbiguityAndFileShields(t *testing.T) {
	for _, source := range []string{
		`class Guard{void value(Timestamp value){}}`,
		`import foreign.Timestamp;class Guard{void value(Timestamp value){}}`,
		`class Guard{void value(foreign.Timestamp value){}}`,
		`import java.sql.*;import java.util.*;class Guard{void value(Date value){}}`,
		`import java.sql.*;import unknown.*;class Guard{void value(Timestamp value){}}`,
		`class Guard<java>{void value(java.sql.Timestamp value){}}`,
	} {
		ctx := nativeExternalOrigin3930Owner(t, source, "Guard")
		if got := nativeExternalOrigin3930Parameter(t, ctx, "value"); got.externalName != "" || !got.unresolved {
			t.Fatalf("unproved declaration became native: %+v", got)
		}
	}
	ctx := nativeExternalOrigin3930Owner(t, `class Guard{}`, "Guard")
	noFile := ctx.Clone()
	noFile.currentFile = nil
	noOwner := ctx.Clone()
	noOwner.currentClass = nil
	stale := ctx.Clone()
	copyFile := *ctx.currentFile
	stale.currentFile = &copyFile
	for _, bad := range []Ctx{noFile, noOwner, stale} {
		if got := declaredExternalNominalOwner("java.sql.Timestamp", nil, bad); got != "" {
			t.Fatalf("native SQL proof lacked owning file: %s", got)
		}
	}
}
