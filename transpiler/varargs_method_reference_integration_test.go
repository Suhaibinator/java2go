package transpiler

import "testing"

func TestVarargsMethodReferences_AdaptExpandedAndFixedArraySAMArguments(t *testing.T) {
	out := renderGoFileFromJava(t, `
interface VarargsPair {
    int apply(int a, int b);
}

interface VarargsUnary {
    int apply(int a);
}

interface VarargsEmpty {
    int apply();
}

interface VarargsArray {
    int apply(int[] values);
}

interface VarargsReceiverPair {
    int apply(VarargsSummation receiver, int a, int b);
}

interface VarargsReceiverUnary {
    int apply(VarargsSummation receiver, int a);
}

interface VarargsReceiverEmpty {
    int apply(VarargsSummation receiver);
}

interface VarargsReceiverArray {
    int apply(VarargsSummation receiver, int[] values);
}

interface VarargsPairFactory {
    VarargsTotal create(int a, int b);
}

interface VarargsUnaryFactory {
    VarargsTotal create(int a);
}

interface VarargsEmptyFactory {
    VarargsTotal create();
}

interface VarargsArrayFactory {
    VarargsTotal create(int[] values);
}

class VarargsSummation {
    int[] last;

    int sum(int... values) {
        last = values;
        if (values == null) return -1;
        int result = values.length * 100;
        for (int value : values) result += value;
        return result;
    }
}

class VarargsTotal {
    int[] values;
    int total;

    VarargsTotal(int... values) {
        this.values = values;
        if (values == null) {
            total = -1;
        } else {
            total = values.length * 100;
            for (int value : values) total += value;
        }
    }
}

public class VarargsMethodReferenceProgram {
    static int[] last;

    static int sum(int... values) {
        last = values;
        if (values == null) return -1;
        int result = values.length * 100;
        for (int value : values) result += value;
        return result;
    }

    public static String run() {
        int[] values = new int[] { 6, 7 };

        VarargsPair staticPair = VarargsMethodReferenceProgram::sum;
        VarargsUnary staticUnary = VarargsMethodReferenceProgram::sum;
        VarargsEmpty staticEmpty = VarargsMethodReferenceProgram::sum;
        VarargsArray staticArray = VarargsMethodReferenceProgram::sum;
        if (staticPair.apply(2, 3) != 205) return "static pair";
        if (staticUnary.apply(4) != 104) return "static unary";
        if (staticEmpty.apply() != 0) return "static empty";
        int[] firstEmpty = last;
        if (staticEmpty.apply() != 0 || firstEmpty == last) return "static empty identity";
        if (staticArray.apply(values) != 213 || last != values) return "static fixed array";
        if (staticArray.apply(null) != -1 || last != null) return "static null array";

        VarargsSummation receiver = new VarargsSummation();
        VarargsPair boundPair = receiver::sum;
        VarargsUnary boundUnary = receiver::sum;
        VarargsEmpty boundEmpty = receiver::sum;
        VarargsArray boundArray = receiver::sum;
        if (boundPair.apply(8, 9) != 217) return "bound pair";
        if (boundUnary.apply(10) != 110) return "bound unary";
        if (boundEmpty.apply() != 0) return "bound empty";
        if (boundArray.apply(values) != 213 || receiver.last != values) return "bound fixed array";
        if (boundArray.apply(null) != -1 || receiver.last != null) return "bound null array";

        VarargsReceiverPair unboundPair = VarargsSummation::sum;
        VarargsReceiverUnary unboundUnary = VarargsSummation::sum;
        VarargsReceiverEmpty unboundEmpty = VarargsSummation::sum;
        VarargsReceiverArray unboundArray = VarargsSummation::sum;
        if (unboundPair.apply(receiver, 11, 12) != 223) return "unbound pair";
        if (unboundUnary.apply(receiver, 13) != 113) return "unbound unary";
        if (unboundEmpty.apply(receiver) != 0) return "unbound empty";
        if (unboundArray.apply(receiver, values) != 213 || receiver.last != values) return "unbound fixed array";
        if (unboundArray.apply(receiver, null) != -1 || receiver.last != null) return "unbound null array";

        VarargsPairFactory pairFactory = VarargsTotal::new;
        VarargsUnaryFactory unaryFactory = VarargsTotal::new;
        VarargsEmptyFactory emptyFactory = VarargsTotal::new;
        VarargsArrayFactory arrayFactory = VarargsTotal::new;
        if (pairFactory.create(14, 15).total != 229) return "constructor pair";
        if (unaryFactory.create(16).total != 116) return "constructor unary";
        VarargsTotal empty = emptyFactory.create();
        if (empty.total != 0 || empty.values == null) return "constructor empty";
        if (empty.values == emptyFactory.create().values) return "constructor empty identity";
        VarargsTotal fixed = arrayFactory.create(values);
        if (fixed.total != 213 || fixed.values != values) return "constructor fixed array";
        VarargsTotal missing = arrayFactory.create(null);
        if (missing.total != -1 || missing.values != null) return "constructor null array";
        return "ok";
    }
}
`)

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestVarargsMethodReferenceAdaptation(t *testing.T) {
    if got := Run(); got != "ok" {
        t.Fatalf("Run() = %q, want ok", got)
    }
}
`)
}
