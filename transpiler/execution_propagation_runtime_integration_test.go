package transpiler

import "testing"

func TestExecutionPropagation_ReentrantCallsAndCallbacks(t *testing.T) {
	src := `
public class ExecutionPropagationProgram {
    private static final Object LOCK = new Object();
    private static int trace = 0;

    // The generated hidden implementation for ping must avoid this field.
    public int pingJava2goExecution = 5;

    public int ping() {
        synchronized (LOCK) {
            return pingJava2goExecution;
        }
    }

    private static void helper() {
        synchronized (LOCK) {
            trace += 1;
        }
    }

    private static void addTrace(int amount) {
        trace += amount;
    }

    private static int currentTrace() {
        return trace;
    }

    public static int direct() {
        synchronized (LOCK) {
            helper();
            return trace;
        }
    }

    private static int recurse(int depth) {
        synchronized (LOCK) {
            if (depth == 0) {
                return 1;
            }
            return 1 + recurse(depth - 1);
        }
    }

    static class Nested {
        Nested(Object lock) {
            synchronized (lock) {
                ExecutionPropagationProgram.addTrace(10);
            }
        }
    }

    public static int construct() {
        synchronized (LOCK) {
            new Nested(LOCK);
            return trace;
        }
    }

    interface Getter {
        int get();
        Object lock();

        default int twice() {
            synchronized (lock()) {
                return get() + get();
            }
        }
    }

    static class GetterImpl implements Getter {
        Object monitor;

        GetterImpl(Object monitor) {
            this.monitor = monitor;
        }

        public Object lock() {
            return monitor;
        }

        public int get() {
            synchronized (monitor) {
                return 3;
            }
        }
    }

    interface Task {
        void run();
    }

    interface IntTask {
        int run();
    }

    interface Pinger {
        int call(ExecutionPropagationProgram value);
    }

    interface Factory {
        Nested make(Object lock);
    }

    interface DefaultFactory {
        DefaultNested make();
    }

    static class DefaultNested {
        int value = initializeDefault();
    }

    interface Collision {
        int value();
    }

    static class CollisionJava2goExecution {
    }

    static class CollisionImpl implements Collision {
        public int value() {
            synchronized (this) {
                return 8;
            }
        }
    }

    public static int viaInterface() {
        synchronized (LOCK) {
            Getter getter = new GetterImpl(LOCK);
            return getter.twice();
        }
    }

    private static int referenceTarget() {
        synchronized (LOCK) {
            return 7;
        }
    }

    private static int initializeDefault() {
        synchronized (LOCK) {
            trace += 20;
            return trace;
        }
    }

    public static int viaStaticMethodReference() {
        IntTask task = ExecutionPropagationProgram::referenceTarget;
        synchronized (LOCK) {
            return task.run();
        }
    }

    public static int viaBoundMethodReference() {
        ExecutionPropagationProgram target = new ExecutionPropagationProgram();
        IntTask task = target::ping;
        synchronized (LOCK) {
            return task.run();
        }
    }

    public static int viaUnboundMethodReference() {
        Pinger task = ExecutionPropagationProgram::ping;
        synchronized (LOCK) {
            return task.call(new ExecutionPropagationProgram());
        }
    }

    public static int viaConstructorReference() {
        Factory factory = Nested::new;
        synchronized (LOCK) {
            factory.make(LOCK);
            return trace;
        }
    }

    public static int viaDefaultConstructorReference() {
        DefaultFactory factory = DefaultNested::new;
        synchronized (LOCK) {
            return factory.make().value;
        }
    }

    public static int viaCompanionTypeCollision() {
        Collision value = new CollisionImpl();
        synchronized (value) {
            return value.value();
        }
    }

    public static int viaLambda() {
        Object lock = LOCK;
        Task task = () -> {
            synchronized (lock) {
                ExecutionPropagationProgram.addTrace(100);
            }
        };
        synchronized (lock) {
            task.run();
        }
        return currentTrace();
    }

    public static int viaRunnable() {
        Object lock = LOCK;
        Runnable task = new Runnable() {
            public void run() {
                synchronized (lock) {
                    ExecutionPropagationProgram.addTrace(1000);
                }
            }
        };
        synchronized (lock) {
            task.run();
        }
        return currentTrace();
    }

    public static boolean threadTokenIsolated() {
        Thread worker;
        Object lock = LOCK;
        synchronized (lock) {
            int before = currentTrace();
            worker = new Thread(() -> {
                synchronized (lock) {
                    ExecutionPropagationProgram.addTrace(10000);
                }
            });
            worker.start();
            Thread.sleep(30);
            if (currentTrace() != before) {
                return false;
            }
        }
        worker.join();
        return currentTrace() >= 10000;
    }

    public static String run() {
        trace = 0;
        int directResult = direct();
        int recursiveResult = recurse(3);
        int constructorResult = construct();
        int interfaceResult = viaInterface();
        int staticReferenceResult = viaStaticMethodReference();
        int boundReferenceResult = viaBoundMethodReference();
        int unboundReferenceResult = viaUnboundMethodReference();
        int constructorReferenceResult = viaConstructorReference();
        int defaultConstructorReferenceResult = viaDefaultConstructorReference();
        int companionCollisionResult = viaCompanionTypeCollision();
        int lambdaResult = viaLambda();
        int runnableResult = viaRunnable();
        boolean isolated = threadTokenIsolated();
        int collisionResult = new ExecutionPropagationProgram().ping();
        return directResult + ":" + recursiveResult + ":" + constructorResult + ":" +
               interfaceResult + ":" + staticReferenceResult + ":" + boundReferenceResult + ":" +
               unboundReferenceResult + ":" + constructorReferenceResult + ":" + defaultConstructorReferenceResult + ":" +
               companionCollisionResult + ":" + lambdaResult + ":" +
               runnableResult + ":" + isolated + ":" + collisionResult;
    }
}
	`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestExecutionPropagation(t *testing.T) {
	const want = "1:4:11:6:7:5:5:21:41:8:141:1141:true:5"
	if got := Run(); got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}

func TestExecutionPropagation_SynchronizedEnumAndResourceClose(t *testing.T) {
	src := `
public class ExecutionPropagationEdges {
    enum ReentrantEnum {
        ONLY;

        public synchronized int outer() {
            synchronized (this) {
                return inner();
            }
        }

        private int inner() {
            synchronized (this) {
                return 9;
            }
        }

        public synchronized String toString() {
            synchronized (this) {
                return "enum-" + inner();
            }
        }
    }

    static class Resource implements AutoCloseable {
        public int closeJava2goExecution = 0;
        int closed = 0;

        public synchronized void close() {
            synchronized (this) {
                closed++;
            }
        }
    }

    public static int run() {
        Resource resource = new Resource();
        synchronized (resource) {
            try (AutoCloseable __java2goCloseExecutionReceiver = resource) {
                // The deferred close still runs while the outer monitor is held.
            }
        }
        String text;
        synchronized (ReentrantEnum.ONLY) {
            text = String.valueOf(ReentrantEnum.ONLY);
        }
        Object erased = ReentrantEnum.ONLY;
        String erasedText;
        synchronized (erased) {
            erasedText = String.valueOf(erased);
        }
        String compound = "x";
        synchronized (erased) {
            compound += erased;
        }
        return ReentrantEnum.ONLY.outer() * 10 + resource.closed + text.length() +
               erasedText.length() + compound.length();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestSynchronizedEnumAndResourceClose(t *testing.T) {
	if got := Run(); got != 110 {
		t.Fatalf("Run() = %d, want 110", got)
	}
}
`)
}

