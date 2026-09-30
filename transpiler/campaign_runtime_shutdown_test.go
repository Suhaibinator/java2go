package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeShutdownNow(t *testing.T) {
	const source = `import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
public class CampaignRuntimeShutdownNow {
    public static String run() throws Exception {
        ExecutorService pool = Executors.newSingleThreadExecutor();
        CountDownLatch ready = new CountDownLatch(1);
        CountDownLatch wait = new CountDownLatch(1);
        CountDownLatch open = new CountDownLatch(0);
        AtomicInteger interrupted = new AtomicInteger(0);
        AtomicInteger ran = new AtomicInteger(0);
        pool.execute(() -> {
            ready.countDown();
            try { wait.await(); }
            catch (InterruptedException expected) {
                interrupted.incrementAndGet();
                try { open.await(); interrupted.incrementAndGet(); }
                catch (InterruptedException uncleared) { interrupted.set(-1); }
            }
        });
        ready.await();
        Runnable direct = () -> { ran.addAndGet(1); };
        Runnable submitted = () -> { ran.addAndGet(10); };
        pool.execute(direct);
        Future<?> future = pool.submit(submitted);
        List<Runnable> pending = pool.shutdownNow();
        boolean ended = pool.awaitTermination(10, TimeUnit.SECONDS);
        boolean exact = pending.get(0) == direct;
        boolean wrapper = pending.get(1) != submitted;
        boolean untouched = !future.isDone() && !future.isCancelled() && ran.get() == 0;
        pending.get(0).run();
        pending.get(1).run();
        return pending.size() + ":" + ended + ":" + exact + ":" + wrapper + ":" + untouched
            + ":" + interrupted.get() + ":" + ran.get() + ":" + future.isDone()
            + ":" + pool.shutdownNow().size();
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeShutdownNow", source)
	t.Logf("JDK shutdown oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestShutdownOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
