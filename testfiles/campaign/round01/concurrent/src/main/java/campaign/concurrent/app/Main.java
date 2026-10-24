package campaign.concurrent.app;

import campaign.concurrent.model.Job;
import campaign.concurrent.model.Result;
import campaign.concurrent.service.Ledger;
import campaign.concurrent.service.TaskProcessor;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

public final class Main {
    private Main() {}

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    private static void await(CountDownLatch latch) throws InterruptedException {
        require(latch.await(10, TimeUnit.SECONDS), "latch timed out");
    }

    private static void shutdown(ExecutorService executor) throws InterruptedException {
        executor.shutdown();
        require(executor.awaitTermination(10, TimeUnit.SECONDS), "executor did not terminate");
    }

    private static Job job(int seed, int index) {
        String id = "j" + index;
        String payload = seed + "|π|" + (char) ('A' + index) + "|" + (seed * (index + 3));
        return new Job(id, payload);
    }

    private static String cancellationPhase(int seed) throws Exception {
        Ledger ledger = new Ledger();
        ExecutorService executor = Executors.newSingleThreadExecutor();
        CountDownLatch entered = new CountDownLatch(1);
        CountDownLatch release = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(2);
        Job first = job(seed, 0);
        Job cancelled = new Job("cancel", "should never run");
        Job failed = new Job("fail", "  ");
        try {
            ledger.queue(first.id());
            Future<Result> firstFuture = executor.submit(
                    new TaskProcessor(first, ledger, entered, release, done));
            await(entered);
            ledger.queue(cancelled.id());
            Future<Result> cancelledFuture = executor.submit(
                    new TaskProcessor(cancelled, ledger, null, null, done));
            ledger.queue(failed.id());
            Future<Result> failedFuture = executor.submit(
                    new TaskProcessor(failed, ledger, null, null, done));
            require(cancelledFuture.cancel(false), "queued cancellation rejected");
            ledger.cancel(cancelled.id());
            release.countDown();
            Result firstResult = firstFuture.get(10, TimeUnit.SECONDS);
            require(first.id().equals(firstResult.id()), "wrong first result");
            String failureType = "missing";
            try {
                failedFuture.get(10, TimeUnit.SECONDS);
            } catch (ExecutionException expected) {
                failureType = expected.getCause().getClass().getSimpleName();
            }
            await(done);
            shutdown(executor);
            require(cancelledFuture.isCancelled(), "future did not stay cancelled");
            ledger.assertState(first.id(), "SUCCEEDED");
            ledger.assertState(cancelled.id(), "CANCELLED");
            ledger.assertState(failed.id(), "FAILED");
            ledger.assertCounts(2, 2);
            require("IllegalArgumentException".equals(failureType), "wrong failure: " + failureType);
            return firstResult.encoded() + ":CANCELLED:" + failureType;
        } finally {
            release.countDown();
            executor.shutdownNow();
        }
    }

    private static String parallelPhase(int seed) throws Exception {
        Ledger ledger = new Ledger();
        ExecutorService executor = Executors.newFixedThreadPool(3);
        CountDownLatch entered = new CountDownLatch(3);
        CountDownLatch release = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(6);
        List<Future<Result>> futures = new ArrayList<>();
        try {
            for (int i = 1; i <= 6; i++) {
                Job current = job(seed, i);
                ledger.queue(current.id());
                futures.add(executor.submit(new TaskProcessor(current, ledger,
                        i <= 3 ? entered : null, i <= 3 ? release : null, done)));
            }
            await(entered);
            release.countDown();
            StringBuilder output = new StringBuilder();
            for (int i = 0; i < futures.size(); i++) {
                Result result = futures.get(i).get(10, TimeUnit.SECONDS);
                String expectedId = "j" + (i + 1);
                require(expectedId.equals(result.id()), "result order changed");
                if (i > 0) {
                    output.append(',');
                }
                output.append(result.id()).append('=').append(result.encoded());
            }
            await(done);
            shutdown(executor);
            for (int i = 1; i <= 6; i++) {
                ledger.assertState("j" + i, "SUCCEEDED");
            }
            ledger.assertCounts(6, 6);
            return output.toString();
        } finally {
            release.countDown();
            executor.shutdownNow();
        }
    }

    public static void main(String[] args) throws Exception {
        int seed = Integer.parseInt(args[0]);
        System.out.println("seed=" + seed);
        System.out.println("probe=" + cancellationPhase(seed));
        System.out.println("batch=" + parallelPhase(seed));
        System.out.println("invariants=7 worker-owned successes; 1 worker-owned failure; 1 queued cancellation; terminated");
    }
}