func TestExecutionPropagation_VariadicSAMAdapter(t *testing.T) {
	src := `
interface VariadicTask {
    int sum(int... values);
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import (
	"testing"

	"github.com/NickyBoy89/java2go/stdjava"
)

func TestVariadicSAM(t *testing.T) {
	task := NewvariadicTaskFuncAdapter(func(values ...int32) int32 {
		var total int32
		for _, value := range values {
			total += value
		}
		return total
	})
	if got := task.Sum(1, 2, 3); got != 6 {
		t.Fatalf("public Sum = %d, want 6", got)
	}
	executionTask := task.(variadicTaskJava2goExecution)
	if got := executionTask.SumJava2goExecution(stdjava.NewExecution(), 4, 5, 6); got != 15 {
		t.Fatalf("execution Sum = %d, want 15", got)
	}
}
`)
}

func TestExecutionPropagation_SynchronizedAnonymousSAM(t *testing.T) {
	src := `
interface SynchronizedTask {
    int run();
}

public class SynchronizedAnonymousSAM {
    public static int run() {
        SynchronizedTask task = new SynchronizedTask() {
            public synchronized int run() {
                synchronized (this) {
                    return 42;
                }
            }
        };
        synchronized (task) {
            return task.run();
        }
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestSynchronizedAnonymousSAM(t *testing.T) {
	if got := Run(); got != 42 {
		t.Fatalf("Run() = %d, want 42", got)
	}
}
`)
}
