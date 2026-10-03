package campaign.concurrent.service;

import campaign.concurrent.model.Job;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Callable;
import java.util.concurrent.CountDownLatch;
import org.apache.commons.codec.binary.Hex;
import campaign.concurrent.model.Result;

public final class TaskProcessor implements Callable<Result> {
    private final Job job;
    private final Ledger ledger;
    private final CountDownLatch entered;
    private final CountDownLatch release;
    private final CountDownLatch done;

    public TaskProcessor(Job job, Ledger ledger, CountDownLatch entered,
                         CountDownLatch release, CountDownLatch done) {
        this.job = job;
        this.ledger = ledger;
        this.entered = entered;
        this.release = release;
        this.done = done;
    }

    @Override
    public Result call() throws Exception {
        ledger.start(job.id());
        try {
            if (entered != null) {
                entered.countDown();
                release.await();
            }
            String payload = job.payload();
            if (payload.trim().isEmpty()) {
                throw new IllegalArgumentException("blank payload");
            }
            String encoded = Hex.encodeHexString(payload.getBytes(StandardCharsets.UTF_8));
            String decoded = new String(Hex.decodeHex(encoded), StandardCharsets.UTF_8);
            if (!payload.equals(decoded)) {
                throw new AssertionError("codec round trip failed for " + job.id());
            }
            ledger.succeed(job.id());
            return new Result(job.id(), encoded);
        } catch (Exception failure) {
            ledger.fail(job.id());
            throw failure;
        } finally {
            done.countDown();
        }
    }
}
