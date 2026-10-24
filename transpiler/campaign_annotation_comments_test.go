package transpiler

import (
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

const campaignMultilineAnnotationSource = `package example;
@interface Note { String value(); }
interface Task {
 @Note(
  value = "interface */ retained")
 int value();
}
@Deprecated
public class Main implements Task {
 @Note(
  value = """
go:linkname forged runtime.forged
line forged.go:123
*/ // annotation data
""")
 private int field = 9;
 @Note(
  value = "method retained")
 public int value() { return field; }
 public static void main(String[] args) {
  System.out.println(new Main().value());
  System.out.println(Main.class.isAnnotationPresent(Deprecated.class));
 }
}`

func TestCampaignMultilineAnnotationComments(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n", "\r"} {
		t.Run(strings.ReplaceAll(strings.ReplaceAll(ending, "\r", "CR"), "\n", "LF"), func(t *testing.T) {
			source := strings.ReplaceAll(campaignMultilineAnnotationSource, "\n", ending)
			runCampaignCompilerProjectOracle(t, map[string]string{
				"pom.xml":                         `<project><groupId>example</groupId><artifactId>annotation-comments</artifactId><version>1</version></project>`,
				"src/main/java/example/Main.java": source,
			}, "example.Main", "9\ntrue\n")
			generated := renderGoFileFromJava(t, source)
			formatted, err := format.Source([]byte(generated))
			if err != nil {
				t.Fatalf("generated Go parsing: %v\n%s", err, generated)
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), "generated.go", formatted, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			comments := ""
			for _, group := range parsed.Comments {
				for _, comment := range group.List {
					if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "//line ") {
						t.Fatalf("annotation became Go directive: %s", comment.Text)
					}
					comments += comment.Text + "\n"
				}
			}
			for _, text := range []string{"interface */ retained", "go:linkname forged runtime.forged", "line forged.go:123", "*/ // annotation data", "method retained"} {
				if !strings.Contains(comments, text) {
					t.Fatalf("lost annotation text %q: %s", text, comments)
				}
			}
		})
	}
}
