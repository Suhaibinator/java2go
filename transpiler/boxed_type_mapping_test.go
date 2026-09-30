package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestBoxedTypeMapping_ReferenceABI(t *testing.T) {
	helper := setupParseHelper(t, "class WrapperTypes {}")
	cases := map[string]string{
		"Boolean": "*stdjava.Boolean", "Byte": "*stdjava.Byte",
		"Short": "*stdjava.Short", "Character": "*stdjava.Character",
		"Integer": "*stdjava.Integer", "Long": "*stdjava.Long",
		"Float": "*stdjava.Float", "Double": "*stdjava.Double",
		"java.lang.Integer": "*stdjava.Integer", "Number": "stdjava.JavaNumber",
		"Comparable<Integer>": "any", "Comparable": "any",
		"int": "int32", "long": "int64", "char": "rune", "boolean": "bool",
	}
	for javaType, want := range cases {
		t.Run(javaType, func(t *testing.T) {
			var got bytes.Buffer
			if err := printer.Fprint(&got, token.NewFileSet(), javaTypeStringToGoTypeExpr(javaType, nil, helper.Ctx)); err != nil {
				t.Fatal(err)
			}
			if got.String() != want {
				t.Fatalf("mapped type = %s, want %s", got.String(), want)
			}
		})
	}
}

func TestBoxedTypeMapping_SourceDeclarationShadowsImplicitImport(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
class Integer {}
class Number {}
interface Comparable<T> {}
class ShadowedWrappers {
    Integer source;
    java.lang.Integer builtin;
    Number sourceNumber;
    java.lang.Number builtinNumber;
    Comparable<Integer> sourceComparable;
    java.lang.Comparable<java.lang.Integer> builtinComparable;
}`))
	for _, want := range []string{
		"source *integer", "builtin *stdjava.Integer", "sourceNumber *number",
		"builtinNumber stdjava.JavaNumber", "sourceComparable comparable[*integer]", "builtinComparable any",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in generated declarations:\n%s", want, out)
		}
	}
}

func TestBoxedTypeInference_NominalHierarchy(t *testing.T) {
	helper := setupParseHelper(t, "class WrapperInference {}")
	for _, test := range []struct {
		actual, expected string
		assignable       bool
	}{
		{"Integer", "Number", true}, {"Double", "Number", true},
		{"Character", "Number", false}, {"Boolean", "Number", false},
		{"Integer", "Object", true}, {"Number", "Object", true},
		{"Integer", "Comparable<Integer>", true}, {"Integer", "Comparable<Long>", false},
		{"Integer", "Comparable<? extends Number>", true}, {"Integer", "Comparable<? super Integer>", true},
		{"String", "Comparable<String>", true}, {"String", "Comparable<Integer>", false},
		{"Integer[]", "Number[]", true}, {"Integer[]", "int[]", false},
	} {
		if got := javaInferenceTypeAssignable(test.actual, test.expected, helper.Ctx); got != test.assignable {
			t.Errorf("assignable(%s, %s) = %v, want %v", test.actual, test.expected, got, test.assignable)
		}
	}
	if got := javaInferenceLeastUpperBound([]string{"Integer", "Long", "Double"}, helper.Ctx); got != "java.lang.Number" {
		t.Errorf("numeric wrapper LUB = %q, want java.lang.Number", got)
	}
	if got := javaInferenceLeastUpperBound([]string{"Integer", "Character"}, helper.Ctx); got != "Object" {
		t.Errorf("numeric and character LUB = %q, want Object", got)
	}
}

func TestBoxedTypeInference_PrimitiveInvocationUsesWrapperTypeArgument(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
class BoxedInference {
    static <T> T identity(T value) { return value; }
    static Integer run() { return identity(17); }
}`))
	if !strings.Contains(out, "identityJava2goExecution[*stdjava.Integer]") || !strings.Contains(out, "stdjava.BoxInteger(") {
		t.Fatalf("generic invocation must infer and construct an Integer object:\n%s", out)
	}
}

func TestBoxedTypeInference_IdentityNullAndNumberBounds(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class BoxedGenericObjects {
    static <T> T identity(T value) { return value; }
    static <T extends Number> T choose(boolean first, T left, T right) {
        return first ? left : right;
    }
    static <N extends Number> Number mixedBound(N value) {
        return choose(false, 1, value);
    }
    public static int run() {
        var inferred = identity(17);
        Integer exact = inferred;
        Integer missing = identity(null);
        Integer integer = Integer.valueOf(3);
        Long longer = Long.valueOf(4);
        Number number = choose(false, integer, longer);
        int score = inferred == exact ? 1 : 0;
        if (identity(integer) == integer) score += 2;
        if (missing == null) score += 4;
        if (number == longer) score += 8;
        if (mixedBound(longer) == longer) score += 16;
        score += identity(5);
        return score;
    }
}
`, 36)
}

func TestBoxedTypeMapping_QualifiedWrapperBesideSourceInteger(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
class Integer {
    int value;
    Integer(int value) { this.value = value; }
}
public class ShadowedIntegerObjects {
    Integer source;
    java.lang.Integer builtin;
    static <T> T identity(T value) { return value; }
    public static int run() {
        ShadowedIntegerObjects holder = new ShadowedIntegerObjects();
        holder.source = new Integer(3);
        holder.builtin = identity(4);
        return holder.source.value + holder.builtin;
    }
}
`, 7)
}

func TestBoxedTypeInference_InstanceOwnerAndMethodParameters(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class BoxedInstanceParameters {
    static class Holder<U> {
        U value;
        void set(U value) { this.value = value; }
        <T> T pair(U owner, T value) { this.value = owner; return value; }
        <T> T identity(T value) { return value; }
    }
    static class IntegerHolder extends Holder<Integer> {}
    static <T> int call(Holder<T> holder, T owner) {
        Integer boxed = holder.pair(owner, 7);
        return boxed;
    }
    public static int run() {
        Holder<Integer> holder = new Holder<>();
        holder.set(3 * 3);
        Integer result = holder.identity(-1);
        int mixed = call(holder, holder.value);
        IntegerHolder inherited = new IntegerHolder();
        inherited.set(4);
        return holder.value + result + mixed + inherited.value;
    }
}
`, 19)
}
