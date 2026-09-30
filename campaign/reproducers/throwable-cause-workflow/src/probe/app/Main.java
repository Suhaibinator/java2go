package probe.app;

import probe.cause.InheritedCause;
import probe.cause.NullTextCause;
import probe.cause.ThrowingCause;
import probe.state.Cleanup;
import probe.state.Trace;

public final class Main {
    private Main() {}

    private static String show(String text) {
        return text == null ? "<null>" : text;
    }

    private static String runOnWorker(int seed, int round, InheritedCause inherited,
                                      Trace inheritedTrace, NullTextCause nullText, Trace nullTrace,
                                      ThrowingCause throwing, Trace throwingTrace,
                                      RuntimeException conversionFailure,
                                      RuntimeException closeFailure) {
        StringBuilder out = new StringBuilder();
        Exception checked = new Exception(inherited);
        out.append("checked=").append(show(checked.getMessage()))
           .append(",sameCause=").append(checked.getCause() == inherited)
           .append(",string=").append(checked.toString()).append(';');

        RuntimeException unchecked = new RuntimeException(inherited);
        out.append("unchecked=").append(show(unchecked.getMessage()))
           .append(",sameCause=").append(unchecked.getCause() == inherited)
           .append(",string=").append(unchecked.toString()).append(';');

        Exception nullChecked = new Exception(nullText);
        RuntimeException nullUnchecked = new RuntimeException(nullText);
        out.append("nullChecked=").append(show(nullChecked.getMessage()))
           .append(",sameCause=").append(nullChecked.getCause() == nullText)
           .append(",string=").append(nullChecked.toString()).append(';');
        out.append("nullUnchecked=").append(show(nullUnchecked.getMessage()))
           .append(",sameCause=").append(nullUnchecked.getCause() == nullText)
           .append(",string=").append(nullUnchecked.toString()).append(';');

        boolean checkedCaught = false;
        int checkedSuppressed = -1;
        try (Cleanup cleanup = new Cleanup(throwingTrace, closeFailure)) {
            new Exception(throwing);
        } catch (RuntimeException failure) {
            checkedCaught = failure == conversionFailure;
            Throwable[] suppressed = failure.getSuppressed();
            checkedSuppressed = suppressed.length == 1 && suppressed[0] == closeFailure ? 1 : suppressed.length;
        }
        out.append("checkedThrow=").append(checkedCaught)
           .append(",suppressedClose=").append(checkedSuppressed).append(';');

        boolean runtimeCaught = false;
        try (Cleanup cleanup = new Cleanup(throwingTrace, null)) {
            new RuntimeException(throwing);
        } catch (RuntimeException failure) {
            runtimeCaught = failure == conversionFailure;
        }
        out.append("runtimeThrow=").append(runtimeCaught).append(';');

        Exception absentChecked = new Exception((Throwable) null);
        RuntimeException absentUnchecked = new RuntimeException((Throwable) null);
        out.append("absent=").append(show(absentChecked.getMessage())).append('/')
           .append(show(absentUnchecked.getMessage())).append('/')
           .append(absentChecked.getCause() == null).append('/')
           .append(absentUnchecked.getCause() == null).append(';');

        out.append("inherited{").append(inheritedTrace.snapshot()).append("};")
           .append("nullText{").append(nullTrace.snapshot()).append("};")
           .append("throwing{").append(throwingTrace.snapshot()).append('}');
        return "seed=" + seed + ",round=" + round + ":" + out;
    }

    public static void main(String[] args) throws InterruptedException {
        int seed = Integer.parseInt(args[0]);
        int rounds = 1 + seed % 3;
        for (int round = 0; round < rounds; round++) {
            final int index = round;
            Trace inheritedTrace = new Trace("inherited-" + seed + '-' + round);
            Trace nullTrace = new Trace("null-" + seed + '-' + round);
            Trace throwingTrace = new Trace("throwing-" + seed + '-' + round);
            InheritedCause inherited = new InheritedCause(inheritedTrace);
            NullTextCause nullText = new NullTextCause(nullTrace);
            RuntimeException conversionFailure = new IllegalStateException("convert-" + seed + '-' + round);
            RuntimeException closeFailure = new IllegalArgumentException("close-" + seed + '-' + round);
            ThrowingCause throwing = new ThrowingCause(throwingTrace, conversionFailure);
            String[] result = new String[1];
            Thread worker = new Thread(() -> result[0] = runOnWorker(seed, index,
                inherited, inheritedTrace, nullText, nullTrace, throwing, throwingTrace,
                conversionFailure, closeFailure), "worker-" + seed + '-' + round);
            worker.start();
            worker.join();
            System.out.println(result[0]);
        }
    }
}
