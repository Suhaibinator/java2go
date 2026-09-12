package parity.thread_lifecycle;
import java.util.concurrent.atomic.AtomicBoolean;
public class ThreadLifecycle {
    public static String run() throws Exception {
        Thread thread = new Thread(() -> {});
        thread.join();
        String result = "" + thread.isAlive();
        thread.start();
        thread.join();
        result = result + ":" + thread.isAlive();
        try { thread.start(); } catch (IllegalThreadStateException e) { result = result + ":restarted"; }
        AtomicBoolean release = new AtomicBoolean(false);
        Thread blocked = new Thread(() -> { while (!release.get()) { } });
        blocked.start();
        blocked.join(1);
        result = result + ":" + blocked.isAlive();
        release.set(true);
        blocked.join();
        return result + ":" + blocked.isAlive();
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
