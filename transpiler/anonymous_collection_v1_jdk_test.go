package transpiler

import (
	"os"
	"testing"
)

func TestAnonymousCollectionFullConcurrentThreeJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/map_reference_projection/ConcurrentMapControl.java")
	if err != nil {
		t.Fatal(err)
	}
	want := "basic-erasure-source=true\nputAll-order-prefix=true\nlive-views-entry=true\ncallbacks-atomic=true\n"
	observations := []campaignCompilerProjectObservation{{name: "repeat1", stdout: want}, {name: "repeat2", stdout: want}, {name: "repeat3", stdout: want}}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>concurrent</groupId><artifactId>original</artifactId><version>1</version></project>`, "src/main/java/campaign/concurrentmap/ConcurrentMapControl.java": string(source)}, "campaign.concurrentmap.ConcurrentMapControl", observations)
}
func TestAnonymousCollectionBinderCollisionThreeJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/anonymous_collection_collision_v1/AnonymousBinderCollisionControl.java")
	if err != nil {
		t.Fatal(err)
	}
	want := "object-defaults=true\nsource-overloads=true\nsame-execution=true\n"
	observations := []campaignCompilerProjectObservation{{name: "repeat1", stdout: want}, {name: "repeat2", stdout: want}, {name: "repeat3", stdout: want}}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>anonymous</groupId><artifactId>binders</artifactId><version>1</version></project>`, "src/main/java/anonymouscollision/AnonymousBinderCollisionControl.java": string(source)}, "anonymouscollision.AnonymousBinderCollisionControl", observations)
}

func TestAnonymousCollectionFrozenFiveThreeJDK21(t *testing.T) {
	controls := []struct{ file, path, main, want string }{
		{"Shadow.java", "anonymousshadow/Shadow.java", "anonymousshadow.Shadow", "source-binder-and-qualified-owner=true\n"},
		{"SourceRemovalControl.java", "anonymouscollections/Main.java", "anonymouscollections.Main", "ancestry-defaults-generic=true\noverrides-order=true\nexecution-mutation-prefix=true\nselectors-allocation=true\n"},
		{"Context.java", "anonymouscontext/Context.java", "anonymouscontext.Context", "nested-method-boundaries-execution=true\n"},
		{"Overrides.java", "anonymousoverrides/Overrides.java", "anonymousoverrides.Overrides", "default-overload-field-dispatch=true\n"},
		{"Reference.java", "anonymousreference/Reference.java", "anonymousreference.Reference", "collection-reference-identity-order=true\n"},
	}
	for _, control := range controls {
		t.Run(control.main, func(t *testing.T) {
			source, err := os.ReadFile("testdata/anonymous_abstract_collection/" + control.file)
			if err != nil {
				t.Fatal(err)
			}
			observations := []campaignCompilerProjectObservation{{name: "repeat1", stdout: control.want}, {name: "repeat2", stdout: control.want}, {name: "repeat3", stdout: control.want}}
			runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>anonymous</groupId><artifactId>frozen</artifactId><version>1</version></project>`, "src/main/java/" + control.path: string(source)}, control.main, observations)
		})
	}
}
