package transpiler

import "testing"

func TestMapEntrySourceApplicabilityNominalControls(t *testing.T) {
	helper := setupParseHelper(t, `
import java.util.Map;
class Direct implements Map.Entry<String,String> {}
class Generic<K,V> implements Map.Entry<K,V> {}
class Child extends Generic<String,Integer> {}
class Shaped { String getKey() { return ""; } String getValue() { return ""; } String setValue(String s) { return s; } }
interface Entry<K,V> {}
class Shadow implements Entry<String,String> {}
`)
	for _, test := range []struct {
		actual, expected string
		want             bool
	}{
		{"Direct", "java.util.Map.Entry<String,String>", true},
		{"Direct", "java.util.Map.Entry<String,Object>", false},
		{"Direct", "java.util.Map.Entry", true},
		{"Generic<String,Integer>", "java.util.Map.Entry<String,Integer>", true},
		{"Child", "java.util.Map.Entry<String,Integer>", true},
		{"Child", "java.util.Map.Entry<String,String>", false},
		{"Generic", "java.util.Map.Entry<String,Integer>", true},
		{"Shaped", "java.util.Map.Entry<String,String>", false},
		{"Shadow", "java.util.Map.Entry<String,String>", false},
		{"Direct", "Entry<String,String>", false},
		{"Direct[]", "java.util.Map.Entry<String,String>", false},
	} {
		if got := javaInferenceTypeAssignable(test.actual, test.expected, helper.Ctx); got != test.want {
			t.Errorf("assignable(%s, %s) = %v, want %v", test.actual, test.expected, got, test.want)
		}
	}
}
