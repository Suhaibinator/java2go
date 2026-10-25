//go:build darwin || linux

package directoryoracle_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf16"

	j "github.com/NickyBoy89/java2go/stdjava"
)

// These observations are from three fresh JDK21 processes on the default
// filesystem. Setup and cleanup use only a fresh owned test root, without
// following symbolic links. The operation under test is the existing API.
type directoryObservation struct {
	Name, Before, After, Exception, MessageUTF16 string
	CauseNull                                    bool
}

func directoryObservations(t *testing.T) []directoryObservation {
	t.Helper()
	data, err := os.ReadFile("testdata/directory_jdk21_observations.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []directoryObservation
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 13 {
		t.Fatalf("observation count=%d, want13", len(rows))
	}
	return rows
}
func directorySnapshot(t *testing.T, root string) string {
	t.Helper()
	var entries []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries = append(entries, rel+":L="+filepath.ToSlash(target))
		case entry.IsDir():
			entries = append(entries, rel+":D")
		default:
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entries = append(entries, rel+":F="+hex.EncodeToString(data))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	return strings.Join(entries, "|")
}
func directorySetup(t *testing.T, name, root string) *j.JavaPath {
	t.Helper()
	must := func(err error) {
		if err != nil {
			t.Fatalf("setup %s: %v", name, err)
		}
	}
	path := filepath.Join(root, "leaf")
	switch name {
	case "fresh-leaf":
	case "fresh-nested":
		path = filepath.Join(root, "a", "b", "leaf")
	case "existing-nested":
		path = filepath.Join(root, "a", "b")
		must(os.MkdirAll(path, 0700))
	case "existing-file":
		must(os.WriteFile(path, []byte{65, 0, 66}, 0600))
	case "file-parent":
		must(os.WriteFile(filepath.Join(root, "block"), []byte{70}, 0600))
		path = filepath.Join(root, "block", "child", "leaf")
	case "link-dir-leaf", "link-dir-parent":
		must(os.Mkdir(filepath.Join(root, "real"), 0700))
		must(os.Symlink("real", filepath.Join(root, "link")))
		path = filepath.Join(root, "link")
		if name == "link-dir-parent" {
			path = filepath.Join(path, "child")
		}
	case "link-file":
		must(os.WriteFile(filepath.Join(root, "real"), []byte{76}, 0600))
		must(os.Symlink("real", filepath.Join(root, "link")))
		path = filepath.Join(root, "link")
	case "dangling-leaf", "dangling-parent":
		must(os.Symlink("missing", filepath.Join(root, "link")))
		path = filepath.Join(root, "link")
		if name == "dangling-parent" {
			path = filepath.Join(path, "child")
		}
	case "link-loop-parent":
		must(os.Symlink("b", filepath.Join(root, "a")))
		must(os.Symlink("a", filepath.Join(root, "b")))
		path = filepath.Join(root, "a", "child")
	case "relative-dot":
		must(os.Mkdir(filepath.Join(root, "inside"), 0700))
		cwd, err := os.Getwd()
		must(err)
		rel, err := filepath.Rel(cwd, root)
		must(err)
		path = rel + string(filepath.Separator) + "inside/../dot-leaf"
	case "null-path":
		return nil
	default:
		t.Fatalf("unknown observation %q", name)
	}
	return j.NewJavaPath(path)
}
func directoryInvoke(path *j.JavaPath) (result *j.JavaPath, class string) {
	defer func() {
		if value := recover(); value != nil {
			normalized := j.NormalizePanic(value)
			if throwable, ok := normalized.(j.Throwable); ok {
				class = throwable.ThrowableTypeName()
			} else {
				class = fmt.Sprintf("nonThrowable:%T", normalized)
			}
		}
	}()
	result = j.FilesCreateDirectories(path)
	return
}
func directoryMessageUnits(message string) string {
	if j.StringIsNull(message) {
		return "<null>"
	}
	var output strings.Builder
	for _, unit := range utf16.Encode([]rune(message)) {
		fmt.Fprintf(&output, "%04x", unit)
	}
	return output.String()
}
func directoryInvokeDetailed(path *j.JavaPath, base string) (result *j.JavaPath, class, message, canonicalMessage string, causeNull bool) {
	defer func() {
		if value := recover(); value != nil {
			normalized := j.NormalizePanic(value)
			throwable, ok := normalized.(j.Throwable)
			if !ok {
				class = fmt.Sprintf("nonThrowable:%T", normalized)
				return
			}
			class = throwable.ThrowableTypeName()
			message = directoryMessageUnits(strings.ReplaceAll(j.GetMessage(normalized), base, "<ROOT>"))
			canonical := j.JavaThrowableMessageExecution(nil, normalized)
			if canonical == nil {
				canonicalMessage = "<null>"
			} else {
				var output strings.Builder
				// Canonical message is inspected as exact UTF16, without UTF8 roundtripping.
				for _, unit := range canonical.UTF16Copy() {
					fmt.Fprintf(&output, "%04x", unit)
				}
				canonicalMessage = strings.ReplaceAll(output.String(), directoryMessageUnits(base), directoryMessageUnits("<ROOT>"))
			}
			causeNull = j.GetCause(normalized) == nil
		}
	}()
	result = j.FilesCreateDirectories(path)
	return
}
func directoryContractCases(t *testing.T, wantFailure bool) {
	for _, row := range directoryObservations(t) {
		if (row.Exception != "") != wantFailure {
			continue
		}
		t.Run(row.Name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, row.Name)
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			path := directorySetup(t, row.Name, root)
			if got := directorySnapshot(t, root); got != row.Before {
				t.Fatalf("setup snapshot=%q want%q", got, row.Before)
			}
			result, class, message, canonicalMessage, causeNull := directoryInvokeDetailed(path, base)
			t.Logf("actual identity=%t exception=%q", result == path, class)
			if class != row.Exception {
				t.Errorf("exception=%q, JDK21=%q", class, row.Exception)
			}
			if row.Exception != "" {
				if message != row.MessageUTF16 {
					t.Errorf("messageUTF16=%q, JDK21=%q", message, row.MessageUTF16)
				}
				if canonicalMessage != row.MessageUTF16 {
					t.Errorf("canonicalMessageUTF16=%q, JDK21=%q", canonicalMessage, row.MessageUTF16)
				}
				if causeNull != row.CauseNull {
					t.Errorf("causeNull=%t, JDK21=%t", causeNull, row.CauseNull)
				}
			}
			if row.Exception == "" {
				if result != path {
					t.Errorf("returned distinct JavaPath, JDK21 returns identical reference")
				}
				if result == nil || result.ToString() != path.ToString() {
					t.Errorf("returned path value changed")
				}
			}
			if got := directorySnapshot(t, root); got != row.After {
				t.Errorf("after=%q, JDK21=%q", got, row.After)
			}
			if err := os.RemoveAll(root); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(root); !os.IsNotExist(err) {
				t.Errorf("scoped cleanup err=%v", err)
			}
		})
	}
}
func TestFilesCreateDirectoriesReturnIdentityAndLifecycle(t *testing.T) {
	directoryContractCases(t, false)
}
func TestFilesCreateDirectoriesFailureAndSymlinkLifecycle(t *testing.T) {
	directoryContractCases(t, true)
}
func TestFilesCreateDirectoriesCleanupAndRepeatedCall(t *testing.T) {
	root := t.TempDir()
	path := j.NewJavaPath(filepath.Join(root, "repeat", "leaf"))
	for i := 0; i < 2; i++ {
		result, class := directoryInvoke(path)
		if class != "" {
			t.Fatalf("call%d exception=%s", i, class)
		}
		if result != path {
			t.Errorf("call%d returned distinct reference", i)
		}
	}
	sameContent := j.NewJavaPath(path.ToString())
	if sameContent == path {
		t.Fatal("invalid independent-path setup")
	}
	result, class := directoryInvoke(sameContent)
	if class != "" || result != sameContent {
		t.Errorf("independent same-content path result identity=%t class=%q", result == sameContent, class)
	}
	if err := os.RemoveAll(filepath.Join(root, "repeat")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cleanup entries=%d err=%v", len(entries), err)
	}
}
