package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestFileChannelReadPositionalCanonicalResultAndFacade(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"long", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,3L);}}`},
		{"int-widening", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,-3);}}`},
		{"qualified", `class Probe{java.nio.channels.FileChannel c;java.nio.ByteBuffer b;int run(){return c.read(b,3L);}}`},
		{"typed-null-buffer", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;int run(){return c.read((ByteBuffer)null,-1L);}}`},
		{"unique-bare-null-buffer", `import java.nio.channels.FileChannel;class Probe{FileChannel c;int run(){return c.read(null,-1L);}}`},
		{"donor-under-caller-binders", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Donor{static FileChannel channel(){return null;}static ByteBuffer buffer(){return null;}}class Probe{<FileChannel,ByteBuffer>int run(){return Donor.channel().read(Donor.buffer(),3L);}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			ctx.executionContextName = "__currentExecution"
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			result, admitted := inferIntrinsicMethodResultType(call, ctx, helper.File.Source)
			if !admitted || result != "int" {
				t.Fatalf("canonical positional read result = %q/%t, want int/true", result, admitted)
			}
			expression, rewritten := tryInstanceIntrinsic(call.ChildByFieldName("object"), "read", helper.File.Source, ctx)
			if !rewritten || expression == nil {
				t.Fatal("same canonical signature was not emitted")
			}
			var out bytes.Buffer
			if err := printer.Fprint(&out, token.NewFileSet(), expression); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			if !strings.Contains(text, "FileChannelReadAtVirtualExecution(__currentExecution,") || strings.Contains(text, "FileChannelReadAtExecution(") || strings.Contains(text, "NewExecution(") {
				t.Fatalf("positional read lost virtual/current-Execution contract: %s", text)
			}
		})
	}
}

func TestFileChannelReadRelativeUsesCurrentVirtualFacade(t *testing.T) {
	helper := setupParseHelper(t, `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b);}}`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	ctx.executionContextName = "__currentExecution"
	call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	expression, rewritten := tryInstanceIntrinsic(call.ChildByFieldName("object"), "read", helper.File.Source, ctx)
	if !rewritten || expression == nil {
		t.Fatal("existing relative read was not emitted")
	}
	var out bytes.Buffer
	if err := printer.Fprint(&out, token.NewFileSet(), expression); err != nil {
		t.Fatal(err)
	}
	if text := out.String(); !strings.Contains(text, "FileChannelReadVirtualExecution(__currentExecution,") || strings.Contains(text, "FileChannelReadExecution(") {
		t.Fatalf("relative read remains native-only: %s", text)
	}
}

func TestFileChannelReadPositionalRefusesUnprovedSignature(t *testing.T) {
	for _, source := range []string{
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1.5);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer[] b;int run(){return c.read(b,1L);}}`,
		`import java.nio.channels.FileChannel;class ByteBuffer{}class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`,
		`import foreign.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,unknown());}}`,
		`import java.nio.ByteBuffer;class Probe<FileChannel extends java.nio.channels.FileChannel>{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`,
		`import java.nio.channels.FileChannel;class Probe<ByteBuffer extends java.nio.ByteBuffer>{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if result, admitted := inferIntrinsicMethodResultType(call, ctx, helper.File.Source); admitted {
				t.Fatalf("unproved signature admitted as %s", result)
			}
		})
	}
}

func TestFileChannelReadPositionalSourceOverloadNotIntercepted(t *testing.T) {
	source := `class ByteBuffer{}class FileChannel{int read(ByteBuffer b,long p){return 41;}String read(String b,long p){return b;}}class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`
	generated := renderGoFileFromJava(t, source)
	for _, forbidden := range []string{"FileChannelReadAtVirtualExecution", "FileChannelReadAtExecution", "FileChannelReadVirtualExecution"} {
		if strings.Contains(generated, forbidden) {
			t.Fatalf("ordinary source overload intercepted by %s", forbidden)
		}
	}
}
