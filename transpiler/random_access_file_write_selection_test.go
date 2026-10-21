package transpiler

import (
	"strings"
	"testing"
)

func TestRAFWriteNativeAllocationProducer(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"resource", `import java.io.RandomAccessFile;class Probe{void run(String path){try(RandomAccessFile f=new RandomAccessFile(path,"rw")){f.setLength(0);f.write(new byte[]{1,-1});}}}`},
		{"qualified", `class Probe{void run(String path){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw")){f.setLength(2147483648L);f.write((byte[])null);}}}`},
		{"direct", `import java.io.RandomAccessFile;class Probe{void run(String path){new RandomAccessFile(path,"rw").setLength(0);new RandomAccessFile(path,"rw").write(new byte[0]);}}`},
		{"resource-capture", `import java.io.RandomAccessFile;class Probe{void run(String path){try(RandomAccessFile f=new RandomAccessFile(path,"rw")){Runnable r=()->{f.setLength(2L);f.write(new byte[0]);};r.run();}}}`},
		{"caller-binder-shield", `class Probe{<RandomAccessFile>void run(String path){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw")){f.setLength(0);f.write(new byte[0]);}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetDiagnostics()
			setStrictMode(false)
			t.Cleanup(func() { resetDiagnostics(); setStrictMode(false) })
			out := renderGoFileFromJava(t, tc.source)
			for _, operation := range []string{"RandomAccessFileSetLengthExecution(", "RandomAccessFileWriteBytesExecution("} {
				if strings.Count(out, operation) != 1 {
					t.Fatalf("operation %s missing/duplicated:\n%s", operation, out)
				}
			}
			if len(Diagnostics()) != 0 {
				t.Fatalf("accepted allocation diagnosed: %+v", Diagnostics())
			}
			if strings.Contains(out, ".setLength(") || strings.Contains(out, ".write(") {
				t.Fatalf("unlowered RAF member:\n%s", out)
			}
		})
	}
}

func TestRAFWriteNativeAllocationUnknownFlowHeld(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"parameter", `class Probe{void run(java.io.RandomAccessFile f){f.write(new byte[0]);}}`},
		{"getter", `class Probe{static java.io.RandomAccessFile file(){return null;}void run(){Probe.file().setLength(0);}}`},
		{"ordinary-local", `class Probe{void run(String path){java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw");f.setLength(0);}}`},
		{"mutable-alias", `class Probe{void run(String path,java.io.RandomAccessFile other){java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw");f=other;f.write(new byte[0]);}}`},
		{"unknown-array", `class Probe{void run(String path,int[] b){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw")){f.write(b);}}}`},
		{"rank-two", `class Probe{void run(String path,byte[][] b){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw")){f.write(b);}}}`},
		{"wrong-overload", `class Probe{void run(String path){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw")){f.write(1);}}}`},
		{"floating-length", `class Probe{void run(String path){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(path,"rw")){f.setLength(1.5);}}}`},
		{"resource-unknown-initializer", `class Probe{void run(java.io.RandomAccessFile other){try(java.io.RandomAccessFile f=other){f.setLength(0);}}}`},
		{"existing-resource", `class Probe{void run(java.io.RandomAccessFile f){try(f){f.setLength(0);}}}`},
	} {
		for _, strict := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "-permissive", true: "-strict"}[strict], func(t *testing.T) {
				resetDiagnostics()
				setStrictMode(strict)
				t.Cleanup(func() { resetDiagnostics(); setStrictMode(false) })
				var caught any
				var out string
				func() { defer func() { caught = recover() }(); out = renderGoFileFromJava(t, tc.source) }()
				ds := Diagnostics()
				if len(ds) != 1 || ds[0].Kind != "intrinsic invocation" {
					t.Fatalf("exact-one refusal = %+v; panic=%v; out=%s", ds, caught, out)
				}
				if strict {
					if _, ok := caught.(strictModeError); !ok {
						t.Fatalf("strict panic %T:%v", caught, caught)
					}
				} else {
					if caught != nil || !strings.Contains(out, "unsupported intrinsic invocation") || strings.Contains(out, "RandomAccessFileWriteBytesExecution(") || strings.Contains(out, "RandomAccessFileSetLengthExecution(") {
						t.Fatalf("bad placeholder/panic %v:\n%s", caught, out)
					}
				}
			})
		}
	}
}

