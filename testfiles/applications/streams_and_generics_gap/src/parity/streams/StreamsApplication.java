package parity.streams;

import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;

public class StreamsApplication {
    public static void main(String[] args) {
        MathProcessor<Double> processor = new MathProcessor<>();
        List<Double> inputs = Arrays.asList(1.5, 2.0, 3.14159, 4.0, -1.0, 5.5);
        List<Double> results = processor.process(inputs);
        System.out.println("Results: " + results);
    }
}
