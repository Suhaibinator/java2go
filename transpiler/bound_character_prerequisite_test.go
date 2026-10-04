package transpiler

import (
 "os"
 "path/filepath"
 "strconv"
 "testing"
)

func TestCanonicalStringReplaceBoundCharacterJVM(t *testing.T) {
 source, err := os.ReadFile(filepath.Join("testdata", "bound_character_prerequisite", "Probe.java"))
 if err != nil { t.Fatal(err) }
 var observations []campaignCompilerProjectObservation
 for _, seed := range []int{17,41,97} {
  for repeat:=1;repeat<=3;repeat++ {
   name:=strconv.Itoa(seed)+"-"+strconv.Itoa(repeat)
   expected, err := os.ReadFile(filepath.Join("testdata", "bound_character_prerequisite", name+".stdout"))
   if err != nil { t.Fatal(err) }
   observations=append(observations,campaignCompilerProjectObservation{name:name,args:[]string{strconv.Itoa(seed)},stdout:string(expected)})
  }
 }
 runCampaignCompilerStrictProjectObservations(t,map[string]string{
  "pom.xml":`<project><modelVersion>4.0.0</modelVersion><groupId>boundchar</groupId><artifactId>probe</artifactId><version>1</version></project>`,
  "src/main/java/boundchar/Probe.java":string(source),
 },"boundchar.Probe",observations)
}
