package transpiler

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func rafFile209OriginalDigest(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("campaign_digest_io_test.go")
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "const source = `"
	begin := strings.Index(string(raw), prefix)
	if begin < 0 {
		t.Fatal("original DigestIO Java constant missing")
	}
	tail := string(raw)[begin+len(prefix):]
	end := strings.IndexByte(tail, '`')
	if end < 0 {
		t.Fatal("original DigestIO Java constant unterminated")
	}
	java := tail[:end]
	if fmt.Sprintf("%x", sha256.Sum256([]byte(java))) != "2d7a621a71793f0f36392de853a9a120a216e19dcaefaa9744e0534192191a43" {
		t.Fatal("original DigestIO Java changed")
	}
	return java
}

func TestRAFFile209CanonicalConstructor(t *testing.T) {
	cases := []struct{ name, java, invocation string }{
		{"native-var", `import java.io.*;class Probe{Object run()throws Exception{File file=new File("raf-path");return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"qualified-new", `class Probe{Object run()throws Exception{return new java.io.RandomAccessFile(new java.io.File("raf-path"),"r");}}`, `new java.io.RandomAccessFile(new java.io.File("raf-path"),"r")`},
		{"typed-null", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile((java.io.File)null,"r");}}`, `new RandomAccessFile((java.io.File)null,"r")`},
		{"canonical-temp-factory", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(File.createTempFile("raf",".bin"),"r");}}`, `new RandomAccessFile(File.createTempFile("raf",".bin"),"r")`},
		{"effectful-mode", `import java.io.*;class Probe{static int calls;static String mode(){calls=calls+1;return "r";}Object run()throws Exception{return new RandomAccessFile(File.createTempFile("raf",".bin"),mode());}}`, `new RandomAccessFile(File.createTempFile("raf",".bin"),mode())`},
		{"original-digest", rafFile209OriginalDigest(t), `new RandomAccessFile(file, "r")`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			strictRoutingState(t)
			defer func() {
				if observed := recover(); observed != nil {
					failure, ok := observed.(strictModeError)
					if !ok || failure.diagnostic.Kind != "intrinsic invocation" || failure.diagnostic.NodeType != "object_creation_expression" || failure.diagnostic.Message != test.invocation {
						t.Fatalf("unexpected compiler frontier: %v", observed)
					}
					if len(Diagnostics()) != 1 {
						t.Fatalf("unexpected diagnostic multiplicity: %v", Diagnostics())
					}
					t.Errorf("MISSING_CANONICAL_FILE_CTOR: %s", test.invocation)
				}
			}()
			generated := renderGoFileFromJava(t, test.java)
			if len(Diagnostics()) != 0 {
				t.Fatalf("successful File constructor retained diagnostics: %v", Diagnostics())
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			ast.Inspect(parsed, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "NewRandomAccessFileFileExecution" {
					return true
				}
				calls++
				if len(call.Args) != 3 {
					t.Errorf("File constructor must retain execution, File, mode: %d arguments", len(call.Args))
				}
				return true
			})
			if calls != 1 {
				t.Fatalf("canonical File constructor call count=%d, want1", calls)
			}
		})
	}
}

func TestRAFFile209HeldBoundaries(t *testing.T) {
	cases := []struct{ name, java, invocation string }{
		{"unknown-formal", `import java.io.*;class Probe{Object run(File file)throws Exception{return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"unknown-field", `import java.io.*;class Probe{File file;Object run()throws Exception{return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"unknown-return", `import java.io.*;class Probe{static File path(File file){return file;}Object run(File file)throws Exception{return new RandomAccessFile(path(file),"r");}}`, `new RandomAccessFile(path(file),"r")`},
		{"reassigned-native-local", `import java.io.*;class Probe{Object run(File other)throws Exception{File file=new File("x");file=other;return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"identity-cast-unknown", `import java.io.*;class Probe{Object run(File file)throws Exception{return new RandomAccessFile((java.io.File)file,"r");}}`, `new RandomAccessFile((java.io.File)file,"r")`},
		{"bare-null", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(null,"r");}}`, `new RandomAccessFile(null,"r")`},
		{"source-file-override", `import java.io.*;class SourceFile extends File{SourceFile(){super("x");}public String getPath(){return "y";}}class Probe{Object run(SourceFile file)throws Exception{return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"anonymous-file", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"){public String getPath(){return "y";}},"r");}}`, `new RandomAccessFile(new File("x"){public String getPath(){return "y";}},"r")`},
		{"cast-source-file", `import java.io.*;class SourceFile extends File{SourceFile(){super("x");}public String getPath(){return "y";}}class Probe{Object run(SourceFile file)throws Exception{return new RandomAccessFile((java.io.File)file,"r");}}`, `new RandomAccessFile((java.io.File)file,"r")`},
		{"foreign-file", `import java.io.RandomAccessFile;import foreign.File;class Probe{Object run(File file)throws Exception{return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"type-binder-file", `import java.io.RandomAccessFile;class Probe<File>{Object run(File file)throws Exception{return new RandomAccessFile(file,"r");}}`, `new RandomAccessFile(file,"r")`},
		{"nonstring-mode", `import java.io.*;class Probe{Object run(File file)throws Exception{return new RandomAccessFile(file,7);}}`, `new RandomAccessFile(file,7)`},
		{"wrong-arity", `import java.io.*;class Probe{Object run(File file)throws Exception{return new RandomAccessFile(file);}}`, `new RandomAccessFile(file)`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			strictRoutingState(t)
			defer func() {
				observed := recover()
				failure, ok := observed.(strictModeError)
				if !ok || failure.diagnostic.Kind != "intrinsic invocation" || failure.diagnostic.NodeType != "object_creation_expression" || failure.diagnostic.Message != test.invocation || len(Diagnostics()) != 1 {
					t.Fatalf("held boundary needs sole RAF constructor refusal: %v / %v", observed, Diagnostics())
				}
			}()
			renderGoFileFromJava(t, test.java)
			t.Fatal("held File constructor escaped strict refusal")
		})
	}
}
