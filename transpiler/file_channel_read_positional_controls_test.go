package transpiler

import "testing"

func TestFileChannelReadPositionalAdditionalOwnedForms(t *testing.T) {
	for _, source := range []string{
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return this.c.read(b,0L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{int run(FileChannel c,ByteBuffer b,Long p){return c.read(b,p);}}`,
		`import java.nio.ByteBuffer;import java.io.RandomAccessFile;class Probe{RandomAccessFile f;ByteBuffer b;int run(){return f.getChannel().read(b,0L);}}`,
		`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{int run(FileChannel __java2goChannel,ByteBuffer __java2goBuffer,char __java2goPosition){return __java2goChannel.read(__java2goBuffer,__java2goPosition);}}`,
		`class Donor{static java.nio.channels.FileChannel channel(){return null;}static java.nio.ByteBuffer buffer(){return null;}}class Probe{static class FileChannel{}static class ByteBuffer{}int run(){return Donor.channel().read(Donor.buffer(),0L);}}`,
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

func TestFileChannelReadPositionalRejectionWithCurrentExecution(t *testing.T) {
	for _, test := range []struct {
		source       string
		wantAdmitted bool
	}{
		{`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1.5);}}`, false},
		{`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer[] b;int run(){return c.read(b,1L);}}`, false},
		{`import java.nio.channels.FileChannel;class ByteBuffer{}class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`, false},
		{`import foreign.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`, false},
		{`import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,unknown());}}`, false},
		{`import java.nio.ByteBuffer;class Probe<FileChannel extends java.nio.channels.FileChannel>{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`, false},
		{`import java.nio.channels.FileChannel;class Probe<ByteBuffer extends java.nio.ByteBuffer>{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`, false},
		{`import java.nio.ByteBuffer;class Probe{FileChannel c;ByteBuffer b;int run(){return c.read(b,1L);}}`, false},
		{`import java.nio.channels.FileChannel;class Probe{void run(FileChannel c){Runnable r=()->c.read(null,0L);}}`, true},
	} {
		t.Run(test.source, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			ctx.executionContextName = "__currentExecution"
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			result, admitted := inferIntrinsicMethodResultType(call, ctx, helper.File.Source)
			if admitted != test.wantAdmitted || admitted && result != "int" {
				t.Fatalf("selected signature result = %q/%t, want admission %t", result, admitted, test.wantAdmitted)
			}
			if test.wantAdmitted {
				description, described := fileChannelReadNullDescription(call.ChildByFieldName("object"), ctx, helper.File.Source)
				if !described || description != `"<parameter1>"` {
					t.Fatalf("captured declaration null description = %q/%t", description, described)
				}
			}
		})
	}
}
