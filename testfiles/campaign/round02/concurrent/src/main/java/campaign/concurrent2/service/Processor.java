package campaign.concurrent2.service;

import campaign.concurrent2.io.TrackedInput;
import campaign.concurrent2.model.Job;
import campaign.concurrent2.model.Result;
import java.io.ByteArrayOutputStream;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.concurrent.Callable;
import java.util.concurrent.CountDownLatch;
import org.apache.commons.codec.binary.Hex;

public final class Processor implements Callable<Result> {
    private static final class Context {
        private int uses;
        private String task;
    }

    private static final ThreadLocal<Context> CURRENT = ThreadLocal.withInitial(() -> new Context());

    private final Job job;
    private final Ledger ledger;
    private final CountDownLatch entered;
    private final CountDownLatch release;
    private final CountDownLatch done;

    public Processor(Job job, Ledger ledger, CountDownLatch entered,
                     CountDownLatch release, CountDownLatch done) {
        this.job = job;
        this.ledger = ledger;
        this.entered = entered;
        this.release = release;
        this.done = done;
    }

    @Override
    public Result call() throws Exception {
        Context context = CURRENT.get();
        if (context.uses != 0 || context.task != null) {
            throw new AssertionError("worker context leaked into " + job.id());
        }
        context.uses++;
        context.task = job.id();
        ledger.start(job.id(), context);
        try {
            if (entered != null) {
                entered.countDown();
                release.await();
            }
            byte[] payload = job.payload().getBytes(StandardCharsets.UTF_8);
            Result result;
            try (TrackedInput input = new TrackedInput(payload, ledger);
                 ByteArrayOutputStream buffer = new ByteArrayOutputStream()) {
                byte[] chunk = new byte[3];
                int length;
                while ((length = input.read(chunk, 0, chunk.length)) != -1) {
                    buffer.write(chunk, 0, length);
                }
                if (job.payload().trim().isEmpty()) {
                    throw new IllegalArgumentException("blank task body");
                }
                byte[] copied = buffer.toByteArray();
                String hex = Hex.encodeHexString(copied);
                if (!Arrays.equals(copied, Hex.decodeHex(hex))) {
                    throw new AssertionError("codec mismatch " + job.id());
                }
                result = new Result(job.id(), hex, copied.length);
            }
            ledger.succeed(job.id());
            return result;
        } catch (Exception problem) {
            ledger.fail(job.id());
            throw problem;
        } finally {
            CURRENT.remove();
            done.countDown();
        }
    }
}
