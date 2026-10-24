package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestFileChannelWriteStaticReturnDeclarationProvenance(t *testing.T) {
	for _, tc := range []struct{ name, java, description string }{
		{"unqualified", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{static FileChannel receiver(FileChannel c){return c;}int run(FileChannel c,ByteBuffer b){return receiver(c).write(b,3L);}}`, `the return value of \"Probe.receiver(java.nio.channels.FileChannel)\"`},
		{"relative", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{static FileChannel receiver(FileChannel c){return c;}int run(FileChannel c,ByteBuffer b){return receiver(c).write(b);}}`, `the return value of \"Probe.receiver(java.nio.channels.FileChannel)\"`},
		{"qualified-fixed", `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Donor{static FileChannel receiver(FileChannel c,int flag,ByteBuffer b){return c;}}class Probe{int run(FileChannel c,ByteBuffer b){return Donor.receiver(c,2,b).write(b,3L);}}`, `the return value of \"Donor.receiver(java.nio.channels.FileChannel, int, java.nio.ByteBuffer)\"`},
		{"caller-binders", `class Donor{static java.nio.channels.FileChannel receiver(java.nio.channels.FileChannel c){return c;}}class Probe{<FileChannel,ByteBuffer>int run(java.nio.channels.FileChannel c,java.nio.ByteBuffer b){return Donor.receiver(c).write(b,3L);}}`, `the return value of \"Donor.receiver(java.nio.channels.FileChannel)\"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, tc.java)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			ctx.executionContextName = "__currentExecution"
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if result, ok := inferIntrinsicMethodResultType(call, ctx, h.File.Source); !ok || result != "int" {
				t.Fatalf("declared static-return write = %q/%t, want int/true", result, ok)
			}
			expr, ok := tryInstanceIntrinsic(call.ChildByFieldName("object"), "write", h.File.Source, ctx)
			if !ok || expr == nil {
				t.Fatal("result/emitter disagreement")
			}
			var out bytes.Buffer
			if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			if text := out.String(); !strings.Contains(text, tc.description) || !strings.Contains(text, "VirtualExecution(__currentExecution,") {
				t.Fatalf("actual declaration diagnostic/currentExecution lost: %s", text)
			}
		})
	}
}
func TestFileChannelWriteStaticReturnUnknownDescriptorsStayHeld(t *testing.T) {
	for _, java := range []string{
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{static FileChannel receiver(FileChannel c){return c;}int run(ByteBuffer b){return receiver(unknown()).write(b,1L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{static<T> FileChannel receiver(FileChannel c){return c;}int run(FileChannel c,ByteBuffer b){return receiver(c).write(b,1L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{static FileChannel receiver(FileChannel...c){return null;}int run(FileChannel c,ByteBuffer b){return receiver(c).write(b,1L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{static FileChannel receiver(Missing c){return null;}int run(ByteBuffer b){return receiver(null).write(b,1L);}}`,
	} {
		h := setupParseHelper(t, java)
		ctx := h.Ctx.Clone()
		ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
		ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
		ctx.executionContextName = "__currentExecution"
		call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
		if result, ok := inferIntrinsicMethodResultType(call, ctx, h.File.Source); ok {
			t.Fatalf("unproved static receiver admitted %q: %s", result, java)
		}
	}
}
