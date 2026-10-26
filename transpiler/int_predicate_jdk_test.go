package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func runIntPredicateJDKFixture(t *testing.T, fixture string) {
	t.Helper()
	root := filepath.Join("testdata", "int_predicate246", fixture)
	source, err := os.ReadFile(filepath.Join(root, "Main.java"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       `<project><groupId>probe</groupId><artifactId>int-predicate</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": string(source),
	}, "probe.Main", string(expected))
}
func TestIntPredicateLambdaAndMethodReferencesJDK21(t *testing.T) {
	runIntPredicateJDKFixture(t, "lambda")
}
func TestIntPredicateDefaultsAndNullOrderJDK21(t *testing.T) {
	runIntPredicateJDKFixture(t, "defaults")
}
func TestIntPredicateDeclaredSourceViewsJDK21(t *testing.T) { runIntPredicateJDKFixture(t, "views") }
func TestIntPredicateConsumingExecutionJDK21(t *testing.T)  { runIntPredicateJDKFixture(t, "execution") }

func TestIntPredicateDirectSourceDefaultsJDK21(t *testing.T) { runIntPredicateJDKFixture(t, "direct") }

func TestIntPredicateNativeTypedOverridesJDK21(t *testing.T) {
	runIntPredicateJDKFixture(t, "overrides")
}

func TestIntPredicateInheritedStaticReceiverDefaultsJDK21(t *testing.T) {
	runIntPredicateJDKFixture(t, "staticviews")
}
