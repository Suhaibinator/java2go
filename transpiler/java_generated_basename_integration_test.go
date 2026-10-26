package transpiler

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestJavaGeneratedBasenamesPreserveDistinctFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	for _, branch := range []string{"package", "relative", "fallback"} {
		t.Run(branch, func(t *testing.T) {
			names := []string{"$A", "A$B", "_Hidden", ".Hidden", "java2goSource_2441", "Java2goSource_2441"}
			emitted := map[string]string{}
			for _, stem := range names {
				name := filepath.Join(root, "nested", stem+".java")
				original := []byte("class OriginalIdentity {}")
				file := parsing.SourceFile{Name: name, Source: original}
				var inputRoots []string
				if branch == "package" {
					file.Symbols = &symbol.FileScope{Package: "p.nested"}
				}
				if branch == "relative" {
					inputRoots = []string{root}
				}
				output := outputRelativePath(file, inputRoots, "p")
				basename := filepath.Base(output)
				if strings.Contains(basename, "$") || strings.HasPrefix(basename, "_") || strings.HasPrefix(basename, ".") || !strings.HasSuffix(basename, ".go") {
					t.Fatalf("Java unit %q became an unsafe or ignored Go filename %q", stem, output)
				}
				if previous, exists := emitted[strings.ToLower(output)]; exists {
					t.Fatalf("distinct Java units %q and %q collide at %q", previous, stem, output)
				}
				emitted[strings.ToLower(output)] = stem
				expectedDirectory := filepath.Join(strings.TrimPrefix(filepath.Dir(name), string(filepath.Separator)))
				if branch != "fallback" {
					expectedDirectory = "nested"
				}
				if filepath.Dir(output) != expectedDirectory {
					t.Fatalf("directory changed: %q, want %q", filepath.Dir(output), expectedDirectory)
				}
				if file.Name != name || string(file.Source) != string(original) || file.Symbols != nil && file.Symbols.Package != "p.nested" {
					t.Fatal("Java identity changed during filename emission")
				}
			}
		})
	}
}

func TestJavaGeneratedBasenameNamespaceProjectJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                 `<project><modelVersion>4.0.0</modelVersion><groupId>dollar</groupId><artifactId>filenames</artifactId><version>1</version></project>`,
		"src/main/java/p/$A.java": `package p; public class $A {public static int read(){return 3;}}`,
		"src/main/java/p/java2goSource_2441.java": `package p; public class java2goSource_2441 {public static int read(){return 11;}}`,
		"src/main/java/p/_Hidden.java":            `package p; class _Hidden {static int read(){return 17;}}`,
		"src/main/java/p/.Hidden.java":            `package p; class DotHidden {static int read(){return 19;}}`,
		"src/main/java/p/Main.java": `package p; public class Main {public static void main(String[] args){
 System.out.println($A.read()+":"+java2goSource_2441.read()+":"+_Hidden.read()+":"+DotHidden.read()+":"+
  $A.class.getSimpleName()+":"+java2goSource_2441.class.getSimpleName()+":"+_Hidden.class.getSimpleName());
}}`,
	}, "p.Main", "3:11:17:19:$A:java2goSource_2441:_Hidden\n")
}
