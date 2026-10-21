package transpiler

import (
	"bytes"
	"go/format"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func fileChannelCatalogRendered(t *testing.T, name string, ctx Ctx) (string, bool) {
	t.Helper()
	expression, mapped := digestIORuntimeTypeExpr(name, ctx)
	if !mapped {
		return "", false
	}
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), expression); err != nil {
		t.Fatal(err)
	}
	return out.String(), true
}

func TestFileChannelDeclarationCatalogCanonicalSignatures(t *testing.T) {
	for _, row := range []struct{ java, want string }{
		{"java.nio.channels.FileChannel", "*stdjava.FileChannel"},
		{"java.nio.channels.ReadableByteChannel", "stdjava.JavaReadableByteChannel"},
		{"java.nio.channels.WritableByteChannel", "stdjava.JavaWritableByteChannel"},
		{"java.nio.channels.FileChannel.MapMode", "*stdjava.JavaFileChannelMapMode"},
		{"java.nio.MappedByteBuffer", "*stdjava.JavaMappedByteBuffer"},
		{"java.nio.channels.FileLock", "*stdjava.JavaFileLock"},
	} {
		t.Run(row.java, func(t *testing.T) {
			got, ok := fileChannelCatalogRendered(t, row.java, Ctx{})
			if !ok || got != row.want {
				t.Fatalf("canonical ABI=%q,%v want=%q", got, ok, row.want)
			}
		})
	}
}

func TestFileChannelDeclarationCatalogImportsAndShadows(t *testing.T) {
	for _, row := range []struct {
		name, source, query, want string
		extra                     map[string]string
	}{
		{name: "single import", source: "import java.nio.channels.ReadableByteChannel; class Main {}", query: "ReadableByteChannel", want: "stdjava.JavaReadableByteChannel"},
		{name: "demand", source: "import java.nio.channels.*; class Main {}", query: "WritableByteChannel", want: "stdjava.JavaWritableByteChannel"},
		{name: "nested single import", source: "import java.nio.channels.FileChannel.MapMode; class Main {}", query: "MapMode", want: "*stdjava.JavaFileChannelMapMode"},
		{name: "nested demand", source: "import java.nio.channels.FileChannel.*; class Main {}", query: "MapMode", want: "*stdjava.JavaFileChannelMapMode"},
		{name: "canonical outer", source: "import java.nio.channels.FileChannel; class Main {}", query: "FileChannel.MapMode", want: "*stdjava.JavaFileChannelMapMode"},
		{name: "unimported", source: "class Main {}", query: "FileChannel"},
		{name: "foreign exact", source: "class Main {}", query: "other.FileChannel"},
		{name: "foreign import", source: "import other.FileChannel; import java.nio.channels.*; class Main {}", query: "FileChannel"},
		{name: "source local", source: "class Main {static class FileChannel{}}", query: "FileChannel"},
		{name: "source nested", source: "class Main {static class FileChannel{static class MapMode{}}}", query: "FileChannel.MapMode"},
		{name: "type prefix miss", source: "class Main {static class java{}}", query: "java.nio.channels.FileChannel"},
		{name: "import owns short name", source: "import java.nio.channels.FileChannel; class Main {static class java{}}", query: "FileChannel", want: "*stdjava.FileChannel"},
		{name: "source demand competitor", source: "package app; import java.nio.channels.*; import p.*; class Main{}", query: "FileLock", extra: map[string]string{"p/FileLock.java": "package p; public class FileLock{}"}},
		{name: "unrelated source member", source: "class Main {static class FileLock{}}", query: "java.nio.channels.FileLock", want: "*stdjava.JavaFileLock"},
	} {
		t.Run(row.name, func(t *testing.T) {
			sources := map[string]string{"app/Main.java": row.source}
			for n, s := range row.extra {
				sources[n] = s
			}
			ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
			before := ctx.Clone()
			imports := map[string]string{}
			for k, v := range ctx.currentFile.Imports {
				imports[k] = v
			}
			for i := 0; i < 3; i++ {
				got, ok := fileChannelCatalogRendered(t, row.query, ctx)
				if ok != (row.want != "") || got != row.want {
					t.Fatalf("mapping=%q,%v want=%q", got, ok, row.want)
				}
				if !reflect.DeepEqual(before, ctx) || !reflect.DeepEqual(imports, ctx.currentFile.Imports) {
					t.Fatal("catalog changed lexical/import context")
				}
			}
		})
	}
}

