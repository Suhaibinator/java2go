package parity.callable_forms;
import java.util.concurrent.*;
class CallableJob implements Callable<Integer> {
    public Integer call() { return 19; }
}
public class CallableForms {
    public static synchronized Integer answer() { return 13; }
    public static String run() throws Exception {
        ExecutorService pool = Executors.newSingleThreadExecutor();
        Callable<Integer> referenced = CallableForms::answer;
        Callable<Integer> anonymous = new Callable<Integer>() {
            public Integer call() { return 17; }
        };
        Future<Integer> a = pool.submit(referenced);
        Future<Integer> b = pool.submit(anonymous);
        Future<Integer> c = pool.submit(CallableForms::answer);
        CallableJob job = new CallableJob();
        Future<Integer> d = pool.submit(job);
        String result = a.get() + ":" + b.get() + ":" + c.get() + ":" + referenced.call() + ":" + anonymous.call() + ":" + d.get();
        pool.shutdown();
        pool.awaitTermination(1, TimeUnit.SECONDS);
        return result;
    }
    public static void main(String[] args) throws Exception { System.out.println(run()); }
}
