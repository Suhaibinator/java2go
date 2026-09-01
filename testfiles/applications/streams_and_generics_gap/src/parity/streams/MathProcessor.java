package parity.streams;

import java.util.List;
import java.util.stream.Collectors;

public class MathProcessor<T extends Number> {
    public List<Double> process(List<T> inputs) {
        return inputs.stream()
            .filter(n -> n.doubleValue() > 0)
            .map(n -> Math.sqrt(n.doubleValue()) * Math.PI)
            .sorted()
            .collect(Collectors.toList());
    }
}
