package transpiler

import (
 "os"
 "path/filepath"
 "testing"
)

func runCanonicalPrerequisiteJVM(t *testing.T, name string) {
 t.Helper()
 source, err := os.ReadFile(filepath.Join("testdata", "canonical_prerequisites", name+".java"))
 if err != nil { t.Fatal(err) }
 expected, err := os.ReadFile(filepath.Join("testdata", "canonical_prerequisites", name+".stdout"))
 if err != nil { t.Fatal(err) }
 runCampaignCompilerStrictProjectOracle(t, map[string]string{
  "pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>prereq</groupId><artifactId>canonical-boundary</artifactId><version>1</version></project>`,
  "src/main/java/prereq/"+name+".java": string(source),
 }, "prereq."+name, string(expected))
}

func TestCanonicalStringReplaceJVM(t *testing.T) { runCanonicalPrerequisiteJVM(t, "ReplaceProbe") }
func TestCanonicalMessageConstructorsJVM(t *testing.T) { runCanonicalPrerequisiteJVM(t, "ConstructorProbe") }
func TestCanonicalIntegerTextBoundaryJVM(t *testing.T) { runCanonicalPrerequisiteJVM(t, "IntegerProbe") }
