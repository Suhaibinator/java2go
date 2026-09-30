package parity.integerbranch.model;

public class SortSearchResult {
    private final int length;
    private final int rounds;
    private final long queries;
    private final long hits;
    private final long misses;
    private final long comparisons;
    private final long signature;

    public SortSearchResult(int length, int rounds, long queries, long hits,
                            long misses, long comparisons, long signature) {
        this.length = length;
        this.rounds = rounds;
        this.queries = queries;
        this.hits = hits;
        this.misses = misses;
        this.comparisons = comparisons;
        this.signature = signature;
    }

    public int length() {
        return this.length;
    }

    public int rounds() {
        return this.rounds;
    }

    public long queries() {
        return this.queries;
    }

    public long hits() {
        return this.hits;
    }

    public long misses() {
        return this.misses;
    }

    public long comparisons() {
        return this.comparisons;
    }

    public long signature() {
        return this.signature;
    }
}
