package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Every legal project receives an actual JDK oracle before strict translation.
func TestCampaignWrapperConstantsDimensionsJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>wrapperfacts</groupId><artifactId>dimensions</artifactId><version>1</version></project>
`,
		"src/main/java/app/Main.java": `package app;
public class Main {
 static int stored=2;
 static int type(int n){return 1;} static int type(long n){return 2;}
 static int type(float n){return 3;} static int type(double n){return 4;}
 public static void main(String[] args){
  int[] fields={Byte.SIZE,Byte.BYTES,Short.SIZE,Short.BYTES,Integer.SIZE,Integer.BYTES,Long.SIZE,Long.BYTES,Float.SIZE,Float.BYTES,Double.SIZE,Double.BYTES,Float.PRECISION,Double.PRECISION,Float.MAX_EXPONENT,Float.MIN_EXPONENT,Double.MAX_EXPONENT,Double.MIN_EXPONENT};
  for(int i=0;i<fields.length;i++)System.out.println(fields[i]);
  System.out.println(type(java.lang.Long.BYTES)); System.out.println(type(Float.MIN_EXPONENT));
  Object boxed=Short.SIZE; System.out.println(boxed.getClass()==Integer.class);
  int offset=3;offset+=Long.BYTES;offset-=Byte.BYTES;System.out.println(offset);
  stored+=Double.BYTES;System.out.println(stored);
  int[] values=new int[Long.BYTES];values[Byte.BYTES]+=Float.PRECISION;System.out.println(values[1]);
  byte tiny=127;tiny+=Byte.BYTES;short narrow=32767;narrow+=Short.BYTES;char unit=65535;unit+=Byte.BYTES;
  System.out.println(tiny);System.out.println(narrow);System.out.println((int)unit);
  System.out.println(1L<<Integer.SIZE);System.out.println(type((long)Long.BYTES));
  System.out.println(Imported.read());
 }
}
`,
		"src/main/java/app/Imported.java": `package app;import static java.lang.Long.BYTES;import static java.lang.Float.PRECISION;import static java.lang.Float.MAX_EXPONENT;import static java.lang.Float.MIN_EXPONENT;import static java.lang.Long.BYTES;
public class Imported{public static int read(){return BYTES+PRECISION+MAX_EXPONENT+MIN_EXPONENT;}}
`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}

func TestCampaignWrapperConstantsExistingJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>wrapperfacts</groupId><artifactId>existing</artifactId><version>1</version></project>
`,
		"src/main/java/app/Main.java": `package app;public class Main{public static void main(String[] args){
 System.out.println(Boolean.TRUE==Boolean.valueOf(true));System.out.println(Boolean.FALSE==Boolean.valueOf(false));
 System.out.println(Boolean.TYPE==boolean.class);System.out.println(Byte.TYPE==byte.class);System.out.println(Short.TYPE==short.class);System.out.println(Integer.TYPE==int.class);System.out.println(Long.TYPE==long.class);System.out.println(Float.TYPE==float.class);System.out.println(Double.TYPE==double.class);
 System.out.println(Byte.MIN_VALUE);System.out.println(Byte.MAX_VALUE);System.out.println(Short.MIN_VALUE);System.out.println(Short.MAX_VALUE);System.out.println(Integer.MIN_VALUE);System.out.println(Integer.MAX_VALUE);System.out.println(Long.MIN_VALUE);System.out.println(Long.MAX_VALUE);
 System.out.println(Float.MIN_VALUE>0.0f&&Float.MIN_VALUE<Float.MIN_NORMAL&&Float.MIN_NORMAL<Float.MAX_VALUE);System.out.println(Double.MIN_VALUE>0.0&&Double.MIN_VALUE<Double.MIN_NORMAL&&Double.MIN_NORMAL<Double.MAX_VALUE);
 System.out.println(Float.NaN!=Float.NaN);System.out.println(Double.NaN!=Double.NaN);System.out.println(Float.POSITIVE_INFINITY>Float.MAX_VALUE&&Float.NEGATIVE_INFINITY<0.0f);System.out.println(Double.POSITIVE_INFINITY>Double.MAX_VALUE&&Double.NEGATIVE_INFINITY<0.0);
}}
`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}

func TestCampaignWrapperConstantsInitializationJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>wrapperfacts</groupId><artifactId>initialization</artifactId><version>1</version></project>
`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static RuntimeException marker=new RuntimeException("marker");
 public static void main(String[] args){
  System.out.println(Holder.WIDTH);System.out.println(Holder.EXP);System.out.println(ImportedHolder.WIDTH);
  System.out.println(Holder.value);System.out.println(Holder.value);System.out.println(ImportedHolder.value);
  System.out.println(Failing.WIDTH);try{System.out.println(Failing.value);}catch(ExceptionInInitializerError failure){System.out.println(failure.getCause()==marker);}
 }
}
`,
		"src/main/java/app/Holder.java": `package app;public class Holder{public static final int WIDTH=java.lang.Long.BYTES;public static final int EXP=java.lang.Float.MIN_EXPONENT;static{System.out.println("holder-init");}public static int value=9;}
`,
		"src/main/java/app/ImportedHolder.java": `package app;import static java.lang.Integer.SIZE;public class ImportedHolder{public static final int WIDTH=SIZE;static{System.out.println("imported-init");}public static int value=17;}
`,
		"src/main/java/app/Failing.java": `package app;public class Failing{public static final int WIDTH=java.lang.Double.BYTES;static{if(true)throw Main.marker;}public static int value=23;}
`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}

func TestCampaignWrapperConstantsGuardsJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"src/main/java/app/Long.java": `package app;public class Long{public static int BYTES=61;}
`,
		"src/main/java/app/SamePackage.java": `package app;public class SamePackage{public static int read(){return Long.BYTES;}}
`,
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>wrapperfacts</groupId><artifactId>guards</artifactId><version>1</version></project>
`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static class Long{static int BYTES=47;static int SIZE=91;}
 static int local(){class Long{static int BYTES=67;}return Long.BYTES;}
 static int effects;static Long source(){effects++;return null;}
 static <Long> int bound(){return java.lang.Long.BYTES;}
 public static void main(String[] args){System.out.println(Long.BYTES);System.out.println(Long.SIZE);System.out.println(java.lang.Long.BYTES);System.out.println(bound());System.out.println(source().BYTES);System.out.println(effects);System.out.println(Imported.read());System.out.println(Foreign.read());System.out.println(SamePackage.read());System.out.println(local());}
}
`,
		"src/main/java/app/Imported.java": `package app;import static java.lang.Long.*;public class Imported{public static int read(){int BYTES=73;return BYTES+SIZE;}}
`,
		"src/main/java/app/Foreign.java": `package app;import foreign.Long;public class Foreign{public static int read(){return Long.BYTES;}}
`,
		"src/main/java/foreign/Long.java": `package foreign;public class Long{public static int BYTES=53;}
`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}

func campaignWrapperConstantsStrictProject(t *testing.T, files map[string]string) {
	t.Helper()
	root := t.TempDir()
	defer campaignWrapperConstantsExport(t, root)
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var sources []string
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(name) == ".java" {
			sources = append(sources, path)
		}
	}
	sort.Strings(sources)
	run := func(stage, dir, name string, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-p=1 -race -mod=readonly")
		cmd.WaitDelay = 2 * time.Second
		var out, errout bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errout
		err := cmd.Run()
		if artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR"); artifactDir != "" {
			for leaf, data := range map[string][]byte{stage + ".stdout": out.Bytes(), stage + ".stderr": errout.Bytes(), stage + ".argv": []byte(strings.Join(cmd.Args, "\n") + "\n")} {
				if writeErr := writeProjectFile(filepath.Join(artifactDir, t.Name(), "commands", leaf), data); writeErr != nil {
					t.Fatal(writeErr)
				}
			}
		}
		if err != nil || ctx.Err() != nil {
			t.Fatalf("%s: %v / %v stdout=%q stderr=%q", stage, err, ctx.Err(), out.String(), errout.String())
		}
		if errout.Len() != 0 {
			t.Fatalf("%s stderr=%q", stage, errout.String())
		}
		t.Logf("%s: %v passed", stage, cmd.Args)
		return out.Bytes()
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	classes := filepath.Join(root, "classes")
	run("javac", root, javac, append([]string{"--release", "21", "-d", classes}, sources...)...)
	want := run("JVM", root, java, "-cp", classes, "app.Main")
	if len(want) == 0 {
		t.Fatal("JVM emitted no observations")
	}
	t.Logf("actual JDK stdout %q", want)
	if artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR"); artifactDir != "" {
		for name, raw := range map[string][]byte{"jdk.stdout": want, "jdk.stderr": {}} {
			if err := writeProjectFile(filepath.Join(artifactDir, t.Name(), name), raw); err != nil {
				t.Fatal(err)
			}
		}
	}
	generated := filepath.Join(root, "generated")
	// The ordinary project API performs strict conversion of every source file.
	if err := Run([]string{"-strict", "-maven", root, "-main-class", "app.Main", "-runtime", repo, "-output", generated}); err != nil {
		t.Fatal(err)
	}
	originalMod, err := os.ReadFile(filepath.Join(generated, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeMod, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeSum, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	firstLine := strings.SplitN(string(originalMod), "\n", 2)[0]
	buildMod := firstLine + "\n" + strings.SplitN(string(runtimeMod), "\n", 2)[1] + "\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => " + filepath.ToSlash(repo) + "\n"
	if err := os.WriteFile(filepath.Join(generated, "go.mod"), []byte(buildMod), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated, "go.sum"), runtimeSum, 0600); err != nil {
		t.Fatal(err)
	}
	run("all generated packages race build", generated, "go", "build", "-p=1", "-mod=readonly", "-race", "./...")
	binary := filepath.Join(root, "generated-app")
	run("entry point race build", generated, "go", "build", "-p=1", "-mod=readonly", "-race", "-o", binary, "./cmd/app")
	got := run("generated Go", root, binary)
	if !bytes.Equal(got, want) {
		t.Fatalf("Go=%q Java=%q", got, want)
	}
	if artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR"); artifactDir != "" {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			name := entry.Name()
			if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".java" && name != "go.mod" && name != "go.sum" && name != "pom.xml" {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return writeProjectFile(filepath.Join(artifactDir, t.Name(), relative), raw)
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("matching observations %q", got)
}

// Retain unchanged source and generated evidence on every failure path.
func campaignWrapperConstantsExport(t *testing.T, root string) {
	t.Helper()
	artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR")
	if artifactDir == "" {
		return
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name := entry.Name()
		if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".java" && name != "go.mod" && name != "go.sum" && name != "pom.xml" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeProjectFile(filepath.Join(artifactDir, t.Name(), relative), data)
	})
	if err != nil {
		t.Errorf("export retained evidence: %v", err)
	}
}
