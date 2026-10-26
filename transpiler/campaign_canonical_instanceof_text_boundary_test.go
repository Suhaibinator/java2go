package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestCanonicalInstanceofStaticBooleanType(t *testing.T) {
	file := parsing.SourceFile{Source: []byte(`class Probe {
 boolean plain(Object value) { return value instanceof String; }
 boolean grouped(Object value) { return (value instanceof String); }
 boolean pattern(Object value) { return value instanceof String text; }
}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	found := 0
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node.Type() == "instanceof_expression" || node.Type() == "parenthesized_expression" {
			actual, known := inferExprJavaType(node, Ctx{}, file.Source)
			if !known || actual != "boolean" {
				t.Fatalf("%s has type %q, known=%v; expected primitive boolean", node.Content(file.Source), actual, known)
			}
			found++
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(file.Ast)
	if found != 4 {
		t.Fatalf("expected three instanceof expressions and one grouped expression, found %d", found)
	}
}

func TestCampaignCanonicalInstanceofPrimitiveTextJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>instanceof-text</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static int effects;
 static Object probe(int value) { effects=effects*10+value; return value==1 ? "text" : null; }
 public static void main(String[] args) {
  System.out.println("direct="+(probe(1) instanceof String)+":"+(probe(2) instanceof String)+":"+effects);
  var stored=probe(3) instanceof String;
  System.out.println("stored="+stored+":"+String.valueOf(probe(4) instanceof String)+":"+effects);
 }
}`,
	}, "probe.Main", "direct=true:false:12\nstored=false:false:1234\n")
}
