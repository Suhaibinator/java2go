import java.lang.reflect.Method;

/** Instrumentation only: invoke the same deterministic main repeatedly in one JVM. */
public final class RepeatMain {
    public static void main(String[] args) throws Exception {
        Method main = Class.forName(args[0]).getMethod("main", String[].class);
        int warmups = Integer.parseInt(args[1]);
        int samples = Integer.parseInt(args[2]);
        String[] workloadArgs = new String[0];
        for (int index = -warmups; index < samples; index++) {
            long start = System.nanoTime();
            main.invoke(null, (Object) workloadArgs);
            long elapsed = System.nanoTime() - start;
            System.err.println("PERF\t" + index + "\t" + elapsed);
        }
    }
}
