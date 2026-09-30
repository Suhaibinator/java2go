package transpiler

import "testing"

func TestMethodLocalIdentityHooksDoNotCollideWithJavaMembers(t *testing.T) {
	src := `
public class LocalIdentityMemberCollisionProgram {
    static class Base {
        int base() { return 1; }
    }

    public static int run() {
        class Leaf {
            int JavaDynamicTypeID = 4;

            int JavaDynamicTypeID() { return 5; }

            int score() {
                return JavaDynamicTypeID * 10 + JavaDynamicTypeID();
            }
        }

        class Child extends Base {
            int Java2goReferenceDynamicType = 6;
            int Java2goReferenceView = 7;

            int Java2goReferenceDynamicType() { return 8; }
            int Java2goReferenceView() { return 9; }

            int score() {
                return Java2goReferenceDynamicType
                    + Java2goReferenceView
                    + Java2goReferenceDynamicType()
                    + Java2goReferenceView();
            }
        }

        Leaf[] leaves = new Leaf[] { new Leaf() };
        Child[] children = new Child[] { new Child() };
        Object[] leafObjects = leaves;
        Object[] childObjects = children;
        leafObjects[0] = new Leaf();
        childObjects[0] = new Child();
        return leaves[0].score() * 100 + children[0].score();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestLocalIdentityMemberCollisionParity(t *testing.T) {
    if got := Run(); got != 4530 {
        t.Fatalf("Run() = %d, want Java member and generated identity hook parity", got)
    }
}
`)
}
