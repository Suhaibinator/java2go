package parity.integerbranch.model;

public class WalkResult {
    private final int finalIndex;
    private final long hotBranches;
    private final long coldBranches;
    private final long accumulator;
    private final long signature;

    public WalkResult(int finalIndex, long hotBranches, long coldBranches,
                      long accumulator, long signature) {
        this.finalIndex = finalIndex;
        this.hotBranches = hotBranches;
        this.coldBranches = coldBranches;
        this.accumulator = accumulator;
        this.signature = signature;
    }

    public int finalIndex() {
        return this.finalIndex;
    }

    public long hotBranches() {
        return this.hotBranches;
    }

    public long coldBranches() {
        return this.coldBranches;
    }

    public long accumulator() {
        return this.accumulator;
    }

    public long signature() {
        return this.signature;
    }
}
