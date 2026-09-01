import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.stream.Collectors;

// Java's Double.compare is a total order that differs from the primitive `<`:
// NaN sorts after every other value (so it moves during a sort), and -0.0 sorts
// before 0.0. Every ordering path must agree on that, and with each other.
public class FloatOrdering {
    public static void main(String[] args) {
        System.out.println(Double.compare(Double.NaN, 1.0));
        System.out.println(Double.compare(1.0, Double.NaN));
        System.out.println(Double.compare(Double.NaN, Double.NaN));

        List<Double> values = new ArrayList<Double>();
        values.add(3.0);
        values.add(Double.NaN);
        values.add(1.0);

        // Collections.sort and Stream.sorted must produce the same order.
        List<Double> sorted = new ArrayList<Double>();
        sorted.addAll(values);
        Collections.sort(sorted);
        System.out.println(sorted);
        System.out.println(values.stream().sorted().collect(Collectors.toList()));

        // ...and so must min/max.
        System.out.println("max " + Collections.max(values));
        System.out.println("min " + Collections.min(values));

        // A comparator built from the natural ordering agrees too.
        List<Double> byComparator = new ArrayList<Double>();
        byComparator.addAll(values);
        byComparator.sort((a, b) -> Double.compare(a, b));
        System.out.println(byComparator);

        // Arrays.sort on a primitive double array uses the same total order.
        double[] array = new double[3];
        array[0] = 3.0;
        array[1] = Double.NaN;
        array[2] = 1.0;
        Arrays.sort(array);
        System.out.println(Arrays.toString(array));
    }
}
