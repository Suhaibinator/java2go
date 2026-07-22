package transpiler

import (
	"strings"
	"testing"
)

func TestSyntheticAndEnumValuesCarryTransitiveArrayIdentity(t *testing.T) {
	src := `
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

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`JavaDynamicTypeID() stdjava.TypeID`,
		`stdjava.TypeID("SyntheticArrayIdentityProgram$SyntheticArrayIdentityProgramAnon1"), stdjava.ObjectTypeID, stdjava.TypeID("SyntheticArrayIdentityProgram$Pair")`,
		`stdjava.RegisterJavaType(stdjava.TypeID("SyntheticArrayIdentityProgram$Sub"), stdjava.ObjectTypeID, stdjava.TypeID("SyntheticArrayIdentityProgram$Super"))`,
		`stdjava.RegisterJavaType(stdjava.TypeID("SyntheticArrayIdentityProgram$Parent"), stdjava.ObjectTypeID, stdjava.TypeID("SyntheticArrayIdentityProgram$Sub"))`,
		`stdjava.RegisterJavaType(stdjava.TypeID("SyntheticArrayIdentityProgram$Inherited"), stdjava.TypeID("SyntheticArrayIdentityProgram$Parent"))`,
		`stdjava.RegisterJavaType(stdjava.TypeID("SyntheticArrayIdentityProgram$Color"), stdjava.ObjectTypeID)`,
		`func (synthetic *SyntheticArrayIdentityProgramcolor) JavaDynamicTypeID() stdjava.TypeID`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("generated identity is missing %q:\n%s", fragment, out)
		}
	}
	if got := strings.Count(flat, `case stdjava.TypeID("SyntheticArrayIdentityProgram$Super"):`); got != 2 {
		t.Fatalf("transitive Super view count = %d, want only Parent and Inherited concrete-hierarchy views:\n%s", got, out)
	}
	if !strings.Contains(flat, `func (sy *SyntheticArrayIdentityProgramsuperOnly) JavaDynamicTypeID() stdjava.TypeID`) {
		t.Fatalf("leaf SuperOnly should use lightweight nominal identity:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestSyntheticArrayIdentityRuntime(t *testing.T) {
    if got := Run(); got != "3456:1:136:1:113" {
        t.Fatalf("Run() = %q, want exact anonymous/interface/enum array parity", got)
    }
}
`)
}
