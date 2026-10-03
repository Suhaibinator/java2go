package transpiler

import (
	"fmt"
	"testing"
)

// The concurrent Codec fixture uses latches to ensure all worker slots are
// occupied before releasing work, and to distinguish execution from queuing.
func TestCampaignRuntimeCountDownLatch(t *testing.T) {
	const source = `import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
public class CampaignRuntimeLatch {
    public static String run() throws Exception {
        CountDownLatch release = new CountDownLatch(2);
        CountDownLatch finished = new CountDownLatch(1);
        AtomicInteger observed = new AtomicInteger(0);
        Thread worker = new Thread(() -> {
            try {
                release.await();
                observed.set(7);
                finished.countDown();
            } catch (InterruptedException failure) { throw new RuntimeException("interrupted"); }
        });
        worker.start();
        boolean early = release.await(0, TimeUnit.NANOSECONDS);
        release.countDown();
        long halfway = release.getCount();
        release.countDown();
        boolean completed = finished.await(10, TimeUnit.SECONDS);
        worker.join();
        release.countDown();
        boolean negativeTimeout = release.await(-1, TimeUnit.SECONDS);
        boolean zeroLatch = new CountDownLatch(0).await(0, TimeUnit.SECONDS);
        String negativeCount = "missing";
        try { new CountDownLatch(-1); }
        catch (IllegalArgumentException expected) { negativeCount = "rejected"; }
        String nullUnit = "missing";
        try { release.await(1, null); }
        catch (NullPointerException expected) { nullUnit = "rejected"; }
        return early + ":" + halfway + ":" + completed + ":" + observed.get()
            + ":" + release.getCount() + ":" + negativeTimeout + ":" + zeroLatch
            + ":" + negativeCount + ":" + nullUnit;
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeLatch", source)
	generated := renderGoFileFromJava(t, source)
	t.Logf("JDK latch oracle: %s", want)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    "unicode/utf16"
)
func TestLatchOracle(t *testing.T) {
    got := Run()
    if got == nil {
        t.Fatal("Run() returned null")
    }
    const want = %q
    if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune(want))) {
        t.Fatalf("JVM %%q != generated Go UTF16 %%v", want, units)
    }
}`, want))
}
