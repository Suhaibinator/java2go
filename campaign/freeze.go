package campaign

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
)

type expectedObservation struct {
	ExitCode int               `json:"exit_code"`
	Stdout   string            `json:"stdout"`
	Stderr   string            `json:"stderr"`
	Files    map[string]string `json:"files,omitempty"`
}

func fixtureHashes(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if d.Name() == "target" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture symlink prohibited: %s", path)
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		result[filepath.ToSlash(rel)] = hash(b)
		return nil
	})
	return result, err
}
func loadExpected(root string, m Manifest) (map[int]expectedObservation, error) {
	result := map[int]expectedObservation{}
	// oracle.json optionally supplies frozen exit and output-file hashes. Its
	// repeated observations must agree and its stdout must match text snapshots.
	var oracle struct {
		Runs []struct {
			Seed   int `json:"seed"`
			Repeat int `json:"repeat"`
			expectedObservation
		} `json:"runs"`
	}
	b, e := os.ReadFile(filepath.Join(root, "oracle.json"))
	if e == nil {
		if e = json.Unmarshal(b, &oracle); e != nil {
			return nil, e
		}
		for _, run := range oracle.Runs {
			if previous, ok := result[run.Seed]; ok && !reflect.DeepEqual(previous, run.expectedObservation) {
				return nil, fmt.Errorf("frozen oracle is nondeterministic for seed %d", run.Seed)
			}
			result[run.Seed] = run.expectedObservation
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	for _, seed := range m.Seeds {
		expected, present := result[seed]
		stdout, e := os.ReadFile(filepath.Join(root, fmt.Sprintf("expected.seed-%d.stdout", seed)))
		if e != nil {
			return nil, fmt.Errorf("frozen stdout snapshot required: %w", e)
		}
		if present && expected.Stdout != string(stdout) {
			return nil, fmt.Errorf("oracle.json and stdout snapshot disagree for seed %d", seed)
		}
		expected.Stdout = string(stdout)
		stderr, e := os.ReadFile(filepath.Join(root, fmt.Sprintf("expected.seed-%d.stderr", seed)))
		if e != nil && !os.IsNotExist(e) {
			return nil, e
		}
		if present && expected.Stderr != string(stderr) {
			return nil, fmt.Errorf("oracle.json and stderr snapshot disagree for seed %d", seed)
		}
		expected.Stderr = string(stderr)
		if len(m.OutputFiles) > 0 {
			if !present {
				return nil, fmt.Errorf("oracle.json with frozen output file hashes required for seed %d", seed)
			}
			for _, name := range m.OutputFiles {
				if len(expected.Files[name]) != 64 {
					return nil, fmt.Errorf("missing frozen output hash %s for seed %d", name, seed)
				}
			}
		}
		result[seed] = expected
	}
	return result, nil
}
func compareExpected(want expectedObservation, got Execution) string {
	if got.TimedOut || got.Error != "" {
		return "oracle execution did not complete"
	}
	if got.ExitCode != want.ExitCode {
		return "frozen exit code differs"
	}
	if got.Stdout != want.Stdout {
		return "frozen stdout differs"
	}
	if got.Stderr != want.Stderr {
		return "frozen stderr differs"
	}
	if want.Files != nil && !reflect.DeepEqual(want.Files, got.Files) {
		return "frozen output files differ"
	}
	return ""
}
