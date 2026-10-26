package probe.join.app;

import java.util.ArrayList;
import java.util.ConcurrentModificationException;
import java.util.List;
import probe.join.source.LiveSource;
import probe.join.source.TrackedText;
import probe.join.state.Trace;

public final class Main {
    private Main() {}

    private static String plain(int seed) {
        Trace trace = new Trace();
        List<CharSequence> values = new ArrayList<>();
        TrackedText first = new TrackedText("plain-a", "A" + seed, trace, null, null, false);
        TrackedText second = new TrackedText("plain-b", "β" + seed, trace, null, null, false);
        values.add(first);
        values.add(second);
        String joined = String.join("|", new LiveSource(values, trace));
        return "plain=" + joined + ",calls=" + first.conversions() + '/' + second.conversions()
            + ",events=" + trace.snapshot();
    }

    private static String replacement(int seed) {
        Trace trace = new Trace();
        List<CharSequence> values = new ArrayList<>();
        TrackedText replacement = new TrackedText("new", "new-" + seed, trace, null, null, false);
        TrackedText original = new TrackedText("old", "old-" + seed, trace, null, null, false);
        TrackedText first = new TrackedText("mutator", "first-" + seed, trace,
            () -> {
                values.set(1, replacement);
                trace.add("set-index-1");
            }, null, false);
        values.add(first);
        values.add(original);
        String joined = String.join("|", new LiveSource(values, trace));
        return "replacement=" + joined + ",calls=" + first.conversions() + '/'
            + original.conversions() + '/' + replacement.conversions()
            + ",size=" + values.size() + ",events=" + trace.snapshot();
    }

    private static String sameSizeStructural(int seed) {
        Trace trace = new Trace();
        List<CharSequence> values = new ArrayList<>();
        TrackedText spare = new TrackedText("spare", "spare-" + seed, trace, null, null, false);
        TrackedText later = new TrackedText("later", "later-" + seed, trace, null, null, false);
        TrackedText first = new TrackedText("structural", "first-" + seed, trace,
            () -> {
                values.add(spare);
                trace.add("add-spare");
                values.remove(values.size() - 1);
                trace.add("remove-spare");
            }, null, false);
        values.add(first);
        values.add(later);
        String result;
        try {
            result = "joined:" + String.join("|", new LiveSource(values, trace));
        } catch (ConcurrentModificationException failure) {
            result = "caught:" + failure.getClass().getSimpleName();
        }
        return "sameSize=" + result + ",size=" + values.size()
            + ",calls=" + first.conversions() + '/' + later.conversions()
            + ",events=" + trace.snapshot();
    }

    private static String nullText(int seed) {
        Trace trace = new Trace();
        List<CharSequence> values = new ArrayList<>();
        TrackedText first = new TrackedText("null", "hidden-" + seed, trace, null, null, true);
        TrackedText second = new TrackedText("tail", "tail-" + seed, trace, null, null, false);
        values.add(first);
        values.add(second);
        String result;
        try {
            result = "joined:" + String.join("|", new LiveSource(values, trace));
        } catch (RuntimeException failure) {
            result = "caught:" + failure.getClass().getSimpleName();
        }
        return "nullText=" + result + ",calls=" + first.conversions() + '/'
            + second.conversions() + ",events=" + trace.snapshot();
    }

    private static String throwing(int seed) {
        Trace trace = new Trace();
        List<CharSequence> values = new ArrayList<>();
        RuntimeException marker = new IllegalStateException("convert-" + seed);
        TrackedText first = new TrackedText("head", "head-" + seed, trace, null, null, false);
        TrackedText failing = new TrackedText("throw", "hidden-" + seed, trace, null, marker, false);
        TrackedText tail = new TrackedText("tail", "tail-" + seed, trace, null, null, false);
        values.add(first);
        values.add(failing);
        values.add(tail);
        boolean sameMarker = false;
        try {
            String.join("|", new LiveSource(values, trace));
        } catch (RuntimeException failure) {
            sameMarker = failure == marker;
        } finally {
            trace.add("finally");
        }
        return "throwing=sameMarker:" + sameMarker + ",calls=" + first.conversions()
            + '/' + failing.conversions() + '/' + tail.conversions()
            + ",events=" + trace.snapshot();
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        System.out.println("seed=" + seed);
        System.out.println(plain(seed));
        System.out.println(replacement(seed));
        System.out.println(sameSizeStructural(seed));
        System.out.println(nullText(seed));
        System.out.println(throwing(seed));
    }
}
