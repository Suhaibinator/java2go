package transpiler

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Frozen bytes were captured by an independent author using the installed
// JDK21, three fresh processes per entry. Author oracle SHA256:
// 34134790f396e4dd2aa874b62bfb0be42fe3204c0695f08a17fa89c30e3a8e1f.
// The strict oracle helper also runs unchanged Java afresh before generated Go.
func factoryActualStdout(t *testing.T, entry string) string {
	t.Helper()
	stdoutSHA := map[string]string{
		"FactoryApp":   "4a6689419b00b11700c9b6246bcfa8936c8f5e1e824db3a7e57030e2d1c1a684",
		"FactoryProbe": "dae1a014e98c4475908e60fb4fb37ccc7b96377992da24b96d6065c244eb55bc",
	}[entry]
	if stdoutSHA == "" {
		t.Fatal("unknown independently observed entry")
	}
	pins := map[string]string{
		"stdout":     stdoutSHA,
		"stderr":     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"status":     "9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa",
		"files.json": "ca3d163bab055381827226140568f3bef7eaac187cebd76878e0b63e9e442356",
	}
	var stdout []byte
	for suffix, pin := range pins {
		data, err := os.ReadFile(filepath.Join("testdata", "complex_factory_original_jvm", entry+"."+suffix))
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != pin {
			t.Fatalf("frozen author %s.%s drifted: %s", entry, suffix, got)
		}
		if suffix == "stdout" {
			stdout = data
		}
	}
	return string(stdout)
}

// The original classes remain unchanged. This additional program makes custom
// Factory callback laziness, call count, returned Box identity and String
// reference identity observable without a native Supplier or timing dependency.
const factoryOriginalClassesProbe = `package complex.factory;

public class FactoryProbe {
    static int calls;
    static String seen;
    static Box<String> token;

    static Factory<Box<String>> make() {
        return seed -> {
            calls++;
            seen = seed;
            return token;
        };
    }

    public static void main(String[] args) {
        token = new Box<>(new String("reference"));
        Factory<Box<String>> factory = make();
        int before = calls;
        Box<String> first = FactoryApp.build(factory);
        Box<String> second = FactoryApp.build(factory);
        System.out.println(before + ":" + calls + ":" + (first == token) + ":" +
            (second == token) + ":" + (first.get() == token.get()) + ":" + (seen == "seed"));
    }
}
`

// javac requires each public top-level declaration in its own named file. Keep
// every original type body byte-identical, duplicating only the package prefix.
// Reassembly proves that no declaration byte was silently dropped or rewritten.
func factoryOriginalJavaFiles(t *testing.T) map[string]string {
	t.Helper()
	source := factorySourceJava(t)
	const originalSHA256 = "6799a960f32ae318fabe767af3971fcaa79591c983967f2e8148fb4375403790"
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(source))); got != originalSHA256 {
		t.Fatalf("original Java changed: SHA256 %s", got)
	}
	headers := []string{"public interface Factory<T>", "public class Box<T>", "public class FactoryApp"}
	names := []string{"Factory", "Box", "FactoryApp"}
	starts := make([]int, len(headers)+1)
	for i, header := range headers {
		if strings.Count(source, header) != 1 {
			t.Fatalf("original public declaration %s missing or ambiguous", header)
		}
		starts[i] = strings.Index(source, header)
		if i > 0 && starts[i] <= starts[i-1] {
			t.Fatal("original declaration order changed")
		}
	}
	starts[len(headers)] = len(source)
	prefix := source[:starts[0]]
	if strings.TrimSpace(prefix) != "package complex.factory;" {
		t.Fatal("original package prefix changed")
	}
	pom, err := os.ReadFile(filepath.Join("testdata", "complex_factory_original_jvm", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(pom)); got != "87e5fb3469fe29c9878bab66ff108ec3a06b29cc6b14c7fbfcc8cd6272d32b80" {
		t.Fatalf("owned factory POM drifted: %s", got)
	}
	files := map[string]string{"pom.xml": string(pom)}
	reassembled := prefix
	for i, name := range names {
		declaration := source[starts[i]:starts[i+1]]
		files["src/main/java/complex/factory/"+name+".java"] = prefix + declaration
		reassembled += declaration
	}
	if reassembled != source {
		t.Fatal("public declaration split changed original Java bytes")
	}
	return files
}

func TestComplexFactoryOriginalJDK21StrictParity(t *testing.T) {
	owner := &campaignCompilerParentCLI{directory: t.TempDir()}
	t.Run("OriginalProgram", func(t *testing.T) {
		runCampaignCompilerStrictProjectObservationsWithExecutionLimit(t, factoryOriginalJavaFiles(t), "complex.factory.FactoryApp",
			[]campaignCompilerProjectObservation{{name: "original", stdout: factoryActualStdout(t, "FactoryApp")}}, 55*time.Second, owner)
	})
	t.Run("CustomFactoryIdentityAndLaziness", func(t *testing.T) {
		files := factoryOriginalJavaFiles(t)
		files["src/main/java/complex/factory/FactoryProbe.java"] = factoryOriginalClassesProbe
		runCampaignCompilerStrictProjectObservationsWithExecutionLimit(t, files, "complex.factory.FactoryProbe",
			[]campaignCompilerProjectObservation{{name: "probe", stdout: factoryActualStdout(t, "FactoryProbe")}}, 55*time.Second, owner)
	})
}
