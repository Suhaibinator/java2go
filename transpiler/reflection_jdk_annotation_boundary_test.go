package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

const reflectionNativeAnnotationBoundarySource = `
import java.lang.annotation.*;
@java.lang.Deprecated @Known("kept")
public class NativeAnnotationBoundary {
 public static int count(){return NativeAnnotationBoundary.class.getAnnotations().length;}
 public static void main(String[] args) {
  System.out.println("known="+NativeAnnotationBoundary.class.getAnnotation(Known.class).value());
  System.out.println("other="+(NativeAnnotationBoundary.class.getAnnotation(Other.class)==null)+":"+NativeAnnotationBoundary.class.isAnnotationPresent(Other.class));
  System.out.println("native="+NativeAnnotationBoundary.class.isAnnotationPresent(java.lang.Deprecated.class));
 }
}
@Retention(RetentionPolicy.RUNTIME) @interface Known {String value();}
@Retention(RetentionPolicy.RUNTIME) @interface Other {}
`

func TestReflectionUnavailableNativeAnnotationArrayRefuses(t *testing.T) {
	output := renderGoFileFromJava(t, reflectionNativeAnnotationBoundarySource)
	runGeneratedWithStdjava(t, output, `
package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestUnavailableArray(t *testing.T){
 var failure any
 func(){defer func(){failure=recover()}();Count()}()
 if !j.CaughtAs(failure,"UnsupportedOperationException"){t.Fatalf("unmodeled annotation silently omitted: %T %v",failure,failure)}
}
`)
}

func TestReflectionUnavailableNativeAnnotationSelectiveJDK21(t *testing.T) {
	pom, err := os.ReadFile(filepath.Join("testdata", "accessible_annotation_gson_minimal", "project", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": string(pom), "src/main/java/NativeAnnotationBoundary.java": reflectionNativeAnnotationBoundarySource}, "NativeAnnotationBoundary", "known=kept\nother=true:false\nnative=true\n")
}

func TestReflectionAnnotationPolicyOwnershipJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "annotation_policy_ownership", "shadow.PolicyWitness")
}
