package transpiler

import "testing"

func TestCICollectorMergePreservesCallerExecution(t *testing.T) {
	const source = `import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
public class CollectorMergeExecution {
 public static String run() {
  Object lock = new Object();
  Thread caller = Thread.currentThread();
  List<String> words = new ArrayList<String>();
  words.add("a"); words.add("b"); words.add("c");
  synchronized (lock) {
   Map<String, String> merged = words.stream().collect(Collectors.toMap(
    word -> "key", word -> word,
    (left, right) -> left + right + ":" + (Thread.currentThread() == caller) + ":" + Thread.holdsLock(lock)));
   return merged.get("key");
  }
 }
}`
	verifyCanonicalStringStreamOracle(t, "CollectorMergeExecution", source)
}
