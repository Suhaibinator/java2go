package transpiler

import (
	"strings"
	"testing"
)

func TestLocaleCategoryCanonicalOwners(t *testing.T) {
	source := `import java.util.Locale;
public class LocaleCategoryOwned {
 static String run(Locale.Category category) {
  Locale.setDefault(Locale.Category.FORMAT, Locale.US);
  Locale.setDefault(category, Locale.ENGLISH);
  return Locale.getDefault(category).getLanguage()+Locale.getDefault(Locale.Category.DISPLAY).getCountry();
 }
}`
	generated := renderGoFileFromJava(t, source)
	for _, need := range []string{"*stdjava.LocaleCategory", "stdjava.LocaleCategoryFORMAT", "stdjava.LocaleCategoryDISPLAY", "stdjava.LocaleSetDefaultCategory", "stdjava.LocaleGetDefaultCategory", "GetLanguageJavaString", "GetCountryJavaString"} {
		if !strings.Contains(generated, need) {
			t.Errorf("missing %s in %s", need, generated)
		}
	}
}

func TestLocaleCategorySourceShadowAndQualifiedJDK(t *testing.T) {
	source := `class Locale {
 static class Category { static final Category FORMAT = new Category(); }
 static int getDefault(Category category) { return 7; }
}
public class LocaleCategoryShadow {
 static int local() { return Locale.getDefault(Locale.Category.FORMAT); }
 static java.util.Locale jdk() { return java.util.Locale.getDefault(java.util.Locale.Category.FORMAT); }
}`
	generated := renderGoFileFromJava(t, source)
	if got := strings.Count(generated, "stdjava.LocaleGetDefaultCategory("); got != 1 {
		t.Fatalf("JDK category dispatch count %d: %s", got, generated)
	}
}

func TestLocaleCategoryArrayDescriptor(t *testing.T) {
	generated := renderGoFileFromJava(t, `import java.util.Locale; public class CategoryArray { static Locale.Category[] make() { return new Locale.Category[] { Locale.Category.FORMAT }; } }`)
	if !strings.Contains(generated, "java.util.Locale$Category") {
		t.Fatal(generated)
	}
}
