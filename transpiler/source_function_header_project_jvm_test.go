package transpiler

import (
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)

// Independent JDK21 source and six stateful observations are copied unchanged.
// The shared helper passes identical arguments to the fresh JVM and generated Go.
func TestSourceFunctionHeaderShadowProjectJVM(t *testing.T) {
 root := filepath.Join("testdata", "function_header_shadow")
 files := map[string]string{"pom.xml": `<project><groupId>review</groupId><artifactId>function-header-shadow</artifactId><version>1</version></project>`}
 for _, name := range []string{"src/main/java/app/Length.java", "src/main/java/app/Main.java"} {
  content, err := os.ReadFile(filepath.Join(root, name)); if err != nil { t.Fatal(err) }; files[name] = string(content)
 }
 raw, err := os.ReadFile(filepath.Join(root, "observations.json")); if err != nil { t.Fatal(err) }
 var observations []struct { Name string `json:"name"`; Args []string `json:"args"`; Stdout string `json:"stdout"`; Stderr string `json:"stderr"` }
 if err := json.Unmarshal(raw, &observations); err != nil { t.Fatal(err) }
 if len(observations) != 6 { t.Fatalf("need all six independently validated observations, got %d", len(observations)) }
 for _, observation := range observations {
  t.Run(observation.Name, func(t *testing.T) {
   if observation.Stderr != "" { t.Fatal("independent oracle requires empty stderr") }
   runCampaignCompilerStrictProjectOracle47Args(t, files, "app.Main", observation.Stdout, observation.Args...)
  })
 }
}
