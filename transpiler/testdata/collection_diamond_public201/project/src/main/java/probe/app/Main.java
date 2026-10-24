package probe.app;

import java.util.ArrayList;
import java.util.Collection;
import probe.flow.Flow;
import probe.model.State;
import probe.shadow.NamespaceChecks;

public final class Main {
    private static void require(boolean condition, String label) {
        if (!condition) throw new AssertionError(label);
    }
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Collection<String> values = new ArrayList<>();
        values.add("head-" + seed);
        values.add("tail-" + (seed * 3 + 1));
        State state = new State(seed);
        int before = Flow.digest(values, state);
        Collection<String> alias = Flow.alias(values);
        alias.add("echo-" + before);
        int after = Flow.digest(values, state);
        String first = values.iterator().next();
        int namespace = NamespaceChecks.run(seed);
        require(alias == values && values.size() == 3, "ordinary Collection alias writes");
        require(first.equals("head-" + seed), "typed first read");
        require(state.count() == 5 && state.last().equals("echo-" + before), "cross-package seeded state");
        require(after == state.checksum() && namespace == seed * 3 + 18, "digest and source namespace constructors");
        System.out.println("ordinary=" + seed + ":" + state.count() + ":" + after + ":" + first + ":" + state.last() + ":" + namespace);
    }
}
