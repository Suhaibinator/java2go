package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeThreadLocal(t *testing.T) {
	const source = `import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;
public class CampaignRuntimeThreadLocal {
 static final AtomicInteger INITIALIZATIONS = new AtomicInteger(0);
 static final ThreadLocal<String> LOCAL = ThreadLocal.withInitial(() -> { INITIALIZATIONS.incrementAndGet(); return Thread.currentThread().getName(); });
 public static String run() throws Exception {
  String main = LOCAL.get();
  ExecutorService worker = Executors.newSingleThreadExecutor();
  Future<String> first = worker.submit(() -> {
   boolean own = LOCAL.get().equals(Thread.currentThread().getName());
   LOCAL.set("kept");
   return own + ":" + LOCAL.get();
  });
  String firstValue = first.get();
  Future<String> second = worker.submit(() -> {
   String kept = LOCAL.get();
   LOCAL.set(null);
   boolean storedNull = LOCAL.get() == null;
   LOCAL.remove();
   boolean renewed = LOCAL.get().equals(Thread.currentThread().getName());
   LOCAL.remove();
   return kept + ":" + storedNull + ":" + renewed;
  });
  String secondValue=second.get();
  worker.shutdown();worker.awaitTermination(10,TimeUnit.SECONDS);
  ThreadLocal<String> empty = new ThreadLocal<>();
  boolean defaultNull = empty.get() == null;
  return firstValue + ":" + secondValue + ":" + main.equals(LOCAL.get()) + ":" + INITIALIZATIONS.get() + ":" + defaultNull;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeThreadLocal", source)
	t.Logf("JVM ThreadLocal oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestThreadLocalOracle(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}

func TestCampaignRuntimeThreadLocalNamedSupplier(t *testing.T) {
	const source = `import java.util.function.Supplier;
import java.util.Optional;
import java.util.concurrent.*;
public class CampaignRuntimeThreadLocalNamedSupplier {
 static class Named implements Supplier<String> {
  public String get(){return Thread.currentThread().getName();}
 }
 static String currentName(){return Thread.currentThread().getName();}
 public static String run() throws Exception {
  String prefix="captured:";
  Supplier<String> supplier=()->prefix+Thread.currentThread().getName();
  Supplier<String> named=new Named();
  Supplier<String> reference=CampaignRuntimeThreadLocalNamedSupplier::currentName;
  ThreadLocal<String> local=ThreadLocal.withInitial(supplier);
  ThreadLocal<String> source=ThreadLocal.withInitial(named);
  ThreadLocal<String> method=ThreadLocal.withInitial(reference);
  ExecutorService pool=Executors.newSingleThreadExecutor();
  Future<String> checked=pool.submit(()-> {
   String name=Thread.currentThread().getName();
   Optional<String> absent=Optional.empty();
   return local.get().equals(prefix+name)+":"+source.get().equals(name)+":"+method.get().equals(name)+":"+supplier.get().equals(prefix+name)+":"+absent.orElseGet(supplier).equals(prefix+name);
  });
  String value=checked.get();pool.shutdown();pool.awaitTermination(10,TimeUnit.SECONDS);return value;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeThreadLocalNamedSupplier", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestNamedSupplierOracle(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
