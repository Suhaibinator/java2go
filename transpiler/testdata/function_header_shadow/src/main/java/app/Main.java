package app;
import java.util.function.Function;
public final class Main {
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Length length = new Length(seed);
        Function<String,Integer> throughInterface = length;
        Object stored = throughInterface;
        Function<String,Integer> recovered = (Function<String,Integer>) stored;
        int first = throughInterface.apply("x" + seed);
        int second = recovered.apply("seed-" + seed);
        int third = recovered.apply("");
        System.out.println(seed + ":" + first + ":" + second + ":" + third + ":"
            + length.calls() + ":" + length.total() + ":"
            + java.util.function.Function.class.isInstance(stored) + ":" + length.bodyMarker());
    }
}
