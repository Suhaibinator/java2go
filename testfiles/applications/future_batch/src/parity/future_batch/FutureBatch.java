package parity.future_batch;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;

public class FutureBatch {
    public static String run() throws Exception {
        ExecutorService pool = Executors.newFixedThreadPool(2);
        Callable<Integer> first = () -> 21;
        Future<Integer> a = pool.submit(first);
        Future<Integer> b = pool.submit(() -> 21);
        AtomicInteger count = new AtomicInteger();
        Future<?> marker = pool.submit(() -> { count.incrementAndGet(); });
        Future<String> label = pool.submit(() -> { count.incrementAndGet(); }, "saved");
        int sum = a.get() + b.get();
        marker.get();
        String result = sum + ":" + label.get() + ":" + count.get();
        Future<Integer> failed = pool.submit(() -> { throw new IllegalStateException("task failed"); });
        try { failed.get(); } catch (ExecutionException e) {
            result = result + ":" + e.getCause().getMessage();
        }
        Future<Integer> after = pool.submit(() -> 7);
        result = result + ":" + after.get();
        pool.shutdown();
        result = result + ":" + pool.awaitTermination(2, TimeUnit.SECONDS);
        result = result + ":" + pool.isShutdown() + ":" + pool.isTerminated();
        try { pool.submit(() -> 9); } catch (RejectedExecutionException e) {
            result = result + ":rejected";
        }
        return result;
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
