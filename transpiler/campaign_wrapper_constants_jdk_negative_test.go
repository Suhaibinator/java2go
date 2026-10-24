package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Synthetic invalid Java cases establish the language boundary with the actual
// pinned JDK. They are separate from the legal all-source strict projects.
func TestCampaignWrapperConstantsJDKRejections(t *testing.T) {
	cases := map[string]string{
		"ambiguous-import": `import static java.lang.Byte.BYTES;import static java.lang.Long.BYTES;class Invalid{int read(){return BYTES;}}
`,
		"boolean-size": `class Invalid{int read(){return Boolean.SIZE;}}
`,
		"int-not-byte": `class Invalid{int take(byte n){return n;}int read(){return take(java.lang.Long.BYTES);}}
`,
		"object-qualifier": `class Invalid{int read(Object value){return value.BYTES;}}
`,
		"unbounded-binder": `class Invalid<Long>{int read(Long value){return value.BYTES;}}
`,
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "Invalid.java")
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, javac, "--release", "21", "-d", filepath.Join(root, "classes"), path)
			cmd.Dir = root
			cmd.WaitDelay = 2 * time.Second
			var out, errout bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &errout
			err = cmd.Run()
			if ctx.Err() != nil {
				t.Fatalf("JDK rejection timed out: %v", ctx.Err())
			}
			if _, ok := err.(*exec.ExitError); !ok || errout.Len() == 0 {
				t.Fatalf("expected actual javac rejection: err=%v stdout=%q stderr=%q", err, out.String(), errout.String())
			}
			t.Logf("actual JDK rejection %s stdout=%q stderr=%q", name, out.String(), errout.String())
			if artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR"); artifactDir != "" {
				for leaf, raw := range map[string][]byte{"Invalid.java": []byte(source), "jdk.stdout": out.Bytes(), "jdk.stderr": errout.Bytes()} {
					if err := writeProjectFile(filepath.Join(artifactDir, t.Name(), leaf), raw); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}
