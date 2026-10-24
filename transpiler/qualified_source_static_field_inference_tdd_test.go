package transpiler

import (
	"encoding/json"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

// Complete Java sources are byte-exact pinned JDK21 pre-oracle inputs. The tests
// inspect the original field expressions, including their declaration identity.
const qualifiedStaticFieldPreoracleSourcesJSON = `{"boxed-null-and-JDK-name-shields":{"app/Main.java":"package app;\nimport shield.Integer;\nimport shield.String;\npublic class Main {public static void main(java.lang.String[] args) {\n System.out.println(\"boxed:\"+provider.Holder.boxed);System.out.println(provider.Holder.boxed+2);\n System.out.println(\"nulls:\"+provider.Holder.absent+\":\"+provider.Holder.nullable+\":\"+provider.Holder.nilObject);\n try {int value=provider.Holder.absent;System.out.println(\"unexpected:\"+value);} catch(NullPointerException ex){System.out.println(\"NPE\");}\n System.out.println(\"text:\"+provider.Holder.text+\":\"+provider.Holder.sourceString+\":\"+provider.Holder.sourceInteger);\n System.out.println(\"source-fields:\"+shield.Integer.MAX_VALUE+\":\"+shield.String.TOKEN+\":\"+shield.Character.MAX_VALUE+\":\"+shield.Arrays.trace);\n System.out.println(\"imported-source:\"+Integer.MAX_VALUE+\":\"+String.TOKEN);\n System.out.println(\"jdk:\"+java.lang.Integer.MAX_VALUE+\":\"+java.lang.Character.MAX_VALUE);\n System.out.println(provider.Holder.sourceInteger.value);\n}}\n","provider/Holder.java":"package provider;\npublic class Holder {\n public static java.lang.Integer boxed=17;public static java.lang.Integer absent;\n public static java.lang.String text=\"jdk-string\";public static java.lang.String nullable;\n public static java.lang.Object nilObject;public static shield.Integer sourceInteger=new shield.Integer();\n public static shield.String sourceString=new shield.String();\n}\n","shield/Arrays.java":"package shield;public class Arrays {public static int trace=314;}\n","shield/Character.java":"package shield;public class Character {public static char MAX_VALUE='Q';}\n","shield/Integer.java":"package shield;public class Integer {public static int MAX_VALUE=73;public int value=17;public java.lang.String toString(){return \"source-integer:\"+value;}}\n","shield/String.java":"package shield;public class String {public static char TOKEN='S';public java.lang.String toString(){return \"source-string\";}}\n"},"inherited-static-initialization-and-hiding":{"app/Main.java":"package app;\nimport consumer.Child;\npublic class Main {public static void main(java.lang.String[] args) {\n System.out.println(\"start:\"+provider.Init.trace);\n System.out.println(\"inherited:\"+consumer.Child.letter);\n System.out.println(\"after-parent:\"+provider.Init.trace);\n System.out.println(\"hidden:\"+consumer.Child.number);\n System.out.println(\"after-child:\"+provider.Init.trace);\n System.out.println(\"declared:\"+provider.Parent.number);\n System.out.println(\"simple:\"+Child.letter+\":\"+Child.number);\n System.out.println(consumer.Child.number+provider.Parent.number);\n}}\n","consumer/Child.java":"package consumer;\npublic class Child extends provider.Parent { public static long number=4294967297L;static {provider.Init.trace=provider.Init.trace*10+2;} }\n","provider/Init.java":"package provider;public class Init {public static int trace;}\n","provider/Parent.java":"package provider;\npublic class Parent { public static char letter='I';public static int number=91;static {Init.trace=Init.trace*10+1;} }\n"},"negative-class-binder-prefix":{"T/Fields.java":"package T;public class Fields {public static int value=73;}\n","app/Main.java":"package app;public class Main<T>{int wrong(){return T.Fields.value;}public static void main(java.lang.String[] args){System.out.println(new Main<java.lang.Object>().wrong());}}\n"},"negative-lexical-missing-member":{"app/Main.java":"package app;public class Main {static class q {} public static void main(java.lang.String[] args){System.out.println(q.Missing.value);}}\n","q/Missing.java":"package q;public class Missing {public static int value=73;}\n"},"negative-method-binder-prefix":{"T/Fields.java":"package T;public class Fields {public static int value=73;}\n","app/Main.java":"package app;public class Main {static <T> int wrong(){return T.Fields.value;}public static void main(java.lang.String[] args){System.out.println(wrong());}}\n"},"primitive-fields-and-value-prefixes":{"app/Main.java":"package app;\nimport provider.Fields;\npublic class Main {\n static void parameter(provider.ValueHolder provider) {System.out.println(\"param:\"+provider.Fields.value);}\n static void blocks() {\n  System.out.println(\"before:\"+provider.Fields.i);\n  {provider.ValueHolder provider=new provider.ValueHolder();System.out.println(\"local:\"+provider.Fields.value);}\n  System.out.println(\"after:\"+provider.Fields.i);\n }\n public static void main(java.lang.String[] args) {\n  System.out.println(provider.Fields.b);System.out.println(provider.Fields.s);System.out.println(provider.Fields.c);\n  System.out.println(provider.Fields.i);System.out.println(provider.Fields.l);System.out.println(provider.Fields.f);\n  System.out.println(provider.Fields.d);System.out.println(provider.Fields.z);\n  System.out.println(\"all:\"+provider.Fields.b+\":\"+provider.Fields.s+\":\"+provider.Fields.c+\":\"+provider.Fields.i+\":\"+provider.Fields.l+\":\"+provider.Fields.f+\":\"+provider.Fields.d+\":\"+provider.Fields.z);\n  System.out.println(provider.Fields.b+provider.Fields.s+provider.Fields.c);\n  System.out.println(provider.Fields.i+provider.Fields.l);System.out.println(provider.Fields.f+provider.Fields.d);\n  System.out.println(provider.Fields.z & true);System.out.println(provider.Fields.bytes[0]+\":\"+provider.Fields.bytes.length);\n  System.out.println(\"nested:\"+provider.Fields.Nested.c+\":\"+provider.Fields.Nested.i);\n  System.out.println(\"imported:\"+Fields.i);parameter(new provider.ValueHolder());blocks();\n }\n}\n","provider/Fields.java":"package provider;\npublic class Fields {\n public static byte b=-7; public static short s=-300; public static char c='Z';\n public static int i=37; public static long l=4294967297L;\n public static float f=1.25f; public static double d=2.5; public static boolean z=true;\n public static byte[] bytes=new byte[]{-128,7};\n public static class Nested { public static char c='N'; public static int i=23; }\n}\n","provider/ValueHolder.java":"package provider;\npublic class ValueHolder {\n public ValueFields Fields=new ValueFields();\n public static class ValueFields { public long value=8589934593L; }\n}\n"},"static-imported-value-prefix":{"app/Main.java":"package app;\nimport static fixture.Values.named;\nimport types.Named;\npublic class Main {public static void main(java.lang.String[] args){int seed=java.lang.Integer.parseInt(args[0]);\n fixture.Values.named=fixture.Values.make(seed);\n System.out.println(\"import-value:\"+named.Fields.number+\":\"+named.Fields.letter);\n System.out.println(\"import-type:\"+Named.Nested.number);\n System.out.println(\"effects:\"+fixture.Values.effects);\n fixture.Values.named=null;\n try {System.out.println(named.Fields.number);System.out.println(\"unexpected-null\");} catch(NullPointerException ex){System.out.println(\"NPE\");}\n}}\n","fixture/Values.java":"package fixture;public class Values {\n public static Root named;public static int effects;\n public static class Root {public Fields Fields;public Root(int seed){Fields=new Fields(seed);}}\n public static class Fields {public long number;public char letter;public Fields(int seed){number=8589934593L+seed;letter=(char)('K'+seed%8);}}\n public static Root make(int seed){effects++;return new Root(seed);}\n}\n","named/Fields.java":"package named;public class Fields {public static int number=73;public static char letter='P';}\n","types/Named.java":"package types;public class Named {public static class Nested {public static long number=4294967299L;}}\n"}}`

func qualifiedStaticFieldSourceTypes(t *testing.T, files map[string]string, methodName string, wants map[string]string) {
	t.Helper()
	ctx := sourceGenericViewDemandTestContext(t, files)
	main := findQualifiedSourceClass("app.Main")
	if main == nil {
		t.Fatal("missing original app.Main declaration")
	}
	ctx = classScopeCtx(main, ctx)
	methods := main.FindMethod().ByOriginalName(methodName)
	if len(methods) != 1 || methods[0].DeclarationNode == nil {
		t.Fatal("missing original method")
	}
	ctx.localScope = methods[0]
	source := []byte(files["app/Main.java"])
	seen := map[string]int{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "field_access" {
			text := node.Content(source)
			if want, selected := wants[text]; selected {
				seen[text]++
				var origin inferredJavaTypeOrigin
				got, known := inferExprJavaType(node, ctx, source, &origin)
				if !known || got != want {
					t.Errorf("QUALIFIED_SOURCE_FIELD_TYPE %s: %q/%t want %q/true", text, got, known, want)
				}
				if known && origin.javaType != want {
					t.Errorf("QUALIFIED_SOURCE_FIELD_ORIGIN %s: %q want %q", text, origin.javaType, want)
				}
				object := node.ChildByFieldName("object")
				owner := resolveClassScopeByIdentifier(ctx, source, object)
				if target := resolveInvocationTarget(object, ctx, source); target != nil && target.classScope != nil {
					owner = target.classScope
				}
				if owner != nil {
					field := findFieldResolutionInHierarchy(owner, node.ChildByFieldName("field").Content(source), ctx)
					if field == nil || field.owner == nil || field.def.OriginalType != want {
						t.Errorf("QUALIFIED_SOURCE_FIELD_DECLARATION %s lost exact declaring field/type", text)
					}
				} else if text != "named.Fields.number" && text != "named.Fields.letter" {
					t.Errorf("QUALIFIED_SOURCE_FIELD_OWNER %s lost source owner", text)
				}
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(methods[0].DeclarationNode)
	for text := range wants {
		if seen[text] == 0 {
			t.Errorf("original field expression %s missing", text)
		}
	}
}

func TestQualifiedSourceStaticFieldOriginalArraysTypes(t *testing.T) {
	var projects []struct {
		Name  string            `json:"name"`
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal([]byte(arraysUtilityProgramsJSON), &projects); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, project := range projects {
		var wants map[string]string
		switch project.Name {
		case "AdversarialOriginal":
			wants = map[string]string{"lookalike.Arrays.calls": "int", "lookalike.Arrays.trace": "int"}
		case "foreignSourceBinders":
			wants = map[string]string{"foreign.Arrays.trace": "int"}
		default:
			continue
		}
		found++
		t.Run(project.Name, func(t *testing.T) {
			files := map[string]string{}
			for path, source := range project.Files {
				const prefix = "src/main/java/"
				if len(path) > len(prefix) && path[:len(prefix)] == prefix {
					files[path[len(prefix):]] = source
				}
			}
			qualifiedStaticFieldSourceTypes(t, files, "main", wants)
		})
	}
	if found != 2 {
		t.Fatal("original full Arrays projects missing")
	}
}

func TestQualifiedSourceStaticFieldPreoracleDeclarationTypes(t *testing.T) {
	var projects map[string]map[string]string
	if err := json.Unmarshal([]byte(qualifiedStaticFieldPreoracleSourcesJSON), &projects); err != nil {
		t.Fatal(err)
	}
	for _, project := range []struct {
		name  string
		wants map[string]string
	}{
		{"primitive-fields-and-value-prefixes", map[string]string{"provider.Fields.b": "byte", "provider.Fields.s": "short", "provider.Fields.c": "char", "provider.Fields.i": "int", "provider.Fields.l": "long", "provider.Fields.f": "float", "provider.Fields.d": "double", "provider.Fields.z": "boolean", "provider.Fields.bytes": "byte[]", "provider.Fields.Nested.c": "char", "provider.Fields.Nested.i": "int", "Fields.i": "int"}},
		{"inherited-static-initialization-and-hiding", map[string]string{"provider.Init.trace": "int", "consumer.Child.letter": "char", "consumer.Child.number": "long", "provider.Parent.number": "int", "Child.letter": "char", "Child.number": "long"}},
		{"boxed-null-and-JDK-name-shields", map[string]string{"provider.Holder.boxed": "java.lang.Integer", "provider.Holder.absent": "java.lang.Integer", "provider.Holder.nullable": "java.lang.String", "provider.Holder.nilObject": "java.lang.Object", "provider.Holder.text": "java.lang.String", "provider.Holder.sourceString": "shield.String", "provider.Holder.sourceInteger": "shield.Integer", "shield.Integer.MAX_VALUE": "int", "shield.String.TOKEN": "char", "shield.Character.MAX_VALUE": "char", "shield.Arrays.trace": "int"}},
		{"static-imported-value-prefix", map[string]string{"named.Fields.number": "long", "named.Fields.letter": "char", "Named.Nested.number": "long", "fixture.Values.effects": "int"}},
	} {
		t.Run(project.name, func(t *testing.T) { qualifiedStaticFieldSourceTypes(t, projects[project.name], "main", project.wants) })
	}
}

func TestQualifiedSourceStaticFieldInvalidOwnerShields(t *testing.T) {
	var projects map[string]map[string]string
	if err := json.Unmarshal([]byte(qualifiedStaticFieldPreoracleSourcesJSON), &projects); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"negative-lexical-missing-member", "negative-method-binder-prefix", "negative-class-binder-prefix"} {
		t.Run(name, func(t *testing.T) {
			files := projects[name]
			ctx := sourceGenericViewDemandTestContext(t, files)
			main := findQualifiedSourceClass("app.Main")
			ctx = classScopeCtx(main, ctx)
			methodName := "wrong"
			if name == "negative-lexical-missing-member" {
				methodName = "main"
			}
			methods := main.FindMethod().ByOriginalName(methodName)
			if len(methods) != 1 {
				t.Fatal("original invalid method missing")
			}
			ctx.localScope = methods[0]
			source := []byte(files["app/Main.java"])
			field := findNode(methods[0].DeclarationNode, "field_access")
			if field == nil {
				t.Fatal("original invalid field missing")
			}
			owner := resolveClassScopeByIdentifier(ctx, source, field.ChildByFieldName("object"))
			if owner != nil {
				t.Errorf("invalid lexical/type-binder prefix rebound to package owner %s", owner.Class.OriginalName)
			}
		})
	}
}
