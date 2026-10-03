package transpiler

import (
	"strings"
	"testing"
)

func TestBoxedStringRoutesCanonicalParsers(t *testing.T) {
	for _, name := range []string{"Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double"} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, "public class BoxedStringRoute {static java.lang."+name+" factory(java.lang.String value){return java.lang."+name+".valueOf(value);}static java.lang."+name+" constructor(java.lang.String value){return new java.lang."+name+"(value);}}")
			parser := map[string]string{"Boolean": "JavaBooleanParseBoolean", "Byte": "JavaByteParseByte", "Short": "JavaShortParseShort", "Integer": "JavaIntegerParseInt", "Long": "JavaLongParseLong", "Float": "JavaFloatParseFloat", "Double": "JavaDoubleParseDouble"}[name]
			assertContains(t, out, "stdjava.Box"+name+"(stdjava."+parser+"(")
			assertContains(t, out, "stdjava.New"+name+"(stdjava."+parser+"(")
		})
	}
}

func TestBoxedStringReferenceRoutesCanonicalParsers(t *testing.T) {
	for _, name := range []string{"Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double"} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, "interface BoxedTextFactory {java.lang."+name+" make(java.lang.String value);}public class BoxedStringReferences {static void run(){BoxedTextFactory factory=java.lang."+name+"::valueOf;BoxedTextFactory constructor=java.lang."+name+"::new;}}")
			parser := map[string]string{"Boolean": "JavaBooleanParseBoolean", "Byte": "JavaByteParseByte", "Short": "JavaShortParseShort", "Integer": "JavaIntegerParseInt", "Long": "JavaLongParseLong", "Float": "JavaFloatParseFloat", "Double": "JavaDoubleParseDouble"}[name]
			assertContains(t, out, "stdjava.Box"+name+"(stdjava."+parser+"(")
			assertContains(t, out, "stdjava.New"+name+"(stdjava."+parser+"(")
		})
	}
}

func TestBoxedStringQualifiedParametersSurviveStringShadow(t *testing.T) {
	out := renderGoFileFromJava(t, `class String {} public class StringOwner {static java.lang.Integer read(java.lang.String value){return new java.lang.Integer(value);}static java.lang.Integer factory(){return java.lang.Integer.valueOf("12");} }`)
	assertContains(t, out, "stdjava.NewInteger(stdjava.JavaIntegerParseInt(value))")
	assertContains(t, out, "stdjava.BoxInteger(stdjava.JavaIntegerParseInt(")
}

func TestBoxedStringDeclarationGuards(t *testing.T) {
	for _, name := range []string{"Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double"} {
		t.Run(name, func(t *testing.T) {
			helper := setupParseHelper(t, "import foreign."+name+";class WrapperOwner {}")
			if _, builtin := builtinJavaWrapperPrimitive(name, helper.Ctx); builtin {
				t.Fatal("foreign import borrowed JDK wrapper")
			}
			if _, builtin := builtinJavaWrapperPrimitive("java.lang.nested."+name, helper.Ctx); builtin {
				t.Fatal("nested foreign spelling borrowed java.lang wrapper")
			}
			if _, builtin := builtinJavaWrapperPrimitive("java.lang."+name, helper.Ctx); !builtin {
				t.Fatal("exact JDK wrapper lost ownership")
			}
		})
	}
	out := renderGoFileFromJava(t, `class Integer {Integer(java.lang.String value){}static Integer valueOf(java.lang.String value){return new Integer(value);}}public class LocalWrapper {static Integer read(){return Integer.valueOf("source");}static <Integer> java.lang.Integer binder(Integer ignored){return java.lang.Integer.valueOf("7");}}`)
	if strings.Contains(out, "stdjava.JavaIntegerParseInt(stdjava.JavaStringLiteralUTF16([]uint16{115, 111, 117, 114, 99, 101})") {
		t.Fatal("source wrapper factory borrowed JDK parser")
	}
}

func TestBoxedStringLexicalBindersDeclineJDKOwnership(t *testing.T) {
	stringBinder := setupParseHelper(t, `public class StringBinder<String> {String value;}`)
	if isBuiltinJavaString("String", stringBinder.Ctx) {
		t.Fatal("String type parameter borrowed JDK String")
	}
	if !isBuiltinJavaString("java.lang.String", stringBinder.Ctx) {
		t.Fatal("canonical String disappeared under binder")
	}
	wrapperBinder := setupParseHelper(t, `public class WrapperBinder<Integer> {Integer value;}`)
	if _, builtin := builtinJavaWrapperPrimitive("Integer", wrapperBinder.Ctx); builtin {
		t.Fatal("Integer type parameter borrowed JDK wrapper")
	}
	if _, builtin := builtinJavaWrapperPrimitive("java.lang.Integer", wrapperBinder.Ctx); !builtin {
		t.Fatal("canonical Integer disappeared under binder")
	}
	out := renderGoFileFromJava(t, `public class ParserBinder<String> {Object reject(String value){return java.lang.Integer.valueOf(value);}}`)
	assertNotContains(t, out, "stdjava.JavaIntegerParseInt(value)")
}
