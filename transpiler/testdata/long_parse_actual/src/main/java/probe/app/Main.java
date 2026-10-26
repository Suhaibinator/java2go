package probe.app;
import probe.shadow.IOException;
import probe.shadow.Long;
import probe.shadow.Paths;
import probe.state.Journal;
public final class Main {
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        long parsed = probe.parse.LongFlow.run(seed);
        int pathState = probe.path.PathsFlow.run(seed);
        probe.error.IOFlow.run(seed);
        String message = new String(new char[]{'s', 0, '\ud800', '\udc00'});
        probe.error.Signal cause = new probe.error.Signal(seed);
        IOException[] shadows = {new IOException(), new IOException(message), new IOException((Throwable) cause), new IOException(message, cause)};
        System.out.println("shadow:" + IOException.calls() + ":" + (shadows[1].message() == message) + ":" + (shadows[2].cause() == cause) + ":" + (shadows[3].message() == message) + ":" + (shadows[3].cause() == cause) + ":" + Journal.units(shadows[1].message()));
        System.out.println("source:parse-path:" + Long.parseLong(message, seed) + ":" + Long.calls() + ":" + (Paths.get(message, (String[]) null) == message) + ":" + Paths.calls());
        System.out.println("state:" + parsed + ":" + pathState + ":" + seed);
    }
}
