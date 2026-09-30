package probe.forward.app;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;
import probe.forward.source.Forwarder;
import probe.forward.source.RawCursor;
import probe.forward.source.Token;
import probe.forward.state.Trace;

public final class Main {
    private Main() {}

    private static String consume(Iterator<CharSequence> stream, RawCursor cursor,
                                  int ignoredCount, Trace trace) {
        Object objectRead = stream.next();
        boolean objectSucceeded = objectRead instanceof Token;
        trace.record("object.returned=" + objectSucceeded);

        for (int i = 0; i < ignoredCount; i++) {
            stream.next();
            trace.record("ignored.returned#" + (i + 1));
        }

        boolean typedCastCaught = false;
        try {
            CharSequence typed = stream.next();
            trace.record("unexpected.typed=" + typed);
        } catch (ClassCastException failure) {
            typedCastCaught = true;
            trace.record("consumer.caught-ClassCastException@" + cursor.position());
        }

        CharSequence nullRead = stream.next();
        trace.record("typed.null=" + (nullRead == null));
        CharSequence tailRead = stream.next();
        trace.record("typed.tail=" + tailRead);
        boolean exhausted = !stream.hasNext();
        trace.record("exhausted=" + exhausted);

        return "objectSucceeded=" + objectSucceeded
            + ",typedCastCaught=" + typedCastCaught
            + ",nullSucceeded=" + (nullRead == null)
            + ",tail=" + tailRead
            + ",cursor=" + cursor.position()
            + ",exhausted=" + exhausted;
    }

    private static String run(int seed) {
        int ignoredCount = 1 + seed % 3;
        Trace trace = new Trace();
        List<Object> values = new ArrayList<>();
        values.add(new Token("object-" + seed));
        for (int i = 0; i < ignoredCount; i++) {
            values.add(new Token("ignored-" + seed + '-' + i));
        }
        values.add(new Token("typed-failure-" + seed));
        values.add(null);
        values.add("tail-" + seed);

        RawCursor cursor = new RawCursor(values, trace);
        Iterator<CharSequence> stream = Forwarder.<CharSequence>fromRaw(cursor, trace);
        String result = consume(stream, cursor, ignoredCount, trace);
        return "seed=" + seed + ",ignoredCount=" + ignoredCount
            + ',' + result + ",events=" + trace.snapshot();
    }

    public static void main(String[] args) {
        System.out.println(run(Integer.parseInt(args[0])));
    }
}
