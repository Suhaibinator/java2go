package transpiler

import (
	"strings"
	"testing"
)

func TestSourceObjectEqualsOverloadRuntimeEntry(t *testing.T) {
	out := renderGoFileFromJava(t, `public class EqualityEntryProbe {
 static class Key {
  public boolean equals(java.lang.Object other){return other==this;}
  public boolean equals(Key other){throw new IllegalStateException("typed");}
  public boolean equalsJava2goExecution(int ignored){throw new IllegalStateException("foreign");}
 }
 static class FalseKey extends Key { public boolean equals(java.lang.Object other){return false;} }
}`)
	if !strings.Contains(out, "EqualsJava2goExecution1(") || !strings.Contains(out, "other any) bool") || !strings.Contains(out, "Java2goInheritedOverload_") {
		t.Fatalf("missing separately named equality entry and retained overloads:\n%s", out)
	}
}

func TestSourceObjectEqualsShadowParameterIsNotRuntimeOverride(t *testing.T) {
	out := renderGoFileFromJava(t, `public class EqualityShadowProbe {
 static class Object {}
 static class Key {
  public boolean equals(Object other){return true;}
  public boolean equals(Key other){return false;}
 }
 static class Child extends Key { public boolean equals(Object other){return false;} }
}`)
	if strings.Contains(out, "other any) bool") {
		t.Fatalf("source Object shadow acquired runtime equality entry:\n%s", out)
	}
}

func TestSourceObjectEqualsGenericBinderLocals(t *testing.T) {
	out := renderGoFileFromJava(t, `public class EqualityBinderProbe {
  static class Key<other> {
   public boolean equals(java.lang.Object target) { return target==this; }
   public boolean equals(Key<other> target) { return false; }
  }
  static class Child<other> extends Key<other> {
   public boolean equals(java.lang.Object target) { return target==this; }
  }
 }`)
	if strings.Count(out, "other_0 any) bool") != 2 {
		t.Fatalf("Object equality bridge argument collides with emitted class binder:\n%s", out)
	}
	if !strings.Contains(out, ", other_0)") {
		t.Fatalf("Object equality bridge did not forward its fresh argument:\n%s", out)
	}
}
