package probe.bridge.app;

import java.util.ArrayList;
import java.util.List;
import probe.bridge.domain.DerivedSequence;
import probe.bridge.domain.SequenceSource;
import probe.bridge.state.Trace;

public final class Main {
    private Main() {}

    @SuppressWarnings({"rawtypes", "unchecked"})
    private static String joinThroughRaw(String delimiter, SequenceSource source) {
        Iterable raw = source;
        return String.join(delimiter, (Iterable<? extends CharSequence>) raw);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    private static String run(int seed) {
        Trace trace = new Trace();
        List<CharSequence> values = new ArrayList<>();
        int pollutedIndex = seed % 3 == 1 ? 2 : 1;
        boolean[] polluted = {false};
        DerivedSequence first = new DerivedSequence("first", "A" + seed, trace, () -> {
            if (!polluted[0]) {
                List raw = values;
                raw.set(pollutedIndex, new Object());
                polluted[0] = true;
                trace.record("raw-set-object-at-" + pollutedIndex);
            }
        });
        DerivedSequence second = new DerivedSequence("second", "B" + seed, trace, null);
        DerivedSequence third = new DerivedSequence("third", "C" + seed, trace, null);
        values.add(first);
        values.add(second);
        values.add(third);
        SequenceSource source = new SequenceSource(values, trace);

        boolean classCast = false;
        try {
            joinThroughRaw("|", source);
        } catch (ClassCastException failure) {
            classCast = true;
            trace.record("caught-ClassCastException");
        }

        values.set(pollutedIndex, pollutedIndex == 1 ? second : third);
        trace.record("restore-index-" + pollutedIndex);
        String recovered = joinThroughRaw("|", source);
        return "seed=" + seed + ",delayedCast=" + classCast
            + ",recovered=" + recovered
            + ",counts=" + first.conversions() + '/' + second.conversions() + '/'
            + third.conversions() + ",size=" + values.size()
            + ",events=" + trace.snapshot();
    }

    public static void main(String[] args) {
        System.out.println(run(Integer.parseInt(args[0])));
    }
}
