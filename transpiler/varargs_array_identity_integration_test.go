package transpiler

import "testing"

func TestVarargsArrays_PreserveIdentityMutationNullAndCovariantStores(t *testing.T) {
	out := renderGoFileFromJava(t, `
interface VarargsCapture {
    int read();
}

class PrimitiveVarargsBase {
    int[] values;

    PrimitiveVarargsBase(int... values) {
        this.values = values;
        if (values != null && values.length != 0) values[0] += 1;
    }
}

class PrimitiveVarargsChild extends PrimitiveVarargsBase {
    PrimitiveVarargsChild(int[] values) {
        super(values);
    }
}

class PrimitiveVarargsThis {
    int[] values;

    PrimitiveVarargsThis(int marker, int... values) {
        this.values = values;
        if (values != null && values.length != 0) values[0] += 1;
    }

    PrimitiveVarargsThis(int[] values) {
        this(0, values);
    }
}

class ReferenceVarargsBase {
    Object[] values;

    ReferenceVarargsBase(Object... values) {
        this.values = values;
        if (values != null && values.length != 0) values[0] = "constructor";
    }

    void reject() {
        values[0] = new Object();
    }
}

class ReferenceVarargsChild extends ReferenceVarargsBase {
    ReferenceVarargsChild(Object[] values) {
        super(values);
    }
}

class ReferenceVarargsThis {
    Object[] values;

    ReferenceVarargsThis(int marker, Object... values) {
        this.values = values;
        if (values != null && values.length != 0) values[0] = "delegated";
    }

    ReferenceVarargsThis(Object[] values) {
        this(0, values);
    }
}

public class VarargsArrayIdentityProgram {
    static int[] primitive(int... values) {
        if (values != null && values.length != 0) values[0] += 1;
        return values;
    }

    static Object[] reference(Object... values) {
        if (values != null && values.length != 0) values[0] = "method";
        return values;
    }

    static void reject(Object... values) {
        values[0] = new Object();
    }

    static VarargsCapture readCaptured(int... values) {
        return () -> values == null ? -1 : values.length == 0 ? 0 : values.length * 100 + values[0];
    }

    static VarargsCapture readLocalCaptured(int... values) {
        class CapturedReader implements VarargsCapture {
            public int read() {
                if (values == null) return -1;
                return values.length == 0 ? 0 : values.length * 100 + values[0];
            }
        }
        return new CapturedReader();
    }

    static VarargsCapture readAnonymousCaptured(int... values) {
        return new VarargsCapture() {
            public int read() {
                if (values == null) return -1;
                return values.length == 0 ? 0 : values.length * 100 + values[0];
            }
        };
    }

    public static String run() {
        int[] numbers = new int[] { 1 };
        if (primitive(numbers) != numbers || numbers[0] != 2) return "primitive alias";
        if (primitive((int[]) null) != null) return "primitive null";
        int[] emptyNumbers = primitive();
        if (emptyNumbers == null || emptyNumbers.length != 0 || emptyNumbers == primitive()) {
            return "primitive empty identity";
        }

        String[] words = new String[] { "original" };
        if (reference(words) != words || !words[0].equals("method")) return "reference alias";
        if (reference((Object[]) null) != null) return "reference null";
        Object[] emptyWords = reference();
        if (emptyWords == null || emptyWords.length != 0 || emptyWords == reference()) {
            return "reference empty identity";
        }
        boolean methodRejected = false;
        try {
            reject(words);
        } catch (ArrayStoreException e) {
            methodRejected = true;
        }
        if (!methodRejected || !words[0].equals("method")) return "method covariance";

        PrimitiveVarargsChild primitiveChild = new PrimitiveVarargsChild(numbers);
        if (primitiveChild.values != numbers || numbers[0] != 3) return "super primitive alias";
        PrimitiveVarargsThis primitiveThis = new PrimitiveVarargsThis(numbers);
        if (primitiveThis.values != numbers || numbers[0] != 4) return "this primitive alias";
        if (new PrimitiveVarargsChild(null).values != null) return "super primitive null";
        if (new PrimitiveVarargsThis(null).values != null) return "this primitive null";

        ReferenceVarargsChild referenceChild = new ReferenceVarargsChild(words);
        if (referenceChild.values != words || !words[0].equals("constructor")) return "super reference alias";
        boolean constructorRejected = false;
        try {
            referenceChild.reject();
        } catch (ArrayStoreException e) {
            constructorRejected = true;
        }
        if (!constructorRejected || !words[0].equals("constructor")) return "constructor covariance";
        ReferenceVarargsThis referenceThis = new ReferenceVarargsThis(words);
        if (referenceThis.values != words || !words[0].equals("delegated")) return "this reference alias";
        if (new ReferenceVarargsChild(null).values != null) return "super reference null";
        if (new ReferenceVarargsThis(null).values != null) return "this reference null";

        int[] capturedValues = new int[] { 21 };
        VarargsCapture lambda = readCaptured(capturedValues);
        VarargsCapture local = readLocalCaptured(capturedValues);
        VarargsCapture anonymous = readAnonymousCaptured(capturedValues);
        if (lambda.read() != 121 || local.read() != 121 || anonymous.read() != 121) return "capture values";
        capturedValues[0] = 22;
        if (lambda.read() != 122 || local.read() != 122 || anonymous.read() != 122) return "capture alias";
        if (readCaptured(23, 24).read() != 223) return "lambda expanded capture";
        if (readLocalCaptured(25, 26).read() != 225) return "local expanded capture";
        if (readAnonymousCaptured(27, 28).read() != 227) return "anonymous expanded capture";
        if (readCaptured((int[]) null).read() != -1) return "lambda null capture";
        if (readLocalCaptured((int[]) null).read() != -1) return "local null capture";
        if (readAnonymousCaptured((int[]) null).read() != -1) return "anonymous null capture";
        if (readCaptured().read() != 0) return "lambda empty capture";
        if (readLocalCaptured().read() != 0) return "local empty capture";
        if (readAnonymousCaptured().read() != 0) return "anonymous empty capture";
        return "ok";
    }
}
`)

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestVarargsArrayIdentity(t *testing.T) {
    if got := Run(); got != "ok" {
        t.Fatalf("Run() = %q, want ok", got)
    }
}
`)
}
