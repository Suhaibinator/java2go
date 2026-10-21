package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func nativeChannelOriginContext(t *testing.T, source, owner string) Ctx {
	t.Helper()
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, owner)
	if ctx.currentClass == nil {
		t.Fatal("actual source owner missing")
	}
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	if ctx.localScope == nil {
		t.Fatal("actual source method missing")
	}
	return ctx
}
func TestDeclaredNativeChannelOriginIdentities(t *testing.T) {
	for _, test := range []struct{ prefix, typ, owner string }{
		{"import java.nio.ByteBuffer;", "ByteBuffer", "java.nio.ByteBuffer"},
		{"", "java.nio.ByteBuffer", "java.nio.ByteBuffer"},
		{"import java.nio.*;", "ByteBuffer", "java.nio.ByteBuffer"},
		{"import java.nio.channels.FileChannel;", "FileChannel", "java.nio.channels.FileChannel"},
		{"", "java.nio.channels.FileChannel", "java.nio.channels.FileChannel"},
		{"import java.nio.channels.*;", "FileChannel", "java.nio.channels.FileChannel"},
		{"import java.io.RandomAccessFile;", "RandomAccessFile", "java.io.RandomAccessFile"},
		{"", "java.io.RandomAccessFile", "java.io.RandomAccessFile"},
		{"import java.io.*;", "RandomAccessFile", "java.io.RandomAccessFile"},
	} {
		t.Run(test.prefix+test.typ, func(t *testing.T) {
			source := test.prefix + "class Probe{Object run(" + test.typ + " value){return value;}}"
			ctx := nativeChannelOriginContext(t, source, "Probe")
			got := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if got.unresolved || got.externalName != test.owner || got.nominalScope != nil || got.parameter != nil || got.arrayRank != 0 {
				t.Fatalf("declaration origin=%+v want canonical %s", got, test.owner)
			}
		})
	}
}
func TestDeclaredNativeChannelOriginShields(t *testing.T) {
	for _, test := range []struct{ source, owner string }{
		{"import foreign.ByteBuffer;class Probe{Object run(ByteBuffer value){return value;}}", "Probe"},
		{"import foreign.*;class Probe{Object run(ByteBuffer value){return value;}}", "Probe"},
		{"import java.nio.*;import foreign.*;class Probe{Object run(ByteBuffer value){return value;}}", "Probe"},
		{"import java.nio.ByteBuffer;class Probe{static class ByteBuffer{}Object run(ByteBuffer value){return value;}}", "Probe"},
		{"class Probe<ByteBuffer extends java.nio.ByteBuffer>{Object run(ByteBuffer value){return value;}}", "Probe"},
		{"class Probe{<FileChannel extends java.nio.channels.FileChannel>Object run(FileChannel value){return value;}}", "Probe"},
		{"class Outer<FileChannel extends java.nio.channels.FileChannel>{class Probe{Object run(FileChannel value){return value;}}}", "Outer.Probe"},
		{"class java{static class nio{static class ByteBuffer{}}}class Probe{Object run(java.nio.ByteBuffer value){return value;}}", "Probe"},
		{"abstract class Channel extends java.nio.channels.FileChannel{}class Probe{Object run(Channel value){return value;}}", "Probe"},
	} {
		t.Run(test.source, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, test.source, test.owner)
			got := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if got.externalName != "" {
				t.Fatalf("source/binder/unknown borrowed native declaration: %+v", got)
			}
		})
	}
	for _, owner := range []string{"java.nio.ByteBuffer", "java.nio.channels.FileChannel", "java.io.RandomAccessFile"} {
		for _, missing := range []string{"file", "class", "stale-file"} {
			t.Run(owner+missing, func(t *testing.T) {
				ctx := nativeChannelOriginContext(t, "class Probe{Object run(Object value){return value;}}", "Probe")
				switch missing {
				case "file":
					ctx.currentFile = nil
				case "class":
					ctx.currentClass = nil
				case "stale-file":
					ctx.currentFile = &symbol.FileScope{}
				}
				if got := declaredExternalNominalOwner(owner, nil, ctx); got != "" {
					t.Fatalf("missing/stale declaring ownership borrowed %s", got)
				}
			})
		}
	}
}
func TestDeclaredNativeChannelOriginDonorOwnership(t *testing.T) {
	source := `class Donor{static java.nio.channels.FileChannel channel(){return null;}}class Probe<FileChannel>{static class ByteBuffer{}Object run(){return Donor.channel();}}`
	ctx := nativeChannelOriginContext(t, source, "Probe")
	node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	var origin inferredJavaTypeOrigin
	actual, known := inferExprJavaType(node, ctx, ctx.currentFile.Source, &origin)
	if !known || origin.unresolved || origin.externalName != "java.nio.channels.FileChannel" || origin.declaringOwner == ctx.currentClass {
		t.Fatalf("actual=%q/%t donor origin=%+v", actual, known, origin)
	}
}

// A known source-package edge remains the preexisting namespace authority for
// unrelated names; the three-entry presence set cannot manufacture absence.
func TestDeclaredNativeChannelOriginPreservesExistingFallback(t *testing.T) {
	ctx := nativeChannelOriginContext(t, "package java.io;import java.io.*;class Probe{Object run(Long value){return value;}}", "java.io.Probe")
	got := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
	if got.unresolved || got.externalName != "java.lang.Long" {
		t.Fatalf("existing source-package fallback lost: %+v", got)
	}
}
