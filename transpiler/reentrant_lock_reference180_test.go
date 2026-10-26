package transpiler

import (
	"strings"
	"testing"
)

func TestReentrantLock180CanonicalExecutionLowering(t *testing.T) {
	const source = `import java.util.concurrent.locks.ReentrantLock;
public class Use {
 final ReentrantLock lock = new ReentrantLock();
 int call() { lock.lock(); try { lock.lock(); return lock.getHoldCount(); } finally { lock.unlock(); lock.unlock(); } }
 boolean held() { return lock.isHeldByCurrentThread(); }
 boolean locked() { return lock.isLocked(); }
}`
	generated := renderGoFileFromJava(t, source)
	for _, required := range []string{"*stdjava.JavaReentrantLock", "stdjava.NewReentrantLock()", "stdjava.ReentrantLockLockExecution(", "stdjava.ReentrantLockUnlockExecution(", "stdjava.ReentrantLockGetHoldCountExecution(", "stdjava.ReentrantLockIsHeldByCurrentThreadExecution(", "stdjava.ReentrantLockIsLocked("} {
		if !strings.Contains(generated, required) {
			t.Fatalf("canonical ReentrantLock missing %s:\n%s", required, generated)
		}
	}
	if strings.Contains(generated, "ConstructReentrantLock") || strings.Contains(generated, ".lock()") || strings.Contains(generated, ".unlock()") {
		t.Fatalf("canonical lock escaped Execution-aware lowering:\n%s", generated)
	}
}

func TestReentrantLock180SourceHomonymLowering(t *testing.T) {
	const source = `class ReentrantLock { int depth; void lock(){depth++;} void unlock(){depth--;} int getHoldCount(){return depth+17;} }
public class Use { ReentrantLock lock = new ReentrantLock(); int call(){lock.lock();try{return lock.getHoldCount();}finally{lock.unlock();}} }`
	generated := renderGoFileFromJava(t, source)
	if strings.Contains(generated, "stdjava.JavaReentrantLock") || strings.Contains(generated, "stdjava.NewReentrantLock") || strings.Contains(generated, "stdjava.ReentrantLock") {
		t.Fatalf("actual source homonym was replaced by canonical lock:\n%s", generated)
	}
}
