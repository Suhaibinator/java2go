//go:build darwin || linux

package directoryoracle_test

import (
	"bytes"
	"os"
	"testing"
)

func TestFilesCreateDirectoriesChildOutputFrameworkTrailers(t *testing.T) {
	want, err := os.ReadFile("testdata/empty_path_removed_cwd_jdk21.stdout")
	if err != nil {
		t.Fatal(err)
	}
	join := func(body []byte, footer string) []byte {
		return append(append([]byte(nil), body...), []byte(footer)...)
	}
	changed := append([]byte(nil), want...)
	changed[0] ^= 1
	extra := append(append([]byte(nil), want...), []byte("PASS\n")...)
	vectors := []struct {
		name         string
		stdout, body []byte
		frame, match bool
	}{
		{"plain", join(want, "PASS\n"), want, true, true},
		{"covered-no-statements", join(want, "PASS\ncoverage: [no statements]\n"), want, true, true},
		{"changed-api-byte", join(changed, "PASS\ncoverage: [no statements]\n"), changed, true, false},
		{"extra-api-pass-line", join(extra, "PASS\ncoverage: [no statements]\n"), extra, true, false},
		{"no-pass", join(want, "coverage: [no statements]\n"), nil, false, false},
		{"failure", join(want, "FAIL\ncoverage: [no statements]\n"), nil, false, false},
		{"missing-newline", join(want, "PASS\ncoverage: [no statements]"), nil, false, false},
		{"different-coverage", join(want, "PASS\ncoverage: 100.0% of statements\n"), nil, false, false},
		{"coverage-spelling", join(want, "PASS\ncoverage: [no statements] \n"), nil, false, false},
		{"extra-trailing-line", join(want, "PASS\ncoverage: [no statements]\nextra\n"), nil, false, false},
		{"duplicated-footer", join(join(want, "PASS\ncoverage: [no statements]\n"), "PASS\ncoverage: [no statements]\n"), join(want, "PASS\ncoverage: [no statements]\n"), true, false},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			got, ok := directoryChildAPIObservations(v.stdout)
			if ok != v.frame {
				t.Fatalf("recognized=%t want%t", ok, v.frame)
			}
			if ok && !bytes.Equal(got, v.body) {
				t.Errorf("decoder changed API bytes: got%q want%q", got, v.body)
			}
			if match := ok && bytes.Equal(got, want); match != v.match {
				t.Errorf("full oracle match=%t want%t", match, v.match)
			}
		})
	}
}