func TestFileChannelDeclarationCatalogBinderPrecedence(t *testing.T) {
	for _, query := range []string{"FileChannel", "MapMode", "java.nio.channels.FileChannel"} {
		binder := strings.Split(query, ".")[0]
		ctx := ordinaryDemandTestContext(t, map[string]string{"app/Main.java": "import java.nio.channels.FileChannel; class Main {<" + binder + "> void probe(){}}"}, "app/Main.java")
		ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
		if _, mapped := fileChannelCatalogRendered(t, query, ctx); mapped {
			t.Fatalf("captured binder %q acquired native declaration", binder)
		}
	}
}

func TestFileChannelDeclarationCatalogInheritedMapMode(t *testing.T) {
	sources := map[string]string{
		"model/Base.java":  "package model; import java.nio.channels.FileChannel; public abstract class Base extends FileChannel{}",
		"model/Child.java": "package model; public abstract class Child extends Base{}",
		"app/Main.java":    "package app; abstract class Main extends model.Child {<FileChannel> void probe(){}}",
	}
	ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("FileChannel", nil)}
	for _, query := range []string{"MapMode", "model.Child.MapMode"} {
		got, ok := fileChannelCatalogRendered(t, query, ctx)
		if !ok || got != "*stdjava.JavaFileChannelMapMode" {
			t.Fatalf("declaring ancestry lost for %s: %q,%v", query, got, ok)
		}
	}
}

func TestFileChannelDeclarationCatalogInheritedShields(t *testing.T) {
	for _, source := range []string{
		"import java.nio.channels.FileChannel; abstract class Main extends FileChannel {static class MapMode{}}",
		"class FileChannel{} class Main extends FileChannel{}",
		"class Main extends Missing{}",
		"class Main extends Other{} class Other extends Main{}",
	} {
		ctx := ordinaryDemandTestContext(t, map[string]string{"app/Main.java": source}, "app/Main.java")
		if scope := findQualifiedSourceClass("Main"); scope != nil {
			ctx.currentClass = scope
		}
		if _, mapped := fileChannelCatalogRendered(t, "MapMode", ctx); mapped {
			t.Fatalf("unproved/shadowed inheritance acquired MapMode: %s", source)
		}
	}
	ctx := ordinaryDemandTestContext(t, map[string]string{"app/Main.java": "import java.nio.channels.FileChannel; class Main{}"}, "app/Main.java")
	ctx.currentClass = &symbol.ClassScope{Superclass: "FileChannel"}
	if _, mapped := fileChannelCatalogRendered(t, "MapMode", ctx); mapped {
		t.Fatal("detached scope borrowed caller import file")
	}
}

func TestFileChannelDeclarationCatalogDonorContext(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"model/Donor.java": "package model; import java.nio.channels.ReadableByteChannel; public class Donor{public ReadableByteChannel result(){return null;}}",
		"app/Main.java":    "package app; class Main {<ReadableByteChannel> void probe(){}}",
	}, "app/Main.java")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	donor := classScopeCtx(findQualifiedSourceClass("model.Donor"), ctx)
	if got, ok := fileChannelCatalogRendered(t, "ReadableByteChannel", donor); !ok || got != "stdjava.JavaReadableByteChannel" {
		t.Fatalf("donor declaration recaptured: %q,%v", got, ok)
	}
	if _, ok := fileChannelCatalogRendered(t, "ReadableByteChannel", ctx); ok {
		t.Fatal("caller binder borrowed donor declaration")
	}
}

func TestFileChannelDeclarationCatalogGenericSourceAncestry(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"model/Base.java": "package model; import java.nio.channels.FileChannel; public abstract class Base<T> extends FileChannel{}",
		"app/Main.java":   "package app; abstract class Main extends model.Base<String>{}",
	}, "app/Main.java")
	if got, ok := fileChannelCatalogRendered(t, "MapMode", ctx); !ok || got != "*stdjava.JavaFileChannelMapMode" {
		t.Fatalf("source generic edge lost canonical root: %q,%v", got, ok)
	}
}
