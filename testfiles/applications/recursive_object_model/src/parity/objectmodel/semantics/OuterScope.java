package parity.objectmodel.semantics;

public final class OuterScope {
    private final int seed;

    public OuterScope(int seed) {
        this.seed = seed;
    }

    public final class Resolver {
        public int descend(int depth) {
            return depth == 0 ? seed : 1 + descend(depth - 1);
        }
    }

    public int resolve(int depth) {
        return new Resolver().descend(depth);
    }
}
