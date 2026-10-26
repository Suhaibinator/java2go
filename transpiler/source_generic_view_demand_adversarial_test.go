package transpiler

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceGenericViewDemandAdversarialProjectJVM(t *testing.T) {
	for _, name := range []string{"shared_mutation", "bridge_dispatch", "shadow_guards"} {
		t.Run(name, func(t *testing.T) {
			directory := filepath.Join("testdata", "source_generic_view_demand_adversarial", name)
			var oracle struct {
				Observations []struct {
					Name   string
					Args   []string
					Stdout string
					Stderr string
				}
			}
			data, err := os.ReadFile(filepath.Join(directory, "oracle.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &oracle); err != nil {
				t.Fatal(err)
			}
			observations := make([]campaignCompilerProjectObservation, len(oracle.Observations))
			for i, o := range oracle.Observations {
				observations[i] = campaignCompilerProjectObservation{name: o.Name, args: o.Args, stdout: o.Stdout, stderr: o.Stderr}
			}
			files := make(map[string]string)
			err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || filepath.Base(path) == "oracle.json" {
					return nil
				}
				relative, err := filepath.Rel(directory, path)
				if err != nil {
					return err
				}
				contents, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				files[filepath.ToSlash(relative)] = string(contents)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			runCampaignCompilerStrictProjectObservations(t, files, "app.Main", observations)
		})
	}
}

func TestSourceGenericViewDemandCompletePlanGuards(t *testing.T) {
	helper := setupParseHelper(t, `class Cell<T>{T value;Cell(T value){this.value=value;}T read(){return value;}}
 class BadCtor<T>{T value;BadCtor(T value){this.value=value;T copy=this.value;}}
 class BadStorage<T>{java.util.List<T> values;}
 class BadBound<T extends java.util.List<String>>{T value;}
 class Use{Cell<?> positive(Object value){return (Cell<?>)value;}BadCtor<?> ctor(Object value){return (BadCtor<?>)value;}BadStorage<?> storage(Object value){return (BadStorage<?>)value;}BadBound<?> bound(Object value){return (BadBound<?>)value;}}`)
	cell := helper.File.Symbols.FindClassScope("Cell")
	if canonicalGenericFamily(cell, helper.Ctx) == nil {
		t.Fatal("resolved source-generic wildcard cast demand did not admit complete Cell plan")
	}
	for _, name := range []string{"BadCtor", "BadStorage", "BadBound"} {
		scope := helper.File.Symbols.FindClassScope(name)
		if plan, err := planGenericFamily(scope, helper.Ctx); err == nil || plan != nil {
			t.Fatalf("unsupported %s plan admitted: %v", name, err)
		}
		if canonicalGenericFamily(scope, helper.Ctx) != nil {
			t.Fatalf("lexical demand bypassed complete %s plan guard", name)
		}
	}
}
