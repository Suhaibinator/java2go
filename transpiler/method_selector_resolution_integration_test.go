package transpiler

import (
	"strings"
	"testing"
)

func TestMethodSelectorResolution_ParameterizedMethodUsesResolvedGoName(t *testing.T) {
	root := t.TempDir()
	writeJavaSource(t, root, "example/report/Ranker.java", `
package example.report;
import java.util.List;
public class Ranker<T> {
    public List<T> sort(List<T> values) { return values; }
}
`)
	writeJavaSource(t, root, "example/report/Report.java", `
package example.report;
import java.util.ArrayList;
import java.util.List;
public class Report {
    public List<String> render() {
        List<String> values = new ArrayList<String>();
        Ranker<String> ranker = new Ranker<String>();
        return ranker.sort(values);
    }
}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["example/report/Report.go"]
	if !strings.Contains(out, "ranker.SortJava2goExecution(__java2goExecution, values)") {
		t.Fatalf("expected a parameterized method call to use its resolved exported Go name, got:\n%s", out)
	}
	if strings.Contains(out, "ranker.sortJava2goExecution(") {
		t.Fatalf("parameterized method call retained Java casing:\n%s", out)
	}
}

func TestMethodSelectorResolution_CrossPackageGenericReceiverUsesResolvedGoName(t *testing.T) {
	root := t.TempDir()
	writeJavaSource(t, root, "example/rules/Rule.java", `
package example.rules;
public interface Rule<T> {
    boolean allow(T value);
}
`)
	writeJavaSource(t, root, "example/rules/TextRule.java", `
package example.rules;
public class TextRule implements Rule<String> {
    public boolean allow(String value) { return true; }
}
`)
	writeJavaSource(t, root, "example/engine/Engine.java", `
package example.engine;
import example.rules.Rule;
public class Engine<T> {
    public void addRule(Rule<T> rule) {}
}
`)
	writeJavaSource(t, root, "example/app/Application.java", `
package example.app;
import example.engine.Engine;
import example.rules.TextRule;
public class Application {
    public static void configure() {
        Engine<String> engine = new Engine<String>();
        engine.addRule(new TextRule());
    }
}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["example/app/Application.go"]
	if !strings.Contains(out, "engine.AddRuleJava2goExecution(__java2goExecution, rules.NewTextRuleJava2goExecution(__java2goExecution))") {
		t.Fatalf("expected a cross-package generic receiver call to use its resolved exported Go name, got:\n%s", out)
	}
	if strings.Contains(out, "engine.addRuleJava2goExecution(") {
		t.Fatalf("cross-package generic receiver call retained Java casing:\n%s", out)
	}
}

func TestMethodSelectorResolution_CrossPackageChildFindsInheritedConcreteMethod(t *testing.T) {
	root := t.TempDir()
	writeJavaSource(t, root, "example/base/Base.java", `
package example.base;
public class Base {
    public String inheritedLabel() { return "base"; }
}
`)
	writeJavaSource(t, root, "example/child/Child.java", `
package example.child;
import example.base.Base;
public class Child extends Base {}
`)
	writeJavaSource(t, root, "example/app/Application.java", `
package example.app;
import example.child.Child;
public class Application {
    public static String run() {
        Child child = new Child();
        return child.inheritedLabel();
    }
}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["example/app/Application.go"]
	if !strings.Contains(out, ".InheritedLabelJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected a child receiver to use the inherited method's resolved exported name, got:\n%s", out)
	}
	if strings.Contains(out, ".inheritedLabelJava2goExecution(") {
		t.Fatalf("cross-package superclass traversal retained Java casing:\n%s", out)
	}
}

func TestOverloadResolution_PrefersMostSpecificReferenceType(t *testing.T) {
	src := `
public class ReferenceSpecificityProgram {
    static class SpecificityParent {}
    static class SpecificityMid extends SpecificityParent {}
    static class SpecificityChild extends SpecificityMid {}

    public static String nearest(SpecificityParent value) { return "parent"; }
    public static String nearest(SpecificityMid value) { return "mid"; }

    public static String nullableHierarchy(SpecificityParent value) { return "parent"; }
    public static String nullableHierarchy(SpecificityMid value) { return "mid"; }

    public static String run() {
        SpecificityChild child = new SpecificityChild();
        return nearest(child) + "," + nullableHierarchy(null);
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestReferenceSpecificity(t *testing.T) {
    const want = "mid,mid"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}
