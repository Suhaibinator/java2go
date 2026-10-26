package transpiler

import "testing"

func TestFileChannelWritePositionalAdditionalOwnedForms(t *testing.T) {
	for _, source := range []string{
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return this.c.write(b,0L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{int run(FileChannel c,ByteBuffer b,Long p){return c.write(b,p);}}`,
		`import java.nio.ByteBuffer;import java.io.RandomAccessFile;class Probe{RandomAccessFile f;ByteBuffer b;int run(){return f.getChannel().write(b,0L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{int run(FileChannel __java2goChannel,ByteBuffer __java2goBuffer,char __java2goPosition){return __java2goChannel.write(__java2goBuffer,__java2goPosition);}}`,
		`class Donor{static java.nio.channels.FileChannel channel(){return null;}static java.nio.ByteBuffer buffer(){return null;}}class Probe{static class FileChannel{}static class ByteBuffer{}int run(){return Donor.channel().write(Donor.buffer(),0L);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			ctx.executionContextName = "__currentExecution"
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if result, admitted := inferIntrinsicMethodResultType(call, ctx, helper.File.Source); !admitted || result != "int" {
				t.Fatalf("owned read result = %q/%t", result, admitted)
			}
		})
	}
}

func TestFileChannelWritePositionalRejectionWithCurrentExecution(t *testing.T) {
	for _, source := range []string{
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.write(b,1.5);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer[] b;int run(){return c.write(b,1L);}}`,
		`import java.nio.channels.FileChannel;class ByteBuffer{}class Probe{FileChannel c;ByteBuffer b;int run(){return c.write(b,1L);}}`,
		`import foreign.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.write(b,1L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.write(b,unknown());}}`,
		`import java.nio.ByteBuffer;class Probe<FileChannel extends java.nio.channels.FileChannel>{FileChannel c;ByteBuffer b;int run(){return c.write(b,1L);}}`,
		`import java.nio.channels.FileChannel;class Probe<ByteBuffer extends java.nio.ByteBuffer>{FileChannel c;ByteBuffer b;int run(){return c.write(b,1L);}}`,
		`import java.nio.ByteBuffer;class Probe{FileChannel c;ByteBuffer b;int run(){return c.write(b,1L);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			ctx.executionContextName = "__currentExecution"
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if result, admitted := inferIntrinsicMethodResultType(call, ctx, helper.File.Source); admitted {
				t.Fatalf("unproved signature with real Execution admitted as %s", result)
			}
		})
	}
}
