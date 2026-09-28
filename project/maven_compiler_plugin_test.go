package project

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compilerMetadataPom(t *testing.T, properties, plugin string) string {
	t.Helper()
	return pomFile(t, t.TempDir(), "", rootCoordinates+"<properties>"+properties+"</properties><build><plugins><plugin>"+plugin+"</plugin></plugins></build>")
}

const compilerMetadataCoordinates = `<groupId>org.apache.maven.plugins</groupId><artifactId>maven-compiler-plugin</artifactId><version>3.16.0</version>`
const compilerMetadataProperties = `<maven.compiler.release>21</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>`

func TestDiscoverCompilerPluginFrozenStringPrerequisite(t *testing.T) {
	original, err := os.ReadFile("testdata/compiler-plugin-string-prereq.xml")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "pom.xml")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := Discover(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Modules) != 1 {
		t.Fatalf("modules=%d", len(plan.Modules))
	}
	m := plan.Modules[0]
	if m.Compiler == nil || *m.Compiler != (CompilerMetadata{PluginVersion: "3.16.0", Release: "21", Encoding: "UTF-8"}) {
		t.Fatalf("resolved compiler metadata=%#v", m.Compiler)
	}
	if m.GroupID != "prereq.string" || m.ArtifactID != "string-abi-prereq" || m.Version != "1.0.0" {
		t.Fatalf("coordinates changed: %#v", m)
	}
	if m.SourceDirectory != filepath.Join(root, "src/main/java") {
		t.Fatalf("source path=%s", m.SourceDirectory)
	}
	if len(m.Resources) != 1 || m.Resources[0].Directory != filepath.Join(root, "src/main/resources") {
		t.Fatalf("resources=%#v", m.Resources)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("Discover rewrote frozen POM")
	}
}

