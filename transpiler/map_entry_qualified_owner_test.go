package transpiler

import "testing"

func TestMapEntryQualifiedExternalOwnerControls(t *testing.T) {
	t.Run("source outer retains its member", func(t *testing.T) {
		helper := setupParseHelper(t, `
interface Map { interface Entry<K,V> {} }
class Local implements Map.Entry<String,String> {}
class Canonical implements java.util.Map.Entry<String,String> {}
`)
		for _, test := range []struct {
			actual, expected string
			want             bool
		}{
			{"Local", "Map.Entry<String,String>", true},
			{"Local", "java.util.Map.Entry<String,String>", false},
			{"Canonical", "java.util.Map.Entry<String,String>", true},
			{"Canonical", "Map.Entry<String,String>", false},
		} {
			if got := javaInferenceTypeAssignable(test.actual, test.expected, helper.Ctx); got != test.want {
				t.Errorf("assignable(%s, %s) = %v, want %v", test.actual, test.expected, got, test.want)
			}
		}
	})
	t.Run("single member import is canonical", func(t *testing.T) {
		helper := setupParseHelper(t, `
import java.util.Map.Entry;
class Direct implements Entry<String,String> {}
interface Map { interface Entry<K,V> {} }
`)
		if !javaInferenceTypeAssignable("Direct", "Entry<String,String>", helper.Ctx) {
			t.Error("Imported canonical Entry lost its declared source implementation")
		}
		if javaInferenceTypeAssignable("Direct", "Map.Entry<String,String>", helper.Ctx) {
			t.Error("Imported canonical Entry borrowed the unrelated source Map.Entry")
		}
	})
	t.Run("general registered external outer", func(t *testing.T) {
		helper := setupParseHelper(t, `
import java.lang.Thread;
class State {}
class Probe {}
`)
		if got := resolveClassScopeByQualifiedName(helper.Ctx, "Thread.State"); got != nil {
			t.Error("External Thread.State borrowed the unrelated source State")
		}
	})
}
