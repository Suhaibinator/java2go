package parity.integerbranch.model;

public class SieveResult {
    private final int limit;
    private final int primeCount;
    private final int lastPrime;
    private final long primeSum;
    private final long signature;

    public SieveResult(int limit, int primeCount, int lastPrime,
                       long primeSum, long signature) {
        this.limit = limit;
        this.primeCount = primeCount;
        this.lastPrime = lastPrime;
        this.primeSum = primeSum;
        this.signature = signature;
    }

    public int limit() {
        return this.limit;
    }

    public int primeCount() {
        return this.primeCount;
    }

    public int lastPrime() {
        return this.lastPrime;
    }

    public long primeSum() {
        return this.primeSum;
    }

    public long signature() {
        return this.signature;
    }
}
