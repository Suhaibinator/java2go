package transpiler

import (
	"strings"
	"testing"
)

func TestConcurrency_AtomicAndThreadTranspile(t *testing.T) {
	src := `
import java.util.concurrent.atomic.AtomicInteger;
public class Worker {
    private AtomicInteger counter = new AtomicInteger(0);
    public void bump() {
        counter.incrementAndGet();
    }
    public int total() {
        return counter.get();
    }
    public static void nap() {
        Thread.sleep(1);
    }
}
`
	out := renderGoFileFromJava(t, src)
	for _, want := range []string{
		"*stdjava.AtomicInteger",
		"stdjava.NewAtomicInteger(0)",
		".IncrementAndGet()",
		".Get()",
		"stdjava.ThreadSleep(",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected generated code to contain %q, got:\n%s", want, out)
		}
	}
}

func TestConcurrency_SynchronizedMethodTranspile(t *testing.T) {
	src := `
public class Counter {
    private int count = 0;
    public synchronized void inc() { count = count + 1; }
    public static synchronized void reset() { }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.MonitorEnter(") {
		t.Errorf("instance synchronized method should enter a monitor, got:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.ClassMonitorEnter(") {
		t.Errorf("static synchronized method should enter a class monitor, got:\n%s", out)
	}
	if !strings.Contains(out, "defer stdjava.MonitorExit(") {
		t.Errorf("synchronized method should defer monitor exit, got:\n%s", out)
	}
}

func TestConcurrency_RuntimeMutualExclusion(t *testing.T) {
	// A synchronized increment, driven concurrently from goroutines, must not
	// lose updates. Uses a plain int field guarded by the method monitor.
	src := `
public class SafeCounter {
    private int count = 0;
    public synchronized void inc() { count = count + 1; }
    public synchronized int get() { return count; }
}
`
	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import (
	"sync"
	"testing"
)

func TestMutualExclusion(t *testing.T) {
	c := NewSafeCounter()
	var wg sync.WaitGroup
	const goroutines, per = 30, 200
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < per; j++ {
				c.Inc()
			}
		}()
	}
	wg.Wait()
	if got := c.Get(); got != goroutines*per {
		t.Fatalf("synchronized counter lost updates: got %d, want %d", got, goroutines*per)
	}
}
`)
}

func TestConcurrency_RuntimeThreadJoin(t *testing.T) {
	// Thread.start()/join() must run the runnable to completion before join
	// returns. The runnable sets a field via an AtomicInteger so the main
	// goroutine observes the result deterministically.
	src := `
import java.util.concurrent.atomic.AtomicInteger;
public class Spawner {
    public static int run() {
        AtomicInteger result = new AtomicInteger(0);
        Thread t = new Thread(() -> result.set(7));
        t.start();
        t.join();
        return result.get();
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestJoin(t *testing.T) {
	if got := Run(); got != 7 {
		t.Fatalf("join did not observe the thread's write: got %d, want 7", got)
	}
}
`)
}
