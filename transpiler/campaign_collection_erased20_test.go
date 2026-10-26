package transpiler

import (
	"os"
	"testing"
)

func TestCampaignCollectionErased20JVM(t *testing.T) {
	oracle, err := os.ReadFile("testdata/collection_erased20_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": campaignStaticImportPOM26, "src/main/java/probe/Main.java": `package probe;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
public class Main {
 static int trace;
 @SuppressWarnings({"rawtypes", "unchecked"})
 static boolean pollute(Collection target, Object value) { return target.add(value); }
 static <T> boolean append(Collection<? super T> target, T value) { trace = trace * 10 + 1; return target.add(value); }
 static <T> String read(Collection<? extends T> source) { String out = ""; for (T item : source) out = out + item + ":"; return out; }
 static boolean erase(Collection<?> source, Object item) { return source.remove(item); }
 static Collection<?> identity(Collection<?> source) { return source; }
 static <T> List<T> copy(Collection<? extends T> source) { return new ArrayList<T>(source); }
 static int size(Collection<?> source) { return source.size(); }
 static String exact(Collection<String> source) { String out = ""; for (String item : source) out = out + item; return out; }
 static String badRead(Collection<?> source) { for (Object item : source) return (String)item; return "empty"; }
 public static void main(String[] args) {
  List<String> strings = new ArrayList<String>(); strings.add("seed");
  List<Object> objects = new ArrayList<Object>(); objects.add(strings);
  System.out.println(append(strings, "next") + ":" + append(objects, "wide") + ":" + trace);
  Collection<?> alias = identity(strings);
  System.out.println((alias == strings) + ":" + size(alias) + ":" + exact(strings) + ":" + read(strings));
  System.out.println(erase(alias, "seed") + ":" + strings.get(0) + ":" + size(alias) + ":" + (objects.get(0) == strings));
  List<Object> copied = Main.<Object>copy(strings);
  strings.set(0, "changed");
  System.out.println(read(alias) + ":" + objects.get(1) + ":" + (identity(null) == null));
  try { System.out.println(badRead(objects)); } catch (ClassCastException e) { System.out.println("cast"); }
  try { System.out.println(size(null)); } catch (NullPointerException e) { System.out.println("null"); }
  System.out.println(copied.get(0) + ":" + (copied == objects));
  synchronized(alias) { synchronized(strings) { System.out.println("locked"); } }
  Collection<String> exactAlias = strings; exactAlias.clear();
  System.out.println(strings.isEmpty() + ":" + alias.isEmpty());
  System.out.println("stored=" + pollute(strings, Integer.valueOf(7)) + ":" + size(alias));
  System.out.println("erased=" + read(alias));
  try { String broken = strings.get(0); System.out.println(broken); } catch (ClassCastException e) { System.out.println("late-cast:" + size(alias)); }
  System.out.println("removed=" + erase(alias, Integer.valueOf(7)) + ":" + strings.isEmpty());
 }
}
`}, "probe.Main", string(oracle))
}
