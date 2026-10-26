package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

const channelRootASTSource = `import java.nio.ByteBuffer; import java.nio.channels.FileChannel;
abstract class ChannelRoot extends FileChannel {
 public int read(ByteBuffer b){return 1;}
 public int read(ByteBuffer b,long p){return 2;}
 protected void implCloseChannel(){}
}`

func TestFileChannelSourceConstructionRootProtocol(t *testing.T) {
	out := renderGoFileFromJava(t, channelRootASTSource)
	parsed, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	foundStorage := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok || len(field.Names) != 1 || field.Names[0].Name != "__java2goFileChannel" {
			return true
		}
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			return true
		}
		selector, ok := star.X.(*ast.SelectorExpr)
		foundStorage = ok && selector.Sel.Name == "FileChannel"
		return true
	})
	if !foundStorage {
		t.Fatalf("named canonical carrier storage missing:\n%s", out)
	}
	for _, want := range []string{"*stdjava.ObjectInfo", "NewGeneratedObjectInfo(", "NewFileChannelSourceReference(", "Java2goFileChannelView()", "Java2goFileChannelReadExecution(", "Java2goFileChannelReadAtExecution(", "Java2goFileChannelImplCloseExecution(", "stdjava.JavaNIOFileChannelType"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing canonical root protocol %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\n\t*FileChannel\n") {
		t.Fatal("anonymous external superclass storage retained")
	}
}
func TestFileChannelSourceConstructionHomonymShield(t *testing.T) {
	out := renderGoFileFromJava(t, `class FileChannel {} class Ordinary extends FileChannel {}`)
	if strings.Contains(out, "NewFileChannelSourceReference(") || strings.Contains(out, "__java2goFileChannel ") {
		t.Fatalf("source homonym admitted canonical construction:\n%s", out)
	}
}
