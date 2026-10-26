package transpiler

import (
	"os"
	"testing"
)

func TestAnonymousAbstractCollectionCanonicalShadowJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/anonymous_abstract_collection/Shadow.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>anonymous</groupId><artifactId>shadow</artifactId><version>1</version></project>`,
		"src/main/java/anonymousshadow/Shadow.java": string(source),
	}, "anonymousshadow.Shadow", "source-binder-and-qualified-owner=true\n")
}

// The original Defaults control is retained as an unaccepted static Set.removeAll gate.
func TestAnonymousAbstractCollectionInheritedRemovalJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/anonymous_abstract_collection/SourceRemovalControl.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>anonymous</groupId><artifactId>collections</artifactId><version>1</version></project>`,
		"src/main/java/anonymouscollections/Main.java": string(source),
	}, "anonymouscollections.Main", "ancestry-defaults-generic=true\noverrides-order=true\nexecution-mutation-prefix=true\nselectors-allocation=true\n")
}

func TestAnonymousAbstractCollectionMethodBoundariesJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/anonymous_abstract_collection/Context.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>anonymous</groupId><artifactId>context</artifactId><version>1</version></project>`,
		"src/main/java/anonymouscontext/Context.java": string(source),
	}, "anonymouscontext.Context", "nested-method-boundaries-execution=true\n")
}

func TestAnonymousAbstractCollectionDefaultOverloadsJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/anonymous_abstract_collection/Overrides.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>anonymous</groupId><artifactId>overloads</artifactId><version>1</version></project>`,
		"src/main/java/anonymousoverrides/Overrides.java": string(source),
	}, "anonymousoverrides.Overrides", "default-overload-field-dispatch=true\n")
}

func TestAnonymousAbstractCollectionReferenceComparisonJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/anonymous_abstract_collection/Reference.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>anonymous</groupId><artifactId>reference</artifactId><version>1</version></project>`,
		"src/main/java/anonymousreference/Reference.java": string(source),
	}, "anonymousreference.Reference", "collection-reference-identity-order=true\n")
}

func TestAnonymousAbstractCollectionReferenceProof(t *testing.T) {
	helper := setupParseHelper(t, `class Set {} class Collection {} class AbstractSet {} class AbstractCollection {}
        abstract class Declared extends java.util.AbstractSet<String> {}
        abstract class MapOnly extends java.util.AbstractMap<String,String> {}
        class Shape { int size(){return 0;} java.util.Iterator<String> iterator(){return null;} }
        class Binders<Set,Collection,AbstractSet,AbstractCollection,Declared>{}`)
	for _, candidate := range []string{"java.util.Set<String>", "java.util.Collection<String>", "java.util.AbstractSet<String>", "java.util.AbstractCollection<String>", "Declared"} {
		if !abstractCollectionReferenceComparison(candidate, helper.Ctx) {
			t.Errorf("declared collection identity missing: %s", candidate)
		}
	}
	for _, candidate := range []string{"Set", "Collection", "AbstractSet", "AbstractCollection", "Shape", "MapOnly", "java.util.Map<String,String>", "java.util.Set<String>[]", "Declared[]", "int", "void", "Missing"} {
		if abstractCollectionReferenceComparison(candidate, helper.Ctx) {
			t.Errorf("noncollection identity proof admitted: %s", candidate)
		}
	}
	ctx := classScopeCtx(helper.File.Symbols.FindClassScope("Binders"), helper.Ctx)
	for _, candidate := range []string{"Set", "Collection", "AbstractSet", "AbstractCollection", "Declared"} {
		if abstractCollectionReferenceComparison(candidate, ctx) {
			t.Errorf("binder admitted as platform collection: %s", candidate)
		}
	}
}
