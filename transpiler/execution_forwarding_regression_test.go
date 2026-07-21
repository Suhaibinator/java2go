package transpiler

import (
	"strings"
	"testing"
)

func TestExecutionForwarding_BoundGenericInterfaceMethodReferenceUsesReceiverTypeArguments(t *testing.T) {
	src := `
interface TextSupplier {
    String get();
}

interface GenericReader<T> {
    T read();
}

class StringReader implements GenericReader<String> {
    private String value;

    StringReader(String value) {
        this.value = value;
    }

    public synchronized String read() {
        return value;
    }
}

public class GenericInterfaceMethodReferenceProgram {
    public static String run() {
        GenericReader<String> reader = new StringReader("bound-interface");
        TextSupplier supplier = reader::read;
        synchronized (reader) {
            return supplier.get();
        }
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, ".(genericReaderJava2goExecution[string])") {
		t.Fatalf("bound generic interface reference did not use the receiver's concrete type argument:\n%s", out)
	}
	if strings.Contains(flat, "genericReaderJava2goExecution[T]") {
		t.Fatalf("bound generic interface reference leaked an out-of-scope declaration type parameter:\n%s", out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestBoundGenericInterfaceReference(t *testing.T) {
    if got := Run(); got != "bound-interface" {
        t.Fatalf("Run() = %q, want bound-interface", got)
    }
}
`)
}

func TestExecutionForwarding_BoundGenericAbstractMethodReferenceUsesReceiverTypeArguments(t *testing.T) {
	src := `
interface AbstractTextSupplier {
    String get();
}

abstract class GenericAbstractReader<T> {
    public abstract T read();
}

public class GenericAbstractMethodReferenceProgram {
    public static AbstractTextSupplier bind(GenericAbstractReader<String> reader) {
        return reader::read;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, ".(genericAbstractReaderJava2goExecution[string])") {
		t.Fatalf("bound generic abstract reference did not use the receiver's concrete type argument:\n%s", out)
	}
	if strings.Contains(flat, "genericAbstractReaderJava2goExecution[T]") {
		t.Fatalf("bound generic abstract reference leaked an out-of-scope declaration type parameter:\n%s", out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestGeneratedAbstractReferenceCompiles(t *testing.T) {}
`)
}

func TestExecutionForwarding_VariadicInterfaceMethodReferenceExpandsForwardedSlice(t *testing.T) {
	src := `
interface IntArrayTask {
    int apply(int[] values);
}

interface VariadicSummer {
    int sum(int... values);
}

class Summer implements VariadicSummer {
    public synchronized int sum(int... values) {
        int total = 0;
        for (int value : values) {
            total += value;
        }
        return total;
    }
}

public class VariadicMethodReferenceProgram {
    public static int run() {
        VariadicSummer summer = new Summer();
        IntArrayTask task = summer::sum;
        synchronized (summer) {
            return task.apply(new int[] { 2, 3, 5 });
        }
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, ".SumJava2goExecution(__java2goExecution, values...)") {
		t.Fatalf("execution-aware variadic method-reference call did not expand its forwarded slice:\n%s", out)
	}
	if !strings.Contains(flat, ".Sum(values...)") {
		t.Fatalf("public fallback for variadic method reference did not expand its forwarded slice:\n%s", out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestVariadicMethodReference(t *testing.T) {
    if got := Run(); got != 10 {
        t.Fatalf("Run() = %d, want 10", got)
    }
}
`)
}

func TestExecutionForwarding_VariadicEnumWrapperExpandsForwardedSlice(t *testing.T) {
	src := `
public class VariadicEnumProgram {
    enum Calculator {
        SUM {
            public int calculate(int... values) {
                int total = 0;
                for (int value : values) {
                    total += value;
                }
                return total;
            }
        },
        COUNT;

        public int calculate(int... values) {
            int count = 0;
            for (int ignored : values) {
                count++;
            }
            return count;
        }
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "_VariadicEnumProgramcalculator_SUM_Calculate(__java2goExecution") ||
		strings.Count(flat, "vr, values...)") < 2 {
		t.Fatalf("variadic enum wrapper did not expand the forwarded slice:\n%s", out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestVariadicEnum(t *testing.T) {
    if got := SUM.Calculate(4, 6, 8); got != 18 {
        t.Fatalf("Calculate() = %d, want 18", got)
    }
    if got := COUNT.Calculate(4, 6, 8); got != 3 {
        t.Fatalf("default Calculate() = %d, want 3", got)
    }
}
`)
}
