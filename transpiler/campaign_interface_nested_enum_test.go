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

// Gson ReflectionAccessFilter declares an implicitly public/static member enum
// in an interface and returns it from named and anonymous implementations.
// Exercise that general Java contract before and after package-cycle collapse.
func TestCampaignInterfaceNestedEnumJVMParity(t *testing.T) {
	for _, test := range []struct {
		name, api, impl, app string
		cycle                bool
	}{
		{"same_package", "p", "p", "p", false},
		{"cross_package", "api", "impl", "app", false},
		{"package_cycle", "api", "impl", "app", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			policyExtra, thresholdExtra, anonymousExtra := "", "", ""
			if test.cycle {
				policyExtra = test.impl + ".Threshold next();"
				thresholdExtra = "public Threshold next(){return this;}"
				anonymousExtra = "public Threshold next(){return new Threshold();}"
			}
			files := map[string]string{
				"pom.xml": `<project><groupId>campaign.probe</groupId><artifactId>interface-enum</artifactId><version>1</version></project>`,
				"src/main/java/" + test.api + "/Policy.java": fmt.Sprintf(`package %s;
public interface Policy {
 enum Decision { ALLOW, INDECISIVE, BLOCK }
 Decision check(int value);
 %s
}`, test.api, policyExtra),
				"src/main/java/" + test.impl + "/Threshold.java": fmt.Sprintf(`package %s;
import %s.Policy;
import %s.Policy.Decision;
public final class Threshold implements Policy {
 public Decision check(int value){return value>2?Decision.BLOCK:Decision.ALLOW;}
 %s
}`, test.impl, test.api, test.api, thresholdExtra),
				"src/main/java/" + test.app + "/Main.java": fmt.Sprintf(`package %s;
import %s.Policy;
import %s.Policy.Decision;
import %s.Threshold;
public final class Main {
 public static void main(String[] args){
  Policy normal=new Threshold();
  Policy anonymous=new Policy(){
   public Decision check(int value){return value==0?Decision.INDECISIVE:Decision.ALLOW;}
   %s
  };
  System.out.println(normal.check(3).name());
  System.out.println(anonymous.check(0).name());
  System.out.println(normal.check(1)==Policy.Decision.ALLOW);
  for(Decision decision:Decision.values()){System.out.println(decision.name()+":"+decision.ordinal());}
 }
}`, test.app, test.api, test.api, test.impl, anonymousExtra),
			}
			runInterfaceNestedEnumOracle(t, files, test.app+".Main")
		})
	}
}

func runInterfaceNestedEnumOracle(t *testing.T, files map[string]string, mainClass string) {
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
	if string(want) != "BLOCK\nINDECISIVE\ntrue\nALLOW:0\nINDECISIVE:1\nBLOCK:2\n" {
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
