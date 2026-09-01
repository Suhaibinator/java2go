package parity.integerbranch.model;

public class SearchResult {
    private final int variant;
    private final long solutions;
    private final long nodes;
    private final long branches;
    private final long deadEnds;
    private final long signature;

    public SearchResult(int variant, long solutions, long nodes, long branches,
                        long deadEnds, long signature) {
        this.variant = variant;
        this.solutions = solutions;
        this.nodes = nodes;
        this.branches = branches;
        this.deadEnds = deadEnds;
        this.signature = signature;
    }

    public int variant() {
        return this.variant;
    }

    public long solutions() {
        return this.solutions;
    }

    public long nodes() {
        return this.nodes;
    }

    public long branches() {
        return this.branches;
    }

    public long deadEnds() {
        return this.deadEnds;
    }

    public long signature() {
        return this.signature;
    }
}