func TestRAFWriteSourceDeclarationUnchanged(t *testing.T) {
	for _, source := range []string{
		`class RandomAccessFile{void setLength(long n){}void write(byte[] b){}}class Probe{void run(RandomAccessFile f){f.setLength(0);f.write(new byte[0]);}}`,
		`class Probe<RandomAccessFile>{void run(RandomAccessFile f){f.write(new byte[0]);}}`,
		`class Child extends java.io.RandomAccessFile{Child(String p){super(p,"rw");}void write(byte[] b){}}class Probe{void run(Child f){f.write(new byte[0]);}}`,
	} {
		resetDiagnostics()
		setStrictMode(false)
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "RandomAccessFileWriteBytesExecution(") || strings.Contains(out, "RandomAccessFileSetLengthExecution(") {
			t.Fatalf("source or binder intercepted:\n%s", out)
		}
	}
	resetDiagnostics()
	setStrictMode(false)
}

func TestRAFWriteResultMetadataSameSignature(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   bool
	}{
		{`class Probe{void run(String path){new java.io.RandomAccessFile(path,"rw").write(new byte[]{1});}}`, true},
		{`class Probe{void run(String path){new java.io.RandomAccessFile(path,"rw").setLength(2L);}}`, true},
		{`class Probe{void run(java.io.RandomAccessFile f){f.write(new byte[0]);}}`, false},
		{`class Probe{void run(String path,int[] b){new java.io.RandomAccessFile(path,"rw").write(b);}}`, false},
	} {
		helper := setupParseHelper(t, tc.source)
		ctx := helper.Ctx.Clone()
		ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
		ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
		ctx.executionContextName = "__currentExecution"
		ctx.expectedType = "byte[]"
		ctx.expectedTypeRoot = ctx.localScope.DeclarationNode
		call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
		result, admitted := inferIntrinsicMethodResultType(call, ctx, helper.File.Source)
		if admitted != tc.want || admitted && result != "void" {
			t.Fatalf("result %s/%t want void/%t: %s", result, admitted, tc.want, tc.source)
		}
	}
}

func TestRAFWriteLexicalAllocationIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		writes       int
	}{
		{"sibling", `class Probe{void run(String p,java.io.RandomAccessFile other){try(java.io.RandomAccessFile f=new java.io.RandomAccessFile(p,"rw")){f.write(new byte[0]);} {java.io.RandomAccessFile f=other;f.write(new byte[0]);}}}`, 1},
		{"parameter-lambda", `class Probe{void run(String p,java.io.RandomAccessFile f){Runnable r=()->{f.write(new byte[0]);};}}`, 0},
		{"anonymous", `class Probe{void run(String p){new java.io.RandomAccessFile(p,"rw"){}.write(new byte[0]);}}`, 0},
		{"resource-source-subtype", `class Child extends java.io.RandomAccessFile{Child(String p){super(p,"rw");}void write(byte[] b){}}class Probe{void run(String p){try(java.io.RandomAccessFile f=new Child(p)){f.write(new byte[0]);}}}`, 0},
		{"field", `class Probe{java.io.RandomAccessFile f;void run(){f.write(new byte[0]);}}`, 0},
		{"foreign", `import foreign.RandomAccessFile;class Probe{void run(String p){try(RandomAccessFile f=new RandomAccessFile(p,"rw")){f.write(new byte[0]);}}}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetDiagnostics()
			setStrictMode(false)
			t.Cleanup(func() { resetDiagnostics(); setStrictMode(false) })
			out := renderGoFileFromJava(t, tc.source)
			if strings.Count(out, "RandomAccessFileWriteBytesExecution(") != tc.writes {
				t.Fatalf("lexical certificate admitted wrong allocation:\n%s", out)
			}
		})
	}
}
