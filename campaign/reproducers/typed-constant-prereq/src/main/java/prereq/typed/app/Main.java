package prereq.typed.app;

import prereq.typed.api.Base;
import prereq.typed.api.Trace;
import prereq.typed.mid.Derived;
import prereq.typed.mid.Flow;
import prereq.typed.shadow.String;

public final class Main {
    private static final int SHIFTED = 9;

    private Main() {}

    private static Derived qualifiedDerived(int seed) {
        Trace.add("Q" + seed);
        return null;
    }

    private static String qualifiedShadow(int seed) {
        Trace.add("H" + seed);
        return null;
    }

    public static void main(java.lang.String[] args) {
        int seed = Integer.parseInt(args[0]);
        boolean constantReferences = Base.FOLDED == "base:1:-128"
                && Derived.FOLDED == "base:1:-128:derived:2:2"
                && Flow.CROSS == "base:1:-128:derived:2:2:shadow";
        System.out.println("seed=" + seed + ",constants=" + constantReferences
                + ",before=" + Trace.snapshot());

        int inherited = qualifiedDerived(seed).NARROW;
        int hidden = qualifiedDerived(seed).SHIFTED;
        java.lang.String shadowConstant = qualifiedShadow(seed).CONSTANT;
        System.out.println("qualified=" + inherited + "," + hidden + ","
                + shadowConstant + ",trace=" + Trace.snapshot());

        final int SHIFTED = Base.SHIFTED;
        final byte narrowed = (byte) (127 + SHIFTED);
        final char letter = (char) (65536 + 65);
        final int unsigned = (-1) >>> (32 - SHIFTED);
        final long longShift = 1L << (64 + SHIFTED);
        final java.lang.String folded = Flow.CROSS + ":" + narrowed + ":"
                + letter + ":" + unsigned + ":" + longShift;
        java.lang.String dynamic = folded + ":" + seed;
        java.lang.String dynamicAgain = folded + ":" + seed;
        System.out.println("binding=" + SHIFTED + ",field=" + Main.SHIFTED
                + ",narrow=" + narrowed + ",char=" + (int) letter
                + ",unsigned=" + unsigned + ",long=" + longShift
                + ",folded=" + (folded == "base:1:-128:derived:2:2:shadow:-128:A:1:2")
                + ",dynamicSame=" + (dynamic == dynamicAgain));

        String instance = qualifiedShadow(seed).INSTANCE;
        System.out.println("instance=" + instance.value() + ",trace=" + Trace.snapshot());
        java.lang.String touched = Derived.touch();
        java.lang.String flow = Flow.touch();
        System.out.println("touch=" + (touched == Derived.FOLDED) + ","
                + (flow == Flow.CROSS) + ",trace=" + Trace.snapshot());
        System.out.println("source=" + Flow.shadowValue(seed) + ",trace=" + Trace.snapshot());
    }
}
