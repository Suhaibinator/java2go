package transpiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// ReflectionAccessFilter's anonymous static instances require real interface
// field storage and initialization; fields are not merely signature metadata.
func TestCampaignInterfaceStaticFieldsJVMParity(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		name := "cross_package"
		if cycle {
			name = "package_cycle"
		}
		t.Run(name, func(t *testing.T) {
			backEdge := ""
			if cycle {
				backEdge = "public static api.Policy identity(api.Policy value){return value;}"
			}
			files := map[string]string{
				"pom.xml": `<project><groupId>campaign.probe</groupId><artifactId>interface-fields</artifactId><version>1</version></project>`,
				"src/main/java/support/Trace.java": fmt.Sprintf(`package support;
public final class Trace {
 public static int count=0;
 public static int mark(){return ++count;}
 %s
}`, backEdge),
				"src/main/java/api/Policy.java": `package api;
import support.Trace;
public interface Policy {
 int LIMIT=7;
 String LABEL="policy";
 int FIRST=Trace.mark();
 Policy SHARED=new Policy(){
  private final int serial=Trace.mark();
  public Decision check(int value){return value>2?Decision.BLOCK:Decision.ALLOW;}
  public int serial(){return serial;}
 };
 int LAST=Trace.mark();
 enum Decision { ALLOW, BLOCK }
 Decision check(int value);
 int serial();
}`,
				"src/main/java/app/Main.java": `package app;
import api.Policy;
import support.Trace;
public final class Main {
 static Policy load(){return Policy.SHARED;}
 public static void main(String[] args){
  System.out.println(Policy.LIMIT+":"+Policy.LABEL+":"+Trace.count);
  Policy first=load();
  Policy second=load();
  System.out.println(first==second);
  System.out.println(first.serial()+":"+second.serial()+":"+first.check(1).name()+":"+second.check(4).name());
  System.out.println(Policy.FIRST+":"+Policy.LAST+":"+Trace.count);
 }
}`,
			}
			runInterfaceStaticFieldsOracle(t, files, "app.Main")
		})
	}
}

func runInterfaceStaticFieldsOracle(t *testing.T, files map[string]string, mainClass string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
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
	classes := filepath.Join(root, "classes")
	if output, err := exec.CommandContext(ctx, javac, append([]string{"--release", "21", "-d", classes}, sources...)...).CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, output)
	}
	want, err := exec.CommandContext(ctx, java, "-cp", classes, mainClass).CombinedOutput()
	if err != nil {
		t.Fatalf("JVM: %v\n%s", err, want)
	}
	if string(want) != "7:policy:0\ntrue\n2:2:ALLOW:BLOCK\n1:3:3\n" {
		t.Fatalf("unexpected JVM oracle %q", want)
	}
	t.Logf("JVM oracle: %q", want)
	repo, _ := filepath.Abs("..")
	generated := filepath.Join(root, "generated")
	command := exec.CommandContext(ctx, "go", "run", "./cmd/java2go", "-strict", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-output", generated)
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compiler: %v\n%s", err, output)
	}
	command = exec.CommandContext(ctx, "go", "build", "-race", "-mod=mod", "./...")
	command.Dir = generated
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("all generated packages: %v\n%s", err, output)
	}
	command = exec.CommandContext(ctx, "go", "run", "-race", "-mod=mod", "./cmd/app")
	command.Dir = generated
	command.Env = append(os.Environ(), "GOWORK=off")
	got, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated Go: %v\n%s", err, got)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Go %q differs from JVM %q", got, want)
	}
}
