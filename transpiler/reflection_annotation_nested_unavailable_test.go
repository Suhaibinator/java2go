package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReflectionNestedAnnotationUnavailableSelectiveJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "annotation_nested_unavailable", "NestedAnnotationBoundary")
}

func TestReflectionNestedAnnotationRequestedRemainsExplicit(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "annotation_nested_unavailable", "project", "src", "main", "java", "NestedAnnotationBoundary.java"))
	if err != nil {
		t.Fatal(err)
	}
	output := renderGoFileFromJava(t, string(data))
	runGeneratedWithStdjava(t, output, `
package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestRequestedUnavailable(t *testing.T){
 for _,tc:=range []struct{name string;call func()int32}{
  {"nested",Requested},{"array",ArrayRequested},{"default",DefaultRequested},{"whole-array",AllRequested},
 }{
  t.Run(tc.name,func(t *testing.T){
   var failure any
   func(){defer func(){failure=recover()}();tc.call()}()
   if !j.CaughtAs(failure,"UnsupportedOperationException"){t.Fatalf("requested unsupported annotation was not explicit: %T %v",failure,failure)}
   throwable:=failure.(j.Throwable)
   if throwable.Message()!="annotation proxy metadata is unavailable" || j.GetCause(throwable)!=nil {t.Fatalf("wrong unavailable factory failure: %v",failure)}
  })
 }
}
`)
}
