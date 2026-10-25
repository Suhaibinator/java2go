package transpiler

import (
	"os"
	"strings"
	"testing"
)

func TestGenericFamilyEnumConstructorLocalJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_enum_constructor_local/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>enum-constructor</artifactId><version>1</version></project>`,
		"src/main/java/genericenumconstructor/Main.java": string(source),
	}, "genericenumconstructor.Main", "s|RED|true|BLUE\n")
}

func TestGenericFamilyEnumConstructorLateCastJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_enum_constructor_late_cast/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>enum-constructor-casts</artifactId><version>1</version></project>`,
		"src/main/java/genericenumconstructorvariation/Main.java": string(source),
	}, "genericenumconstructorvariation.Main", "s|true|true|true|true\n")
}

func TestGenericFamilyEnumConstructorVarLocalJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_enum_constructor_var/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>enum-constructor-var</artifactId><version>1</version></project>`,
		"src/main/java/genericenumconstructorvar/Main.java": string(source),
	}, "genericenumconstructorvar.Main", "s|true|true|true|true\n")
}

// All variants retain the exact legal-JDK red fixture and its observations.
// The multiple-declarator case additionally observes left-to-right binding.
func genericFamilyConstructorLoopSources(t *testing.T) []struct{ name, source string } {
	t.Helper()
	data, err := os.ReadFile("testdata/generic_enum_constructor_loop/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	original := string(data)
	header := "for (E local = supplied; again; again = false)"
	if strings.Count(original, header) != 1 {
		t.Fatal("missing frozen loop header")
	}
	cases := []struct{ name, source string }{{"declared", original}, {"inferred", strings.Replace(original, header, "for (var local = supplied; again; again = false)", 1)}, {"uninitialized", strings.Replace(original, header, "for (E local; again; again = false)", 1)}}
	multiple := strings.Replace(original, header, "for (E first = supplied, local = first; again; again = false)", 1)
	multiple = strings.Replace(multiple, "      local = (E) replacement;", "      if (first != supplied || local != first) throw new AssertionError();\n      local = (E) replacement;", 1)
	return append(cases, struct{ name, source string }{"multiple", multiple})
}
func TestGenericFamilyEnumConstructorForLocalsJDK21(t *testing.T) {
	for _, variant := range genericFamilyConstructorLoopSources(t) {
		t.Run(variant.name, func(t *testing.T) {
			runCampaignCompilerStrictProjectOracle(t, map[string]string{
				"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>enum-constructor-loop</artifactId><version>1</version></project>`,
				"src/main/java/genericenumconstructorloop/Main.java": variant.source,
			}, "genericenumconstructorloop.Main", "s|true|true|true|true\n")
		})
	}
}
