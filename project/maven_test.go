package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pomFile(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "pom.xml"), []byte("<project><modelVersion>4.0.0</modelVersion>"+body+"</project>"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const rootCoordinates = "<groupId>test</groupId><artifactId>root</artifactId><version>1</version>"

func TestDiscoverReactorAndTransitiveSources(t *testing.T) {
	root := t.TempDir()
	pomFile(t, root, "", rootCoordinates+`<packaging>pom</packaging><properties><dep.version>2</dep.version></properties><dependencyManagement><dependencies><dependency><groupId>vendor</groupId><artifactId>first</artifactId><version>${dep.version}</version></dependency></dependencies></dependencyManagement><modules><module>app</module></modules>`)
	pomFile(t, root, "app", `<parent><groupId>test</groupId><artifactId>root</artifactId><version>1</version></parent><artifactId>app</artifactId><build><sourceDirectory>${project.basedir}/java</sourceDirectory><resources><resource><directory>assets</directory><targetPath>messages</targetPath></resource></resources></build><dependencies><dependency><groupId>vendor</groupId><artifactId>first</artifactId></dependency><dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>4</version><scope>test</scope></dependency></dependencies>`)
	first := pomFile(t, root, "first", `<groupId>vendor</groupId><artifactId>first</artifactId><version>2</version><dependencies><dependency><groupId>vendor</groupId><artifactId>second</artifactId><version>3</version></dependency></dependencies>`)
	second := pomFile(t, root, "second", `<groupId>vendor</groupId><artifactId>second</artifactId><version>3</version>`)
	unused := pomFile(t, root, "unused", `<groupId>vendor</groupId><artifactId>unused</artifactId><version>1</version>`)
	plan, err := Discover(root, []string{"vendor:first=" + first, "vendor:second=" + second, "vendor:unused=" + unused})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Modules) != 4 {
		t.Fatalf("expected root, app, first, second; got %d", len(plan.Modules))
	}
	for _, m := range plan.Modules {
		if m.ArtifactID == "app" {
			if m.SourceDirectory != filepath.Join(root, "app/java") {
				t.Fatal(m.SourceDirectory)
			}
			if len(m.Resources) != 1 || m.Resources[0].Target != "messages" {
				t.Fatal(m.Resources)
			}
		}
	}
}
func TestDiscoverRejectsUnimplementedMavenSemantics(t *testing.T) {
	cases := []struct{ name, extra, want string }{
		{"missing sources", `<dependencies><dependency><groupId>vendor</groupId><artifactId>lib</artifactId><version>1</version></dependency></dependencies>`, "-dependency-source vendor:lib="},
		{"profiles", `<profiles><profile><id>production</id></profile></profiles>`, "profiles"},
		{"plugins", `<build><plugins><plugin><artifactId>generator</artifactId></plugin></plugins></build>`, "build plugins"},
		{"filtering", `<build><resources><resource><directory>assets</directory><filtering>true</filtering></resource></resources></build>`, "filtered resources"},
		{"escaping target", `<build><resources><resource><directory>assets</directory><targetPath>../../escape</targetPath></resource></resources></build>`, "within resources"},
		{"undefined property", `<build><sourceDirectory>${unknown}/src</sourceDirectory></build>`, "unresolved Maven property"},
		{"recursive property", `<properties><a>${b}</a><b>${a}</b></properties><build><sourceDirectory>${a}</sourceDirectory></build>`, "recursive or malformed"},
		{"BOM", `<dependencyManagement><dependencies><dependency><groupId>x</groupId><artifactId>bom</artifactId><version>1</version><type>pom</type><scope>import</scope></dependency></dependencies></dependencyManagement>`, "BOM"},
		{"system dependency", `<dependencies><dependency><groupId>x</groupId><artifactId>lib</artifactId><version>1</version><scope>system</scope></dependency></dependencies>`, "scope"},
		{"war packaging", `<packaging>war</packaging>`, "packaging"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := pomFile(t, t.TempDir(), "", rootCoordinates+tc.extra)
			_, err := Discover(root, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}
func TestDiscoverRejectsWrongSourceVersionAndParent(t *testing.T) {
	root := t.TempDir()
	pomFile(t, root, "", rootCoordinates+`<dependencies><dependency><groupId>vendor</groupId><artifactId>lib</artifactId><version>1</version></dependency></dependencies>`)
	lib := pomFile(t, root, "lib", `<groupId>vendor</groupId><artifactId>lib</artifactId><version>2</version>`)
	if _, err := Discover(root, []string{"vendor:lib=" + lib}); err == nil || !strings.Contains(err.Error(), `requires version "1"`) {
		t.Fatal(err)
	}
	child := pomFile(t, root, "child", `<parent><groupId>wrong</groupId><artifactId>root</artifactId><version>1</version></parent><artifactId>child</artifactId>`)
	if _, err := Discover(child, nil); err == nil || !strings.Contains(err.Error(), "parent coordinates") {
		t.Fatal(err)
	}
}

func TestDiscoverResolvesInheritedDependencyVersionsInChildContext(t *testing.T) {
	for _, managed := range []bool{false, true} {
		name := "dependency"
		if managed {
			name = "dependencyManagement"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			declaration := `<dependency><groupId>vendor</groupId><artifactId>lib</artifactId><version>${lib.version}</version></dependency>`
			extra := `<dependencies>` + declaration + `</dependencies>`
			if managed {
				extra = `<dependencyManagement><dependencies>` + declaration + `</dependencies></dependencyManagement><dependencies><dependency><groupId>vendor</groupId><artifactId>lib</artifactId></dependency></dependencies>`
			}
			pomFile(t, root, "", rootCoordinates+`<properties><lib.version>1</lib.version></properties>`+extra)
			child := pomFile(t, root, "child", `<parent><groupId>test</groupId><artifactId>root</artifactId><version>1</version></parent><artifactId>child</artifactId><properties><lib.version>2</lib.version></properties>`)
			lib := pomFile(t, root, "lib", `<groupId>vendor</groupId><artifactId>lib</artifactId><version>2</version>`)
			plan, err := Discover(child, []string{"vendor:lib=" + lib})
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Modules) != 2 {
				t.Fatalf("expected child and source dependency, got %d", len(plan.Modules))
			}
		})
	}
}
