package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestFileChannelSourceDispatchAbstractContract(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.nio.ByteBuffer; import java.nio.channels.FileChannel;
 abstract class DispatchRoot extends FileChannel {
  public abstract int read(ByteBuffer b);
  public abstract int read(ByteBuffer b,long p);
  protected abstract void implCloseChannel();
  int invoke(ByteBuffer b){return read(b,3L);}
 }`)
	file, err := parser.ParseFile(token.NewFileSet(), "out.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typ, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := typ.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			if !strings.Contains(strings.ToLower(typ.Name.Name), "dispatchroot") {
				continue
			}
			found++
			names := map[string]bool{}
			for _, field := range iface.Methods.List {
				for _, name := range field.Names {
					names[name.Name] = true
				}
			}
			for _, slot := range []string{fileChannelSourceReadSlot, fileChannelSourceReadAtSlot, fileChannelSourceCloseSlot, fileChannelSourceViewMethod} {
				if !names[slot] {
					t.Errorf("%s missing %s", typ.Name.Name, slot)
				}
			}
		}
	}
	if found < 3 {
		t.Fatalf("missing source contract interfaces: %d\n%s", found, out)
	}
}
