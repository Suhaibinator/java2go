package parity.integerbranch.sieve;

import parity.integerbranch.model.SieveResult;

public class PrimeSieve {
    public static SieveResult run(int limit) {
        boolean[] composite = new boolean[limit + 1];
        for (int candidate = 2; candidate * candidate <= limit; candidate++) {
            if (!composite[candidate]) {
                int multiple = candidate * candidate;
                while (multiple <= limit) {
                    composite[multiple] = true;
                    multiple = multiple + candidate;
                }
            }
        }

        int count = 0;
        int lastPrime = 0;
        long sum = 0L;
        long signature = 1099511628211L;
        for (int value = 2; value <= limit; value++) {
            if (!composite[value]) {
                count = count + 1;
                lastPrime = value;
                sum = sum + value;
                signature = signature * 16777619L
                        + (long) value * (count % 251 + 1);
            }
        }
        return new SieveResult(limit, count, lastPrime, sum, signature);
    }
}
