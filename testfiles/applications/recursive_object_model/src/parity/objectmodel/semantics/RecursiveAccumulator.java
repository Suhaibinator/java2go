package parity.objectmodel.semantics;

public class RecursiveAccumulator {
    protected int unit() {
        return 1;
    }

    public int accumulate(int depth) {
        return depth == 0 ? unit() : unit() + accumulate(depth - 1);
    }
}
