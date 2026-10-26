package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestFileChannelSourceWriteCanonicalDescriptors(t *testing.T) {
	for name, java := range map[string]string{
		"explicit": `abstract class Probe extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer b){return 1;}public int write(java.nio.ByteBuffer b,long p){return 2;}}`,
		"imported": `import java.nio.ByteBuffer;import java.nio.channels.FileChannel;abstract class Probe extends FileChannel {public int write(ByteBuffer b){return 1;}public int write(ByteBuffer b,long p){return 2;}}`,
		"demand":   `import java.nio.*;import java.nio.channels.*;abstract class Probe extends FileChannel {public int write(ByteBuffer b){return 1;}public int write(ByteBuffer b,long p){return 2;}}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": java}, "Probe.java")
			methods := ctx.currentClass.FindMethod().ByOriginalName("write")
			if len(methods) != 2 {
				t.Fatalf("actual declarations=%d", len(methods))
			}
			for _, method := range methods {
				want := "Java2goFileChannelWriteExecution"
				if len(method.Parameters) == 2 {
					want = "Java2goFileChannelWriteAtExecution"
				}
				if got := fileChannelSourceProtocolSlot(method, ctx.currentClass, ctx); got != want {
					t.Errorf("descriptor %s/%d slot=%q want=%q", method.OriginalName, len(method.Parameters), got, want)
				}
			}
		})
	}
}

func TestFileChannelSourceWriteDeclarationShields(t *testing.T) {
	for name, java := range map[string]string{
		"source-channel":        `class FileChannel{}class Probe extends FileChannel {public int write(java.nio.ByteBuffer b,long p){return 1;}}`,
		"unknown-channel":       `abstract class Probe extends MissingChannel {public int write(java.nio.ByteBuffer b,long p){return 1;}}`,
		"source-buffer":         `import java.nio.ByteBuffer;abstract class Probe extends java.nio.channels.FileChannel {static class ByteBuffer{}public int write(ByteBuffer b,long p){return 1;}}`,
		"foreign-buffer":        `import foreign.ByteBuffer;abstract class Probe extends java.nio.channels.FileChannel {public int write(ByteBuffer b,long p){return 1;}}`,
		"buffer-binder":         `abstract class Probe<ByteBuffer> extends java.nio.channels.FileChannel {public int write(ByteBuffer b,long p){return 1;}}`,
		"package-prefix-binder": `abstract class Probe<java> extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer b,long p){return 1;}}`,
		"array-buffer":          `abstract class Probe extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer[] b,long p){return 1;}}`,
		"boxed-position":        `abstract class Probe extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer b,Long p){return 1;}}`,
		"int-position":          `abstract class Probe extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer b,int p){return 1;}}`,
		"generic-method":        `abstract class Probe extends java.nio.channels.FileChannel {public <T> int write(java.nio.ByteBuffer b,long p){return 1;}}`,
		"private-method":        `abstract class Probe extends java.nio.channels.FileChannel {private int write(java.nio.ByteBuffer b,long p){return 1;}}`,
		"static-method":         `abstract class Probe extends java.nio.channels.FileChannel {public static int write(java.nio.ByteBuffer b,long p){return 1;}}`,
		"varargs":               `abstract class Probe extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer b,long... p){return 1;}}`,
		"wrong-result":          `abstract class Probe extends java.nio.channels.FileChannel {public long write(java.nio.ByteBuffer b,long p){return 1;}}`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": java}, "Probe.java")
			for _, method := range ctx.currentClass.FindMethod().ByOriginalName("write") {
				if got := fileChannelSourceProtocolSlot(method, ctx.currentClass, ctx); got != "" {
					t.Fatalf("unproved declaration admitted %q", got)
				}
			}
		})
	}
}

func TestFileChannelSourceWriteDeclaringOwnerContext(t *testing.T) {
	files := map[string]string{
		"p/Parent.java": `package p;import java.nio.ByteBuffer;public abstract class Parent extends java.nio.channels.FileChannel {public int write(ByteBuffer b){return 1;}public int write(ByteBuffer b,long p){return 2;}}`,
		"q/Probe.java":  `package q;public abstract class Probe<ByteBuffer> extends p.Parent {static class FileChannel{}static class java{}}`,
	}
	ctx := ordinaryDemandTestContext(t, files, "q/Probe.java")
	parent := resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass)
	if parent == nil {
		t.Fatal("actual donor ancestor missing")
	}
	for _, method := range parent.FindMethod().ByOriginalName("write") {
		if got := fileChannelSourceProtocolSlot(method, parent, ctx); !strings.HasPrefix(got, "Java2goFileChannelWrite") {
			t.Fatalf("caller captured donor descriptor: %q", got)
		}
	}
}

func TestFileChannelSourceWriteMostDerivedProtocol(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.nio.ByteBuffer;
 abstract class WriteRoot extends java.nio.channels.FileChannel {
  public int read(ByteBuffer b){return 1;}public int read(ByteBuffer b,long p){return 2;}protected void implCloseChannel(){}
  public int write(ByteBuffer b){return 3;}public int write(ByteBuffer b,long p){return 4;}
 }
 abstract class WriteChild extends WriteRoot {
  public int write(ByteBuffer b){return super.write(b)+10;}
  public int write(ByteBuffer b,long p){return super.write(b,p)+20;}
  int invoke(ByteBuffer b){return write(b)+write(b,9L);}
 }`)
	parsed, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	slotSignatures := map[string]int{}
	ast.Inspect(parsed, func(n ast.Node) bool {
		d, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if d.Name.Name == "Java2goFileChannelWriteExecution" || d.Name.Name == "Java2goFileChannelWriteAtExecution" {
			want := 2
			if d.Name.Name == "Java2goFileChannelWriteAtExecution" {
				want = 3
			}
			if len(d.Type.Params.List) != want || d.Type.Results == nil {
				t.Errorf("wrong stable source descriptor for %s", d.Name.Name)
			}
			slotSignatures[d.Name.Name]++
		}
		return true
	})
	for _, slot := range []string{"Java2goFileChannelWriteExecution", "Java2goFileChannelWriteAtExecution"} {
		if slotSignatures[slot] != 2 {
			t.Errorf("%s actual root+child body bridges=%d", slot, slotSignatures[slot])
		}
	}
	for _, want := range []string{"Write:", "WriteAt:", "FileChannelWriteVirtualExecution(__java2goExecution,", "FileChannelWriteAtVirtualExecution(__java2goExecution,"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing declaration-selected source writer %q\n%s", want, out)
		}
	}
	if strings.Count(out, "NewFileChannelSourceReference(") != 1 {
		t.Errorf("source hierarchy carrier allocations=%d", strings.Count(out, "NewFileChannelSourceReference("))
	}
	// Explicit super bypasses the child dispatcher and executes the real parent stable entry.
	for _, want := range []string{".writeRoot.Java2goFileChannelWriteExecution(__java2goExecution,", ".writeRoot.Java2goFileChannelWriteAtExecution(__java2goExecution,"} {
		if !strings.Contains(out, want) {
			t.Errorf("explicit super writer lost direct ancestor body %q", want)
		}
	}
}
