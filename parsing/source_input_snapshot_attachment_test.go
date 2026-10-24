// This portable TDD probe compiles against the original public API.
// Reflection is confined to the attachment probe; direct symbol-package
// contract tests cover the snapshot internals.
package parsing_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func proposedSnapshot(t *testing.T, file *symbol.FileScope) reflect.Value {
	t.Helper()
	getter := reflect.ValueOf(file).MethodByName("SourceInputSnapshot")
	if !getter.IsValid() {
		t.Fatal("ParseSymbols returned no source-input snapshot attachment API")
	}
	results := getter.Call(nil)
	if len(results) != 1 || results[0].Kind() != reflect.Pointer || results[0].IsNil() {
		t.Fatal("ParseSymbols did not attach an owner-bound source-input snapshot")
	}
	return results[0]
}

func proposedSnapshotError(t *testing.T, snapshot reflect.Value) error {
	t.Helper()
	validate := snapshot.MethodByName("Validate")
	if !validate.IsValid() {
		t.Fatal("snapshot has no explicit current-input validation API")
	}
	result := validate.Call(nil)
	if len(result) != 1 {
		t.Fatal("snapshot Validate must return one error")
	}
	if result[0].IsNil() {
		return nil
	}
	err, ok := result[0].Interface().(error)
	if !ok {
		t.Fatal("snapshot Validate result does not implement error")
	}
	return err
}

func TestSourceInputSnapshotParseAttachment(t *testing.T) {
	for _, source := range []string{
		"class Original { int member; }",
		"package owned;",                         // Classless input still owns its actual parse root.
		"class \\uu0043anonical { int member; }", // Existing Unicode path, unchanged.
	} {
		t.Run(fmt.Sprintf("bytes=%d", len(source)), func(t *testing.T) {
			parsed := parsing.SourceFile{Name: "Original.java", Source: []byte(source)}
			if err := parsed.ParseAST(); err != nil {
				t.Fatal(err)
			}
			file := parsed.ParseSymbols()
			originalRoot := file.ParsedRoot()
			if originalRoot == nil || originalRoot != parsed.Ast {
				t.Fatal("ParseSymbols changed the original parse anchor")
			}
			snapshot := proposedSnapshot(t, file)
			if err := proposedSnapshotError(t, snapshot); err != nil {
				t.Fatal("fresh actual ParseSymbols input did not validate", err)
			}
			previous := file.Source[0]
			file.Source[0] ^= 1
			if proposedSnapshotError(t, snapshot) == nil {
				t.Fatal("retained snapshot accepted live parser-source mutation")
			}
			file.Source[0] = previous
			if err := proposedSnapshotError(t, snapshot); err != nil {
				t.Fatal("read-only current-input validation must accept exact restoration", err)
			}
			if file.ParsedRoot() != originalRoot {
				t.Fatal("validation changed actual parsed-root identity")
			}
		})
	}
}
