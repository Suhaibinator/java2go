package campaign.concurrent2.app;

import campaign.concurrent2.io.PlanIO;
import campaign.concurrent2.model.Job;
import campaign.concurrent2.model.Result;
import campaign.concurrent2.service.Ledger;
import campaign.concurrent2.service.Processor;
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
        if (!condition) { throw new AssertionError(message); }
    }

    private static void await(CountDownLatch latch) throws Exception {
        require(latch.await(10, TimeUnit.SECONDS), "latch timeout");
    }

    private static void shutdown(ExecutorService executor) throws Exception {
        executor.shutdown();
        require(executor.awaitTermination(10, TimeUnit.SECONDS), "executor timeout");
    }

    private static Result[] serialPhase(List<Job> plan) throws Exception {
        Ledger ledger = new Ledger();
        ExecutorService executor = Executors.newSingleThreadExecutor();
        CountDownLatch entered = new CountDownLatch(1);
        CountDownLatch release = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(3);
        Job first = new Job("s-" + plan.get(0).id(), plan.get(0).payload());
        Job cancelled = new Job("s-cancel", "never run");
        Job bad = new Job("s-bad", "  ");
        Job recovery = new Job("s-" + plan.get(1).id(), plan.get(1).payload());
        try {
            ledger.queue(first.id());
            Future<Result> firstFuture = executor.submit(new Processor(first, ledger, entered, release, done));
            await(entered);
            ledger.queue(cancelled.id());
            Future<Result> cancelledFuture = executor.submit(new Processor(cancelled, ledger, null, null, done));
            ledger.queue(bad.id());
            Future<Result> badFuture = executor.submit(new Processor(bad, ledger, null, null, done));
            ledger.queue(recovery.id());
            Future<Result> recoveryFuture = executor.submit(new Processor(recovery, ledger, null, null, done));
            require(cancelledFuture.cancel(false), "queued cancellation rejected");
            ledger.cancel(cancelled.id());
            release.countDown();
            Result firstResult = firstFuture.get(10, TimeUnit.SECONDS);
            String failure = "none";
            try {
                badFuture.get(10, TimeUnit.SECONDS);
            } catch (ExecutionException expected) {
                failure = expected.getCause().getClass().getSimpleName();
            }
            require("IllegalArgumentException".equals(failure), "wrong failure " + failure);
            Result recoveryResult = recoveryFuture.get(10, TimeUnit.SECONDS);
            await(done);
            shutdown(executor);
            require(cancelledFuture.isCancelled(), "cancelled future changed");
            ledger.assertComplete(3, new String[] { first.id(), recovery.id() }, bad.id(), cancelled.id());
            return new Result[] { firstResult, recoveryResult };
        } finally {
            release.countDown();
            executor.shutdownNow();
        }
    }

    private static List<Result> parallelPhase(List<Job> plan) throws Exception {
        Ledger ledger = new Ledger();
        ExecutorService executor = Executors.newFixedThreadPool(3);
        CountDownLatch entered = new CountDownLatch(3);
        CountDownLatch release = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(plan.size());
        List<Future<Result>> futures = new ArrayList<>();
        try {
            for (int i = 0; i < plan.size(); i++) {
                Job original = plan.get(i);
                Job job = new Job("p-" + original.id(), original.payload());
                ledger.queue(job.id());
                futures.add(executor.submit(new Processor(job, ledger,
                        i < 3 ? entered : null, i < 3 ? release : null, done)));
            }
            await(entered);
            release.countDown();
            List<Result> ordered = new ArrayList<>();
            for (Future<Result> future : futures) {
                ordered.add(future.get(10, TimeUnit.SECONDS));
            }
            await(done);
            shutdown(executor);
            String[] ids = new String[plan.size()];
            for (int i = 0; i < plan.size(); i++) {
                ids[i] = "p-" + plan.get(i).id();
                require(ids[i].equals(ordered.get(i).id()), "result order");
            }
            ledger.assertComplete(plan.size(), ids, null, null);
            return ordered;
        } finally {
            release.countDown();
            executor.shutdownNow();
        }
    }

    public static void main(String[] args) throws Exception {
        int seed = Integer.parseInt(args[0]);
        List<Job> plan = PlanIO.read(seed);
        Result[] serial = serialPhase(plan);
        List<Result> parallel = parallelPhase(plan);
        StringBuilder output = new StringBuilder();
        output.append("seed=").append(seed).append('\n');
        output.append("serial=").append(serial[0].line()).append('\n');
        output.append("cancelled=s-cancel\nfailed=s-bad:IllegalArgumentException\n");
        output.append("recovery=").append(serial[1].line()).append('\n');
        int bytes = serial[0].byteCount() + serial[1].byteCount();
        for (Result result : parallel) {
            output.append("parallel=").append(result.line()).append('\n');
            bytes += result.byteCount();
        }
        output.append("invariants=7 success,1 failure,1 cancel; 8 streams closed; fresh worker contexts\n");
        PlanIO.write(output.toString());
        System.out.println("seed=" + seed + " jobs=" + plan.size() + " bytes=" + bytes + " file=results.txt");
    }
}
