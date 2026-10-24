package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// This Java source is byte-identical to the original mixed synthetic/enum test.
// Its original native-string Go harness remains untouched.
const frozenSyntheticPairSource = `
public class SyntheticArrayIdentityProgram {
    interface Pair {
        int first();
        int second();
    }

    static Pair pair(int value) {
        return new Pair() {
            public int first() { return value; }
            public int second() { return value + 1; }
        };
    }

    interface Super {
        int base();
        int marker();
    }

    interface Sub extends Super {
        int child();
    }

    static class Parent implements Sub {
        int value;
        Parent(int value) { this.value = value; }
        public int base() { return value; }
        public int marker() { return value + 10; }
        public int child() { return value + 100; }
    }

    static class Inherited extends Parent {
        Inherited(int value) { super(value); }
    }

    static class SuperOnly implements Super {
        int value;
        SuperOnly(int value) { this.value = value; }
        public int base() { return value; }
        public int marker() { return value + 20; }
    }

    enum Color { RED, GREEN, BLUE }

    public static String run() {
        Pair[] pairs = new Pair[] { pair(3), pair(5) };
        int anonymousScore = pairs[0].first() * 1000
            + pairs[0].second() * 100
            + pairs[1].first() * 10
            + pairs[1].second();
        Object[] pairObjects = pairs;
        int anonymousRejected = 0;
        try {
            pairObjects[0] = "bad";
        } catch (ArrayStoreException expected) {
            anonymousRejected = 1;
        }

        Sub[] actual = new Sub[] { new Parent(2), new Inherited(4) };
        Super[] superView = actual;
        int hierarchyScore = superView[0].base() + superView[0].marker()
            + superView[1].base() + superView[1].marker()
            + actual[1].child();
        int hierarchyRejected = 0;
        try {
            superView[0] = new SuperOnly(8);
        } catch (ArrayStoreException expected) {
            hierarchyRejected = 1;
        }

        Color[] colors = new Color[] { Color.RED, Color.GREEN };
        Object[] colorView = colors;
        Color recovered = (Color) colorView[1];
        Object[] objects = new Object[] { Color.BLUE };
        Color fromObject = (Color) objects[0];
        int enumScore = recovered.ordinal() + fromObject.ordinal();
        if (recovered == Color.GREEN && fromObject == Color.BLUE) enumScore += 100;
        try {
            colorView[0] = "bad";
        } catch (ArrayStoreException expected) {
            enumScore += 10;
        }

        return anonymousScore + ":" + anonymousRejected
            + ":" + hierarchyScore + ":" + hierarchyRejected
            + ":" + enumScore;
    }
}
`

func TestSyntheticNestedInterfaceQualifiedEmbeds(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"original_op", referenceArraySyntheticImplementorSource},
		{"original_pair", frozenSyntheticPairSource},
	} {
		t.Run(tc.name, func(t *testing.T) {
			generated := renderGoFileFromJava(t, tc.source)
			file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
			if err != nil {
				t.Fatal(err)
			}
			declared := map[string]bool{}
			ast.Inspect(file, func(node ast.Node) bool {
				if spec, ok := node.(*ast.TypeSpec); ok {
					declared[spec.Name.Name] = true
				}
				return true
			})
			ast.Inspect(file, func(node ast.Node) bool {
				structure, ok := node.(*ast.StructType)
				if !ok {
					return true
				}
				for _, field := range structure.Fields.List {
					if len(field.Names) != 0 {
						continue
					}
					embedded := field.Type
					for {
						switch typ := embedded.(type) {
						case *ast.StarExpr:
							embedded = typ.X
						case *ast.IndexExpr:
							embedded = typ.X
						case *ast.IndexListExpr:
							embedded = typ.X
						default:
							goto resolved
						}
					}
				resolved:
					if identifier, ok := embedded.(*ast.Ident); ok && !declared[identifier.Name] {
						t.Errorf("hoisted embed %s does not denote a generated declaration", identifier.Name)
					}
				}
				return true
			})
		})
	}
}

func TestSyntheticNestedInterfaceOriginalProjectsJVM(t *testing.T) {
	t.Run("op_pair", func(t *testing.T) {
		runCampaignCompilerProjectOracle(t, map[string]string{
			"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>synthetic</groupId><artifactId>nested</artifactId><version>1</version></project>`,
			"src/main/java/ReferenceArraySyntheticImplementors.java": referenceArraySyntheticImplementorSource,
			"src/main/java/SyntheticArrayIdentityProgram.java":       frozenSyntheticPairSource,
			"src/main/java/Main.java":                                `public class Main {public static void main(String[] args){System.out.println(ReferenceArraySyntheticImplementors.lambdaValue()+":"+ReferenceArraySyntheticImplementors.anonymousValue()+":"+ReferenceArraySyntheticImplementors.localValue());System.out.println(SyntheticArrayIdentityProgram.run());}}`,
		}, "Main", "17:19:20\n3456:1:136:1:113\n")
	})
	t.Run("original_nested_math", func(t *testing.T) {
		fixture := filepath.Join("..", "testfiles", "applications", "nested_classes_and_math_gap")
		files := map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>synthetic</groupId><artifactId>nestedmath</artifactId><version>1</version></project>`}
		err := filepath.WalkDir(filepath.Join(fixture, "src"), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			name, err := filepath.Rel(filepath.Join(fixture, "src"), path)
			if err != nil {
				return err
			}
			files["src/main/java/"+filepath.ToSlash(name)] = string(content)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join(fixture, "expected.stdout"))
		if err != nil {
			t.Fatal(err)
		}
		runCampaignCompilerProjectOracle(t, files, "parity.nestedmath.MathFractal", string(expected))
	})
}
