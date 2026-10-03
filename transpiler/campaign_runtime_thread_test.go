package transpiler

import (
	"testing"
)

func TestCampaignRuntimeCurrentThread(t *testing.T) {
	const source = `import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
public class CampaignRuntimeCurrentThread {
    public static String run() throws Exception {
        Thread caller = Thread.currentThread();
        Thread named = new Thread("explicit-name");
        Thread namedTask = new Thread(() -> {}, "task-name");
        Thread[] seen = new Thread[1];
        Thread worker = new Thread(() -> { seen[0] = Thread.currentThread(); });
        worker.run();
        boolean direct = seen[0] == caller;
        worker.start();
        worker.join();
        boolean ownIdentity = seen[0] == worker;
        boolean differentName = !worker.getName().equals(caller.getName());
        ExecutorService pool = Executors.newSingleThreadExecutor();
        Future<String> first = pool.submit(() -> Thread.currentThread().getName());
        Future<String> second = pool.submit(() -> Thread.currentThread().getName());
        String firstName = first.get();
        boolean workerName = !firstName.equals(caller.getName());
        boolean sameWorker = firstName.equals(second.get());
        pool.shutdown();
        boolean ended = pool.awaitTermination(10, TimeUnit.SECONDS);
        return direct + ":" + ownIdentity + ":" + differentName + ":" + workerName + ":" + sameWorker + ":" + ended
            + ":" + named.getName() + ":" + namedTask.getName() + ":" + caller.isAlive();
    }
}`
	verifyCanonicalStringStreamOracle(t, "CampaignRuntimeCurrentThread", source)
}
