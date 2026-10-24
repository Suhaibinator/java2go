package transpiler

import (
	"bytes"
	sitter "github.com/smacker/go-tree-sitter"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func TestFileChannelSourceConversionReturnAndLocal(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.nio.ByteBuffer; import java.nio.channels.FileChannel;
 abstract class ConvertRoot extends FileChannel {
  FileChannel itself(){return this;}
  static FileChannel expose(ConvertRoot source){FileChannel base=source;return base;}
 }`)
	if !strings.Contains(out, "__java2goChannelSource :=") {
		t.Fatalf("certified source projection absent:\n%s", out)
	}
}
func TestFileChannelSourceConversionSelectedFormalOrigin(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"model/Root.java":  "package model; public abstract class Root extends java.nio.channels.FileChannel{}",
		"model/Donor.java": "package model; import java.nio.channels.FileChannel; public class Donor{public static void take(FileChannel value){}}",
		"app/Caller.java":  "package app; public class Caller{static class java{} public static <FileChannel> void probe(model.Root source){model.Donor.take(source);model.Donor.take(null);}}",
	}, "app/Caller.java")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	ctx.className = ctx.currentClass.Class.Name
	ctx.executionContextName = "__java2goExecution"
	var calls []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(ctx.currentFile.Source) == "take" {
			calls = append(calls, n)
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if len(calls) != 2 {
		t.Fatal("missing actual source calls")
	}
	for i, call := range calls {
		var out bytes.Buffer
		if err := format.Node(&out, token.NewFileSet(), ParseExpr(call, ctx.currentFile.Source, ctx)); err != nil {
			t.Fatal(err)
		}
		text := out.String()
		if i == 0 && !strings.Contains(text, "Java2goFileChannelView()") {
			t.Fatalf("callee canonical formal recaptured by caller binder: %s", text)
		}
		if i == 1 && strings.Contains(text, "FileChannel") {
			t.Fatalf("literal null acquired caller binder storage: %s", text)
		}
	}
}

func TestFileChannelSourceConversionNominalAndBinderShields(t *testing.T) {
	for name, java := range map[string]string{
		"source":        `class FileChannel{} class Keep {static FileChannel value(FileChannel v){FileChannel local=v;local=v;return local;}}`,
		"binder":        `class Keep {static <FileChannel> FileChannel value(FileChannel v){FileChannel local=v;local=v;return local;}}`,
		"source-method": `class FileChannel{} class Keep {static void take(FileChannel v){} static void value(FileChannel v){take(v);}}`,
	} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, java)
			if strings.Contains(out, "__java2goChannelSource") || strings.Contains(out, "Java2goFileChannelView") {
				t.Fatalf("noncanonical declaration admitted:\n%s", out)
			}
		})
	}
}
func TestFileChannelSourceConversionCalleeNotCallerSourceHomonym(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"app/FileChannel.java": "package app;class FileChannel{}",
		"model/Root.java":      "package model;public abstract class Root extends java.nio.channels.FileChannel{}",
		"model/Donor.java":     "package model;import java.nio.channels.FileChannel;public class Donor{public static void take(FileChannel value){}}",
		"app/Caller.java":      "package app;public class Caller{public static void probe(model.Root source){model.Donor.take(source);}}",
	}, "app/Caller.java")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	ctx.className = ctx.currentClass.Class.Name
	ctx.executionContextName = "__java2goExecution"
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			call = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), ParseExpr(call, ctx.currentFile.Source, ctx)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Java2goFileChannelView()") {
		t.Fatalf("caller source class captured donor formal: %s", out.String())
	}
}
