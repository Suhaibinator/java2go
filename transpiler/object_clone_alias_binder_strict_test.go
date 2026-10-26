package transpiler

import (
	"go/ast"
	"testing"
)

// Keep the complete pre-existing Java binder control and oracle unchanged.
func TestObjectCloneOriginalShadowedBinderStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>clonealias</groupId><artifactId>clonealias</artifactId><version>1</version></project>`, "src/main/java/ShadowedTypeParameterProgram.java": `
interface ShadowClassMark {
    int classCode();
}

interface ShadowMethodMark {
    int methodCode();
}

interface ShadowLocalMark {
    int localCode();
}

public class ShadowedTypeParameterProgram<T extends ShadowClassMark> {

    static class ClassValue implements ShadowClassMark {
        int code;

        ClassValue(int code) {
            this.code = code;
        }

        public int classCode() {
            return code;
        }
    }

    static class MethodValue implements ShadowMethodMark {
        int code;

        MethodValue(int code) {
            this.code = code;
        }

        public int methodCode() {
            return code;
        }
    }

    static class LocalValue implements ShadowLocalMark {
        int code;

        LocalValue(int code) {
            this.code = code;
        }

        public int localCode() {
            return code;
        }
    }

    T classValue;

    ShadowedTypeParameterProgram(T classValue) {
        this.classValue = classValue;
    }

    <T extends ShadowMethodMark> int score(T methodValue, int localCode) {
        class Local<T extends ShadowLocalMark> {
            T localValue;

            Local(T localValue) {
                this.localValue = localValue;
            }

            int read() {
                return classValue.classCode() * 100
                    + methodValue.methodCode() * 10
                    + localValue.localCode();
            }
        }

        return new Local<LocalValue>(new LocalValue(localCode)).read();
    }

    public static String run() {
        return new ShadowedTypeParameterProgram<ClassValue>(new ClassValue(2))
                .score(new MethodValue(3), 4)
            + ":"
            + new ShadowedTypeParameterProgram<ClassValue>(new ClassValue(5))
                .score(new MethodValue(6), 7);
    }
}
`, "src/main/java/Main.java": `public class Main { public static void main(String[] args) { System.out.println(ShadowedTypeParameterProgram.run()); } }`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "Main", "234:567\n")
}

// Canonical receivers have no owner binders in their body. Both complete
// families and the legacy leaf plan must allocate with physical arguments.
func TestObjectCloneCanonicalAllocationHasNoFreeOwnerBinder(t *testing.T) {
	for _, test := range []struct {
		name, source, owner string
		family              bool
	}{
		{"family", `abstract class Adapter<T>{abstract T read();} interface Factory{<S> Adapter<S> create();} class Use{Adapter<String> make(){return new Adapter<String>(){String read(){return "ok";}};}}`, "Adapter", true},
		{"leaf", `class Box<X>{X value;Box(X value){this.value=value;}} class Use{static <B,T extends B> Box<B> wrap(T value){return new Box<B>(value);}}`, "Box", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			scope := helper.File.Symbols.FindClassScope(test.owner)
			ctx := classScopeCtx(scope, helper.Ctx)
			if !canonicalGenericClass(scope, ctx) {
				t.Fatal("canonical physical fixture no longer admitted")
			}
			if got := canonicalGenericFamily(scope, ctx) != nil; got != test.family {
				t.Fatalf("complete family=%v want %v", got, test.family)
			}
			declarations := generateObjectCloneDecls(ctx)
			canonicalizeGenericReceivers(declarations, ctx)
			if len(declarations) != 1 {
				t.Fatalf("copier declarations=%d", len(declarations))
			}
			method := declarations[0].(*ast.FuncDecl)
			allocation := method.Body.List[0].(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)
			if id, ok := allocation.Fun.(*ast.Ident); !ok || id.Name != "new" {
				t.Fatal("fresh copier allocation missing")
			}
			for _, parameter := range scope.TypeParameters {
				ast.Inspect(method.Body, func(node ast.Node) bool {
					if id, ok := node.(*ast.Ident); ok && id.Name == parameter.EmittedName() {
						t.Errorf("free owner binder %s remains in canonical body", id.Name)
					}
					return true
				})
			}
		})
	}
}

func TestObjectCloneCanonicalLeafStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>clonecanonical</groupId><artifactId>clonecanonical</artifactId><version>1</version></project>`, "src/main/java/clonecanonical/Main.java": `package clonecanonical;
class Value { final int code; Value(int code){this.code=code;} }
class Box<X> implements java.lang.Cloneable {
 final X value; Box(X value){Main.constructions++;this.value=value;}
 Object copy() throws CloneNotSupportedException { return super.clone(); }
}
public class Main {
 static int constructions;
 static <B,T extends B> Box<B> wrap(T value){return new Box<B>(value);}
 public static void main(String[] args) throws CloneNotSupportedException {
  Value value=new Value(7);Box<Value> source=Main.<Value,Value>wrap(value);
  Box<Value> copy=(Box<Value>)source.copy();
  System.out.println((source!=copy)+":"+(source.getClass()==copy.getClass())+":"+(source.value==copy.value)+":"+copy.value.code+":"+constructions);
 }
}`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "clonecanonical.Main", "true:true:true:7:1\n")
}

func TestObjectCloneCanonicalFamilyIdentityStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>clonefamily</groupId><artifactId>clonefamily</artifactId><version>1</version></project>`, "src/main/java/clonefamily/Main.java": `package clonefamily;
class Value { final int code; Value(int code){this.code=code;} }
abstract class Adapter<T> implements java.lang.Cloneable {
 final T value;Adapter(T input){Main.constructions++;this.value=input;}
 abstract T read();
 Object copy() throws CloneNotSupportedException{return super.clone();}
}
interface Factory {<S> Adapter<S> create();}
class Concrete extends Adapter<Value> {
 Concrete(Value input){super(input);}
 Value read(){return value;}
}
public class Main {
 static int constructions;
 public static void main(String[] args) throws CloneNotSupportedException {
  Value value=new Value(9);Concrete source=new Concrete(value);
  Adapter<Value> copy=(Adapter<Value>)source.copy();
  System.out.println((source!=copy)+":"+(copy instanceof Concrete)+":"+(source.getClass()==copy.getClass())+":"+(source.read()==copy.read())+":"+((Value)copy.read()).code+":"+constructions);
 }
}`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "clonefamily.Main", "true:true:true:true:9:1\n")
}
