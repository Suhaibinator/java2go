package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/stdjava"
)

func TestBuiltinReferenceArraysPreserveObjectThrowableAndGenericNullSemantics(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.concurrent.atomic.AtomicInteger;

public class BuiltinReferenceArrays {
    interface Marker { }
    static class Marked implements Marker { }

    static class ClosingResource implements AutoCloseable {
        String message;
        ClosingResource(String message) { this.message = message; }
        public void close() { throw new IllegalStateException(message); }
    }

    static <T> T first(T[] values) {
        return values[0];
    }

    static Throwable[] suppressedValues() {
        try (ClosingResource first = new ClosingResource("alpha");
             ClosingResource second = new ClosingResource("beta")) {
            throw new Exception("primary");
        } catch (Exception primary) {
            return primary.getSuppressed();
        }
    }

    public static int run() {
        Object token = new Object();
        Object[] objects = new Object[] {
            token,
            new ArrayList<String>(),
            new AtomicInteger(3),
            new RuntimeException("runtime")
        };
        int score = 0;
        for (Object value : objects) {
            if (value != null) score++;
        }
        if (objects[0] == token) score += 10;

        Throwable[] suppressed = suppressedValues();
        for (Throwable value : suppressed) {
			if (value.getMessage() == "beta") score = score * 10 + 4;
			if (value.getMessage() == "alpha") score = score * 10 + 5;
        }

        Throwable[] nullable = new Throwable[] { null, new RuntimeException("z") };
        if (nullable[0] == null) score += 100;
        if (nullable[1].getMessage() == "z") score += 1000;

        Marked[] marked = new Marked[1];
        Object[] markedView = marked;
        try {
            markedView[0] = new Object();
            score += 10000;
        } catch (ArrayStoreException expected) {
            score += 20000;
        }

        Marker[] markers = new Marker[1];
        Object[] markerView = markers;
        try {
            markerView[0] = new Object();
            score += 100000;
        } catch (ArrayStoreException expected) {
            score += 200000;
        }

        String genericNull = first(new String[] { null });
        if (genericNull == null) score += 1000000;
        return score;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`stdjava.ReferenceArrayLiteralOf[any]`,
		`stdjava.SuppressedArray(stdjava.GetSuppressed(primary))`,
		`range stdjava.ReferenceArrayIterationElements(suppressed)`,
		`stdjava.ObjectView[stdjava.Throwable]`,
		`firstJava2goExecution[string]`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("built-in reference-array lowering is missing %q:\n%s", fragment, out)
		}
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestBuiltinReferenceArrayParity(t *testing.T) {
	if got := Run(); got != 1222545 {
		t.Fatalf("Run() = %d, want 1222545", got)
    }
}
`)
}

func TestBuiltinReferenceArrayRuntimeKeepsObjectRuleNarrow(t *testing.T) {
	object := stdjava.NewObject()
	objects := stdjava.NewReferenceArray(2, stdjava.ObjectTypeID)
	stdjava.ReferenceArraySet(objects, 0, object)
	var nullObject *byte
	stdjava.ReferenceArraySet(objects, 1, nullObject)
	if got := stdjava.ReferenceArrayGet[any](objects, 0, stdjava.ObjectTypeID); got != object {
		t.Fatal("Object[] did not round-trip an opaque java.lang.Object identity")
	}
	if got := stdjava.ReferenceArrayGet[any](objects, 1, stdjava.ObjectTypeID); got != nil {
		t.Fatalf("Object[] null = %#v, want nil", got)
	}

	const (
		childType     stdjava.TypeID = "builtin-array.Child"
		interfaceType stdjava.TypeID = "builtin-array.Marker"
	)
	stdjava.RegisterJavaType(childType, stdjava.ObjectTypeID)
	stdjava.RegisterJavaType(interfaceType, stdjava.ObjectTypeID)
	for name, array := range map[string]*stdjava.ReferenceArray{
		"class":     stdjava.NewReferenceArray(1, childType),
		"interface": stdjava.NewReferenceArray(1, interfaceType),
	} {
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			stdjava.ReferenceArraySet(array, 0, object)
		}()
		if !stdjava.CaughtAs(recovered, "ArrayStoreException") {
			t.Fatalf("opaque Object stored in narrower %s array: panic = %T (%v)", name, recovered, recovered)
		}
	}

	primary := stdjava.NewException("primary")
	suppressed := stdjava.NewIllegalStateException("suppressed")
	stdjava.AddSuppressed(primary, suppressed)
	array := stdjava.GetSuppressedArray(primary)
	if array.ComponentType() != stdjava.ThrowableTypeID {
		t.Fatalf("suppressed component = %q, want %q", array.ComponentType(), stdjava.ThrowableTypeID)
	}
	if got := stdjava.ReferenceArrayGet[stdjava.Throwable](array, 0, stdjava.ThrowableTypeID); got.Message() != "suppressed" {
		t.Fatalf("Throwable[] read = %v, want suppressed exception", got)
	}
	if actual, ok := stdjava.ObjectDynamicType(suppressed); !ok || actual != stdjava.TypeID("IllegalStateException") {
		t.Fatalf("IllegalStateException runtime type = %q, %v", actual, ok)
	}
	if !stdjava.JavaTypeAssignable(stdjava.TypeID("IllegalStateException"), stdjava.ThrowableTypeID) ||
		!stdjava.JavaTypeAssignable(stdjava.TypeID("IOException"), stdjava.TypeID("Exception")) {
		t.Fatal("built-in Throwable hierarchy is missing runtime assignability edges")
	}
	runtimeOnly := stdjava.NewReferenceArray(1, stdjava.TypeID("RuntimeException"))
	_ = stdjava.ReferenceArraySet(runtimeOnly, 0, suppressed)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = stdjava.ReferenceArraySet(runtimeOnly, 0, stdjava.NewIOException("checked"))
	}()
	if !stdjava.CaughtAs(recovered, "ArrayStoreException") {
		t.Fatalf("IOException stored in RuntimeException[]: panic = %T (%v)", recovered, recovered)
	}
	var nullThrowable stdjava.Throwable
	_ = stdjava.ReferenceArraySet(array, 0, nullThrowable)
	if got := stdjava.ReferenceArrayGet[stdjava.Throwable](array, 0, stdjava.ThrowableTypeID); got != nil {
		t.Fatalf("Throwable[] null = %#v, want nil", got)
	}

	if got := stdjava.ObjectView[string](nil, stdjava.ObjectTypeID); !stdjava.StringIsNull(got) {
		t.Fatalf("erased generic String null = %q, want Java null String sentinel", got)
	}
	if got := stdjava.ObjectView[any](nil, stdjava.ObjectTypeID); got != nil {
		t.Fatalf("Object null = %#v, want nil", got)
	}
}
