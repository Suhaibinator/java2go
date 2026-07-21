package transpiler

import (
	"strings"
	"testing"
)

func TestRunnableExecution_DirectLambdaAndHiddenNameCollision(t *testing.T) {
	src := `
public class RunnableExecutionProgram {
    private static final Object LOCK = new Object();
    private static int trace = 0;

    private static void nested(int digit) {
        synchronized (LOCK) {
            trace = trace * 10 + digit;
        }
    }

    public static int lambdaPath() {
        synchronized (LOCK) {
            Runnable runnable = () -> nested(1);
            runnable.run();
        }
        return trace;
    }

    public static int collisionPath() {
        synchronized (LOCK) {
            Runnable runnable = new Runnable() {
                public void runJava2goExecution() {
                    RunnableExecutionProgram.nested(9);
                }

                public void run() {
                    RunnableExecutionProgram.nested(2);
                }
            };
            runnable.run();
        }
        return trace;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "stdjava.RunRunnableExecution(__java2goExecution, runnable)") {
		t.Fatalf("Runnable calls did not use the execution-preserving bridge:\n%s", out)
	}
	if !strings.Contains(flat, "*RunnableExecutionProgramAnon1) RunJava2goExecution1(") {
		t.Fatalf("Runnable hidden method did not avoid the user-declared name collision:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import (
    "testing"
    "time"
)

func TestRunnableExecution(t *testing.T) {
    results := make(chan [2]int32, 1)
    go func() {
        results <- [2]int32{LambdaPath(), CollisionPath()}
    }()

    select {
    case got := <-results:
        if got != [2]int32{1, 12} {
            t.Fatalf("Runnable execution trace = %v, want [1 12]", got)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("Runnable callback lost its execution token and deadlocked on reentrant monitor entry")
    }
}
`)
}

func TestRunnableExecution_MethodReferencesPreserveCallerToken(t *testing.T) {
	src := `
public class RunnableMethodReferenceProgram {
    private static final Object LOCK = new Object();
    private static int trace = 0;

    private void instanceTarget() {
        synchronized (LOCK) {
            trace = trace * 10 + 1;
        }
    }

    private static void staticTarget() {
        synchronized (LOCK) {
            trace = trace * 10 + 2;
        }
    }

    public int exercise() {
        synchronized (LOCK) {
            Runnable bound = this::instanceTarget;
            Runnable staticReference = RunnableMethodReferenceProgram::staticTarget;
            bound.run();
            staticReference.run();
        }
        return trace;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, ".instanceTargetJava2goExecution") {
		t.Fatalf("bound Runnable method reference selected the public fresh-token wrapper:\n%s", out)
	}
	if !strings.Contains(flat, "staticReference := staticTargetJava2goExecution") {
		t.Fatalf("static Runnable method reference selected the public fresh-token wrapper:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import (
    "testing"
    "time"
)

func TestRunnableMethodReferences(t *testing.T) {
    result := make(chan int32, 1)
    go func() {
        result <- NewRunnableMethodReferenceProgram().Exercise()
    }()

    select {
    case got := <-result:
        if got != 12 {
            t.Fatalf("Runnable method-reference trace = %d, want 12", got)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("Runnable method reference used a fresh execution token and deadlocked")
    }
}
`)
}
