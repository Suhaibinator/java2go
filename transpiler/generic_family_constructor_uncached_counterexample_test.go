package transpiler

import (
	"os"
	"strings"
	"testing"
)

func TestGenericFamilyEnumConstructorUncachedRender(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_enum_constructor_late_cast/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	source = strings.Replace(source, "package genericenumconstructorvariation;", "", 1)
	source = strings.Replace(source, "public static void main(String[] args)", "public static String run()", 1)
	statement := `System.out.println(text.read()+"|"+built+"|"+same+"|"+delayed+"|"+absent);`
	result := `return text.read()+"|"+built+"|"+same+"|"+delayed+"|"+absent;`
	if strings.Count(source, statement) != 1 {
		t.Fatal("missing exact legal-Java observation")
	}
	source = strings.Replace(source, statement, result, 1)
	helper := setupParseHelper(t, source)
	if helper.Ctx.genericFamilies != nil {
		t.Fatal("counterexample requires the existing uncached helper path")
	}
	assertGeneratedLocalConstructorResult(t, source, "s|true|true|true|true")
}

func TestGenericFamilyEnumConstructorForLocalsUncachedRender(t *testing.T) {
	for _, variant := range genericFamilyConstructorLoopSources(t) {
		t.Run(variant.name, func(t *testing.T) {
			source := strings.Replace(variant.source, "package genericenumconstructorloop;", "", 1)
			source = strings.Replace(source, "public static void main(String[] args)", "public static String run()", 1)
			statement := `System.out.println(text.read()+"|"+built+"|"+same+"|"+delayed+"|"+absent);`
			result := `return text.read()+"|"+built+"|"+same+"|"+delayed+"|"+absent;`
			if strings.Count(source, statement) != 1 {
				t.Fatal("missing exact legal-Java loop observation")
			}
			source = strings.Replace(source, statement, result, 1)
			helper := setupParseHelper(t, source)
			if helper.Ctx.genericFamilies != nil {
				t.Fatal("fixture must remain uncached")
			}
			assertGeneratedLocalConstructorResult(t, source, "s|true|true|true|true")
		})
	}
}
