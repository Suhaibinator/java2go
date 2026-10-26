package transpiler

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// Every source-package dependency must survive even when its only Java use is
// a class literal. The packages deliberately have no constructors or static
// member uses in Main, so unrelated selectors cannot mask a missing dependency.
func TestClassLiteralSourcePackageDemand(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"literal/plain/Only.java":      `package literal.plain; public class Only { static { System.out.println("unexpected initialization"); } }`,
		"literal/generic/Box.java":     `package literal.generic; public class Box<T> {}`,
		"literal/face/Face.java":       `package literal.face; public interface Face {}`,
		"literal/nested/Outer.java":    `package literal.nested; public class Outer { public static class Inner {} }`,
		"literal/array/Element.java":   `package literal.array; public class Element {}`,
		"literal/record/Point.java":    `package literal.record; public record Point(int x) {}`,
		"literal/annotation/Flag.java": `package literal.annotation; public @interface Flag {}`,
		"literal/choice/Choice.java":   `package literal.choice; public enum Choice { YES }`,
		"literal/app/Main.java": `package literal.app;
import literal.plain.Only;
import literal.generic.Box;
import literal.face.Face;
import literal.nested.Outer.Inner;
public class Main {
 static Class<?> plain() { return Only.class; }
 static Class<?> generic() { return Box.class; }
 static Class<?> face() { return Face.class; }
 static Class<?> nested() { return Inner.class; }
 static Class<?> array() { return literal.array.Element[][].class; }
 static Class<?> record() { return literal.record.Point.class; }
 static Class<?> annotation() { return literal.annotation.Flag.class; }
 static Class<?> choice() { return literal.choice.Choice.class; }
 static Class<?> primitive() { return int.class; }
 static Class<?> jdk() { return java.lang.String.class; }
}`,
	}
	for name, source := range files {
		if err := writeProjectFile(filepath.Join(root, name), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	generated := convertJavaProjectDir(t, root)["literal/app/Main.go"]
	parsed, err := parser.ParseFile(token.NewFileSet(), "Main.go", generated, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	imports := make(map[string]bool)
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		imports[path] = true
	}
	for _, owner := range []string{"literal/plain", "literal/generic", "literal/face", "literal/nested", "literal/array", "literal/record", "literal/annotation", "literal/choice"} {
		if !imports[owner] {
			t.Errorf("class-literal-only source package %s absent from generated import closure", owner)
		}
	}
	for _, invalid := range []string{"literal/nested/Outer", "java/lang"} {
		if imports[invalid] {
			t.Errorf("class literal demanded non-package %s", invalid)
		}
	}
}
