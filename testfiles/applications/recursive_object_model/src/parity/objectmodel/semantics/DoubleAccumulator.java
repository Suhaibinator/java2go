package parity.objectmodel.semantics;

public final class DoubleAccumulator extends RecursiveAccumulator {
    protected int unit() {
        return 2;
    }

    public int throughSuper(int depth) {
        return super.accumulate(depth);
    }
}
