//go:build darwin || linux

package directoryoracle_test

import (
	"encoding/json"
	j "github.com/NickyBoy89/java2go/stdjava"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesCreateDirectoriesFallbackRelativizeMatrix(t *testing.T) {
	var observations []struct {
		Name, Before, After, Exception, MessageUTF16 string
		CauseNull, ReturnDirectory                   bool
	}
	data, err := os.ReadFile("testdata/directory_jdk21_fallback_observations.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &observations); err != nil {
		t.Fatal(err)
	}
	if len(observations) != 3 {
		t.Fatalf("JDK observation count=%d", len(observations))
	}
	for _, row := range observations {
		t.Run(row.Name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, row.Name)
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			suffix := "a/../leaf"
			switch row.Name {
			case "missing-intermediate-dotdot":
			case "fallback-relative-parent":
				if err := os.Mkdir(filepath.Join(root, "inside"), 0700); err != nil {
					t.Fatal(err)
				}
				suffix = "inside/a/../../leaf"
			case "fallback-normalized-file":
				if err := os.WriteFile(filepath.Join(root, "block"), []byte{90}, 0600); err != nil {
					t.Fatal(err)
				}
				suffix = "a/../block/child"
			default:
				t.Fatalf("unknown observation %q", row.Name)
			}
			if got := directorySnapshot(t, root); got != row.Before {
				t.Fatalf("setup snapshot=%q, JDK21=%q", got, row.Before)
			}
			path := j.NewJavaPath(root + "/" + suffix)
			result, class, message, canonical, causeNull := directoryInvokeDetailed(path, base)
			if class != row.Exception {
				t.Errorf("exception=%q, JDK21=%q", class, row.Exception)
			}
			if row.Exception == "" {
				if result != path {
					t.Error("result identity differs from JDK21")
				}
				if got := j.FilesIsDirectory(result); got != row.ReturnDirectory {
					t.Errorf("directory=%t, JDK21=%t", got, row.ReturnDirectory)
				}
			} else {
				if message != row.MessageUTF16 {
					t.Errorf("messageUTF16=%q, JDK21=%q", message, row.MessageUTF16)
				}
				if canonical != row.MessageUTF16 {
					t.Errorf("canonicalMessageUTF16=%q, JDK21=%q", canonical, row.MessageUTF16)
				}
				if causeNull != row.CauseNull {
					t.Errorf("causeNull=%t, JDK21=%t", causeNull, row.CauseNull)
				}
			}
			if got := directorySnapshot(t, root); got != row.After {
				t.Errorf("after=%q, JDK21=%q", got, row.After)
			}
			if err := os.RemoveAll(root); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(root); !os.IsNotExist(err) {
				t.Errorf("cleanup=%v", err)
			}
		})
	}
}
