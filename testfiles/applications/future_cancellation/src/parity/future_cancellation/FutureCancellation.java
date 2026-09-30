package parity.future_cancellation;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicBoolean;

public class FutureCancellation {
    public static String run() throws Exception {
        ExecutorService pool = Executors.newSingleThreadExecutor();
        AtomicBoolean release = new AtomicBoolean(false);
        Future<Integer> running = pool.submit(() -> {
            while (!release.get()) { Thread.sleep(1); }
            return 11;
        });
        Future<Integer> queued = pool.submit(() -> 99);
        String result = "";
        try { queued.get(0, TimeUnit.NANOSECONDS); } catch (TimeoutException e) {
            result = "timeout";
        }
        result = result + ":" + queued.cancel(false) + ":" + queued.isCancelled() + ":" + queued.isDone();
        try { queued.get(); } catch (CancellationException e) { result = result + ":cancelled"; }
        result = result + ":" + queued.cancel(true);
        pool.shutdown();
        result = result + ":" + pool.awaitTermination(0, TimeUnit.MILLISECONDS);
        release.set(true);
        result = result + ":" + running.get(2, TimeUnit.SECONDS);
        result = result + ":" + pool.awaitTermination(2, TimeUnit.SECONDS);
        return result;
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
