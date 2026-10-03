package transpiler

import (
	"context"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Java bytes and expected streams come from sealed, actually observed JDK21
// author cases. Rejected author predictions remain in their frozen archives.

func TestOrdinaryDemandOwnerIgnoresInheritedMembersSelectsOtherPackageJVM(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml": `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>ordinary-ambiguity-control</artifactId><version>1.0</version><properties><maven.compiler.release>21</maven.compiler.release></properties></project>
`,
		"src/main/java/app/Main.java": `package app;
import p.Owner.*;
import q.*;
public final class Main {
    public X selected;
    public static void main(java.lang.String[] args) {
        int seed = java.lang.Integer.parseInt(args[0]);
        Main workflow = new Main();
        workflow.selected = new X();
        X alias = workflow.selected;
        java.util.LinkedHashMap<java.lang.String, X> values = new java.util.LinkedHashMap<java.lang.String, X>();
        values.put("chosen" + seed, workflow.selected);
        values.put("alias" + seed, alias);
        X displaced = values.put("chosen" + seed, new X());
        X removed = values.remove("alias" + seed);
        p.Left.X left = new p.Left.X(seed);
        p.Right.X right = new p.Right.X(seed);
        int leftState = left.step(seed % 5 + 1);
        int rightState = right.step(seed % 5 + 1);
        System.out.println("selection:" + X.class.getName() + ":" + (X.class == q.X.class) + ":" + (workflow.selected.getClass() == q.X.class));
        System.out.println("identity:" + (workflow.selected == alias) + ":" + (displaced == alias) + ":" + (removed == alias) + ":" + (values.get("chosen" + seed) != alias));
        System.out.println("state:" + seed + ":" + values.size() + ":" + leftState + ":" + rightState + ":" + workflow.selected.state());
    }
}
`,
		"src/main/java/p/Left.java": `package p;
public interface Left {
    public static class X {
        private int state;
        public X(int seed) { state = seed; }
        public int step(int delta) { state += delta; return state; }
        public int state() { return state; }
    }
}
`,
		"src/main/java/p/Owner.java": `package p;
public final class Owner implements Left, Right { }
`,
		"src/main/java/p/Right.java": `package p;
public interface Right {
    public static class X {
        private int state;
        public X(int seed) { state = seed; }
        public int step(int delta) { state += delta * 3; return state; }
        public int state() { return state; }
    }
}
`,
		"src/main/java/q/X.java": `package q;
public final class X {
    public int state() { return 99; }
}
`,
	}, "app.Main", []campaignCompilerProjectObservation{
		{name: "same-seed-17-repeat-1", args: []string{"17"}, stdout: "selection:q.X:true:true\nidentity:true:true:true:true\nstate:17:1:20:26:99\n", stderr: ""},
		{name: "same-seed-17-repeat-2", args: []string{"17"}, stdout: "selection:q.X:true:true\nidentity:true:true:true:true\nstate:17:1:20:26:99\n", stderr: ""},
		{name: "same-seed-17-repeat-3", args: []string{"17"}, stdout: "selection:q.X:true:true\nidentity:true:true:true:true\nstate:17:1:20:26:99\n", stderr: ""},
	})
}

func TestOrdinaryDemandOwnerSelectsDirectDeclaredMemberJVM(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml": `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>ordinary-ambiguity-control</artifactId><version>1.0</version><properties><maven.compiler.release>21</maven.compiler.release></properties></project>
`,
		"src/main/java/app/Main.java": `package app;
import p.Owner.*;
import q.*;
public final class Main {
    public static void main(java.lang.String[] args) {
        int seed = java.lang.Integer.parseInt(args[0]);
        Other control = new Other(seed);
        X value = new X(seed);
        X alias = value;
        int first = alias.step(control.delta());
        p.Owner.X qualified = value;
        int second = qualified.step(control.delta() + 1);
        p.Left.X left = new p.Left.X(seed);
        p.Right.X right = new p.Right.X(seed);
        int leftState = left.step(control.delta());
        int rightState = right.step(control.delta());
        System.out.println("state:" + first + ":" + second + ":" + value.state() + ":" + leftState + ":" + rightState);
        System.out.println("identity:" + (value == alias) + ":" + (value == qualified));
        System.out.println("type:" + X.class.getName() + ":" + (X.class == p.Owner.X.class) + ":" + (value.getClass() == p.Owner.X.class));
    }
}
`,
		"src/main/java/p/Left.java": `package p;
public interface Left {
    public static class X {
        private int state;
        public X(int seed) { state = seed; }
        public int step(int delta) { state += delta; return state; }
        public int state() { return state; }
    }
}
`,
		"src/main/java/p/Owner.java": `package p;
public final class Owner implements Left, Right {
    public static final class X {
        private int state;
        public X(int seed) { state = seed; }
        public int step(int delta) { state += delta * 2; return state; }
        public int state() { return state; }
    }
}
`,
		"src/main/java/p/Right.java": `package p;
public interface Right {
    public static class X {
        private int state;
        public X(int seed) { state = seed; }
        public int step(int delta) { state += delta * 3; return state; }
        public int state() { return state; }
    }
}
`,
		"src/main/java/q/Other.java": `package q;
public final class Other {
    private final int delta;
    public Other(int seed) { delta = seed % 5 + 1; }
    public int delta() { return delta; }
}
`,
	}, "app.Main", []campaignCompilerProjectObservation{
		{name: "same-seed-17-repeat-1", args: []string{"17"}, stdout: "state:23:31:31:20:26\nidentity:true:true\ntype:p.Owner$X:true:true\n", stderr: ""},
		{name: "same-seed-17-repeat-2", args: []string{"17"}, stdout: "state:23:31:31:20:26\nidentity:true:true\ntype:p.Owner$X:true:true\n", stderr: ""},
		{name: "same-seed-17-repeat-3", args: []string{"17"}, stdout: "state:23:31:31:20:26\nidentity:true:true\ntype:p.Owner$X:true:true\n", stderr: ""},
	})
}

func TestOrdinaryDemandInheritedOnlyMemberIsAbsentJavacControl(t *testing.T) {
	sources := map[string]string{
		"src/main/java/app/Main.java": `package app;
import p.Owner.*;
import q.*;
public final class Main {
    public static void main(java.lang.String[] args) {
        int seed = java.lang.Integer.parseInt(args[0]);
        Other control = new Other(seed);
        X value = new X(seed);
        X alias = value;
        int first = alias.step(control.delta());
        p.Left.X qualified = value;
        int second = qualified.step(control.delta() + 1);
        p.Left.X separate = new p.Left.X(seed + 10);
        int independent = separate.step(2);
        System.out.println("state:" + first + ":" + second + ":" + value.state() + ":" + independent);
        System.out.println("identity:" + (value == alias) + ":" + (value == qualified) + ":" + (value != separate));
        System.out.println("type:" + X.class.getName() + ":" + (X.class == p.Left.X.class) + ":" + (value.getClass() == p.Left.X.class));
    }
}
`,
		"src/main/java/p/Left.java": `package p;
public interface Left {
    public static class X {
        private int state;
        public X(int seed) { state = seed; }
        public int step(int delta) { state += delta; return state; }
        public int state() { return state; }
    }
}
`,
		"src/main/java/p/MiddleLeft.java": `package p;
public interface MiddleLeft extends Left { }
`,
		"src/main/java/p/MiddleRight.java": `package p;
public interface MiddleRight extends Left { }
`,
		"src/main/java/p/Owner.java": `package p;
public final class Owner implements MiddleLeft, MiddleRight { }
`,
		"src/main/java/q/Other.java": `package q;
public final class Other {
    private final int delta;
    public Other(int seed) { delta = seed % 5 + 1; }
    public int delta() { return delta; }
}
`,
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	paths := make([]string, 0, len(sources))
	for name, source := range sources {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, javac, append([]string{"-XDrawDiagnostics", "--release", "21", "-d", filepath.Join(root, "classes")}, paths...)...).CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), "compiler.err.cant.resolve.location") {
		t.Fatalf("fresh javac did not confirm missing imported type X: %v\n%s", err, output)
	}
	t.Log("fresh javac confirmed inherited-only ordinary imported type X absent")
	lookup := ordinaryDemandTestContext(t, sources, "src/main/java/app/Main.java")
	if scope := resolveClassScopeByQualifiedName(lookup, "X"); scope != nil {
		t.Fatalf("ordinary Owner.* contributed inherited type %s", qualifiedSourceClassName(scope))
	}
	if len(collectedDiagnostics()) != 0 {
		t.Fatalf("missing ordinary type must remain absent, not ambiguous: %v", collectedDiagnostics())
	}
}