func TestDiscoverCompilerPluginBoundedMetadata(t *testing.T) {
	for _, tc := range []struct{ name, props, plugin string }{
		{"properties", compilerMetadataProperties, compilerMetadataCoordinates},
		{"proc none property", compilerMetadataProperties + `<maven.compiler.proc>none</maven.compiler.proc>`, compilerMetadataCoordinates},
		{"encoding alias override", `<maven.compiler.release>21</maven.compiler.release><project.build.sourceEncoding>ISO-8859-1</project.build.sourceEncoding><encoding>UTF-8</encoding>`, compilerMetadataCoordinates},
		{"default group", compilerMetadataProperties, `<artifactId>maven-compiler-plugin</artifactId><version>3.16.0</version>`},
		{"property interpolation", compilerMetadataProperties + `<compiler.version>3.16.0</compiler.version>`, strings.Replace(compilerMetadataCoordinates, "3.16.0", "${compiler.version}", 1)},
		{"explicit configuration", "", compilerMetadataCoordinates + `<configuration><release>21</release><encoding>UTF-8</encoding><proc>none</proc></configuration>`},
		{"configuration overrides properties", `<maven.compiler.release>17</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>`, compilerMetadataCoordinates + `<configuration><release>21</release></configuration>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Discover(compilerMetadataPom(t, tc.props, tc.plugin), nil)
			if err != nil {
				t.Fatal(err)
			}
			want := CompilerMetadata{PluginVersion: "3.16.0", Release: "21", Encoding: "UTF-8"}
			if tc.name == "proc none property" || tc.name == "explicit configuration" {
				want.Proc = "none"
			}
			if plan.Modules[0].Compiler == nil || *plan.Modules[0].Compiler != want {
				t.Fatalf("metadata=%#v want=%#v", plan.Modules[0].Compiler, want)
			}
		})
	}
}

func TestDiscoverCompilerPluginRejectsUnimplementedSemantics(t *testing.T) {
	for _, tc := range []struct{ name, props, plugin string }{
		{"wrong group", compilerMetadataProperties, strings.Replace(compilerMetadataCoordinates, "org.apache.maven.plugins", "other.vendor", 1)},
		{"unvalidated fixed version", compilerMetadataProperties, strings.Replace(compilerMetadataCoordinates, "3.16.0", "3.15.0", 1)},
		{"missing version", compilerMetadataProperties, `<artifactId>maven-compiler-plugin</artifactId>`},
		{"version range", compilerMetadataProperties, strings.Replace(compilerMetadataCoordinates, "3.16.0", "[3,4)", 1)},
		{"snapshot", compilerMetadataProperties, strings.Replace(compilerMetadataCoordinates, "3.16.0", "3.16.0-SNAPSHOT", 1)},
		{"unknown version property", compilerMetadataProperties, strings.Replace(compilerMetadataCoordinates, "3.16.0", "${unknown}", 1)},
		{"wrong release", strings.Replace(compilerMetadataProperties, ">21<", ">17<", 1), compilerMetadataCoordinates},
		{"missing release", `<project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>`, compilerMetadataCoordinates},
		{"non UTF8", strings.Replace(compilerMetadataProperties, "UTF-8", "ISO-8859-1", 1), compilerMetadataCoordinates},
		{"source target", compilerMetadataProperties + `<maven.compiler.source>21</maven.compiler.source><maven.compiler.target>21</maven.compiler.target>`, compilerMetadataCoordinates},
		{"argument property", compilerMetadataProperties + `<maven.compiler.compilerArgument>--enable-preview</maven.compiler.compilerArgument>`, compilerMetadataCoordinates},
		{"proc full property", compilerMetadataProperties + `<maven.compiler.proc>full</maven.compiler.proc>`, compilerMetadataCoordinates},
		{"encoding alias override", compilerMetadataProperties + `<encoding>ISO-8859-1</encoding>`, compilerMetadataCoordinates},
		{"duplicate configuration field", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><release>21</release><release>17</release></configuration>`},
		{"fork property", compilerMetadataProperties + `<maven.compiler.fork>true</maven.compiler.fork>`, compilerMetadataCoordinates},
		{"processors", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><annotationProcessors><annotationProcessor>example.Processor</annotationProcessor></annotationProcessors></configuration>`},
		{"processor path", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><annotationProcessorPaths><path><groupId>x</groupId><artifactId>processor</artifactId><version>1</version></path></annotationProcessorPaths></configuration>`},
		{"proc full", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><proc>full</proc></configuration>`},
		{"compiler args", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><compilerArgs><arg>--enable-preview</arg></compilerArgs></configuration>`},
		{"executions", compilerMetadataProperties, compilerMetadataCoordinates + `<executions><execution><goals><goal>compile</goal></goals></execution></executions>`},
		{"plugin dependencies", compilerMetadataProperties, compilerMetadataCoordinates + `<dependencies><dependency><groupId>x</groupId><artifactId>y</artifactId><version>1</version></dependency></dependencies>`},
		{"extensions", compilerMetadataProperties, compilerMetadataCoordinates + `<extensions>true</extensions>`},
		{"toolchain", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><jdkToolchain><version>17</version></jdkToolchain></configuration>`},
		{"config merge attribute", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration combine.self="override"><release>21</release></configuration>`},
		{"unknown config", compilerMetadataProperties, compilerMetadataCoordinates + `<configuration><surprise>true</surprise></configuration>`},
		{"unknown plugin metadata", compilerMetadataProperties, compilerMetadataCoordinates + `<surprise>true</surprise>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Discover(compilerMetadataPom(t, tc.props, tc.plugin), nil); err == nil {
				t.Fatal("unsupported compiler semantics silently accepted")
			}
		})
	}
}

func TestDiscoverCompilerPluginRejectsImplicitInheritance(t *testing.T) {
	root := t.TempDir()
	pomFile(t, root, "", rootCoordinates+`<properties>`+compilerMetadataProperties+`</properties><build><plugins><plugin>`+compilerMetadataCoordinates+`</plugin></plugins></build>`)
	child := pomFile(t, root, "child", `<parent><groupId>test</groupId><artifactId>root</artifactId><version>1</version></parent><artifactId>child</artifactId>`)
	if _, err := Discover(child, nil); err == nil {
		t.Fatal("unimplemented compiler plugin inheritance silently accepted")
	}
}

func TestDiscoverCompilerPluginMetadataAbsentForSourceOnly(t *testing.T) {
	plan, err := Discover(pomFile(t, t.TempDir(), "", rootCoordinates), nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Modules[0].Compiler != nil {
		t.Fatal("invented compiler metadata for source-only project")
	}
}

func TestDiscoverCompilerPluginPropertyInheritanceAndOverride(t *testing.T) {
	root := t.TempDir()
	pomFile(t, root, "", rootCoordinates+`<properties><compiler.version>3.16.0</compiler.version><maven.compiler.release>17</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding></properties>`)
	child := pomFile(t, root, "child", `<parent><groupId>test</groupId><artifactId>root</artifactId><version>1</version></parent><artifactId>child</artifactId><properties><maven.compiler.release>21</maven.compiler.release></properties><build><plugins><plugin><artifactId>maven-compiler-plugin</artifactId><version>${compiler.version}</version></plugin></plugins></build>`)
	plan, err := Discover(child, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Modules[0].Compiler == nil || plan.Modules[0].Compiler.Release != "21" || plan.Modules[0].Compiler.PluginVersion != "3.16.0" {
		t.Fatalf("child context not propagated: %#v", plan.Modules[0].Compiler)
	}
}
