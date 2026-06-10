package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGoTestWithStdjava compiles and runs generated code that imports the stdjava
// runtime package. Unlike runGoTestInTempModule it wires a replace directive so
// the temp module resolves stdjava against this repository's copy, allowing the
// intrinsic rewrites (stdjava.StringCharAt, stdjava.NewStringBuilder, ...) to be
// behavior-verified.
func runGoTestWithStdjava(t *testing.T, generatedGo string, goTestSource string) {
	t.Helper()

	repoRoot := repoRootDir(t)
	tempDir := t.TempDir()

	goMod := "module generated\n\ngo 1.25.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed writing temp go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated.go"), []byte(generatedGo), 0644); err != nil {
		t.Fatalf("failed writing generated source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated_behavior_test.go"), []byte(goTestSource), 0644); err != nil {
		t.Fatalf("failed writing generated behavior test: %v", err)
	}

	// Resolve the stdjava dependency's own requirements (golang.org/x/exp) into
	// the temp module before building.
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tempDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s", string(out))
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code behavior verification failed:\n%s", string(out))
	}
}

// repoRootDir returns the absolute path to this repository's root by walking up
// from the test working directory until a go.mod for the java2go module is found.
func repoRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modPath); err == nil {
			if strings.Contains(string(data), "module github.com/NickyBoy89/java2go") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate java2go module root")
		}
		dir = parent
	}
}

func TestRuntime_StringIntrinsics(t *testing.T) {
	src := `
public class StringRuntime {
    public static String upper(String s) {
        return s.toUpperCase();
    }
    public static int firstIndex(String s, String needle) {
        return s.indexOf(needle);
    }
    public static String sub(String s) {
        return s.substring(1, 4);
    }
    public static boolean blank(String s) {
        return s.isBlank();
    }
    public static char at(String s, int i) {
        return s.charAt(i);
    }
    public static int len(String s) {
        return s.length();
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGoTestWithStdjava(t, out, `
package main

import "testing"

func TestStringIntrinsicsBehavior(t *testing.T) {
	if got := Upper("héllo"); got != "HÉLLO" {
		t.Fatalf("Upper = %q", got)
	}
	if got := FirstIndex("abcXdef", "X"); got != 3 {
		t.Fatalf("FirstIndex = %d, want 3", got)
	}
	if got := Sub("abcdef"); got != "bcd" {
		t.Fatalf("Sub = %q, want bcd", got)
	}
	if got := Blank("   "); !got {
		t.Fatalf("Blank(spaces) = false, want true")
	}
	if got := At("héllo", 1); got != 'é' {
		t.Fatalf("At = %q, want é", got)
	}
	if got := Len("héllo"); got != 5 {
		t.Fatalf("Len = %d, want 5", got)
	}
}
`)
}

func TestRuntime_StringBuilder(t *testing.T) {
	src := `
public class SBRuntime {
    public static String build() {
        StringBuilder sb = new StringBuilder();
        sb.append("ab");
        sb.append(1);
        sb.insert(0, "Z");
        sb.reverse();
        return sb.toString();
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGoTestWithStdjava(t, out, `
package main

import "testing"

func TestStringBuilderBehavior(t *testing.T) {
	// "ab" + "1" = "ab1", insert "Z" at 0 -> "Zab1", reverse -> "1baZ"
	if got := Build(); got != "1baZ" {
		t.Fatalf("Build = %q, want 1baZ", got)
	}
}
`)
}

func TestRuntime_MathAndBoxed(t *testing.T) {
	src := `
public class MathRuntime {
    public static int absVal(int x) {
        return Math.abs(x);
    }
    public static int maxVal(int a, int b) {
        return Math.max(a, b);
    }
    public static long roundVal(double d) {
        return Math.round(d);
    }
    public static int parse(String s) {
        return Integer.parseInt(s);
    }
    public static boolean digit(char c) {
        return Character.isDigit(c);
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGoTestWithStdjava(t, out, `
package main

import "testing"

func TestMathAndBoxedBehavior(t *testing.T) {
	if got := AbsVal(-7); got != 7 {
		t.Fatalf("AbsVal = %d, want 7", got)
	}
	if got := MaxVal(3, 9); got != 9 {
		t.Fatalf("MaxVal = %d, want 9", got)
	}
	if got := RoundVal(2.5); got != 3 {
		t.Fatalf("RoundVal = %d, want 3", got)
	}
	if got := Parse("42"); got != 42 {
		t.Fatalf("Parse = %d, want 42", got)
	}
	if got := Digit('5'); !got {
		t.Fatalf("Digit('5') = false, want true")
	}
}
`)
}

func TestRuntime_Collections(t *testing.T) {
	src := `
import java.util.List;
import java.util.ArrayList;
import java.util.Map;
import java.util.HashMap;
public class CollRuntime {
    public static String listJoin() {
        List<String> xs = new ArrayList<String>();
        xs.add("a");
        xs.add("b");
        xs.add("c");
        String out = "";
        for (String s : xs) {
            out = out + s;
        }
        return out + ":" + xs.size() + ":" + xs.get(1);
    }
    public static int mapLookup() {
        Map<String, Integer> m = new HashMap<String, Integer>();
        m.put("one", 1);
        m.put("two", 2);
        return m.get("one") + m.get("two") + m.size();
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGoTestWithStdjava(t, out, `
package main

import "testing"

func TestCollectionsBehavior(t *testing.T) {
	if got := ListJoin(); got != "abc:3:b" {
		t.Fatalf("ListJoin = %q, want abc:3:b", got)
	}
	if got := MapLookup(); got != 5 {
		t.Fatalf("MapLookup = %d, want 5", got)
	}
}
`)
}

func TestRuntime_OptionalMapAndConcat(t *testing.T) {
	src := `
import java.util.Optional;
public class OptRuntime {
    public static int doubled(int x) {
        Optional<Integer> num = Optional.of(x);
        return num.map(n -> n * 2).get();
    }
    public static String orElse(boolean present) {
        Optional<String> o = present ? Optional.of("yes") : Optional.<String>empty();
        return o.orElse("no");
    }
    public static int concatLen() {
        var g = "ab" + "cd";
        return g.length();
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGoTestWithStdjava(t, out, `
package main

import "testing"

func TestOptionalConcatBehavior(t *testing.T) {
	if got := Doubled(21); got != 42 {
		t.Fatalf("Doubled(21) = %d, want 42", got)
	}
	if got := ConcatLen(); got != 4 {
		t.Fatalf("ConcatLen() = %d, want 4", got)
	}
}
`)
}
