package parity.integerbranch.sorting;

import java.util.Arrays;

import parity.integerbranch.model.SortSearchResult;

public class DeterministicSortSearch {
    public static SortSearchResult run(int length, int rounds,
                                       int queriesPerRound, int seed) {
        long queries = 0L;
        long hits = 0L;
        long misses = 0L;
        long comparisons = 0L;
        long signature = 7809847782465536322L;

        for (int round = 0; round < rounds; round++) {
            int[] values = new int[length];
            int generated = seed ^ (round * 1103515245);
            for (int index = 0; index < values.length; index++) {
                generated = generated * 1664525 + 1013904223;
                values[index] = generated ^ (index * 214013 + round * 2531011);
            }

            Arrays.sort(values);
            for (int index = 0; index < values.length; index++) {
                signature = signature * 1000003L
                        + (long) values[index] * (index % 257 + 1);
            }

            int mask = values.length - 1;
            for (int query = 0; query < queriesPerRound; query++) {
                int selector = (query * 104729 + round * 8191) & mask;
                int target;
                if ((query & 3) != 0) {
                    target = values[selector];
                } else {
                    target = query * 1103515245 ^ round * 12345 ^ seed;
                }

                long encoded = binarySearch(values, target);
                int foundIndex = (int) encoded;
                long probeComparisons = encoded >>> 32;
                queries = queries + 1L;
                comparisons = comparisons + probeComparisons;
                if (foundIndex >= 0) {
                    hits = hits + 1L;
                    signature = signature * 1000033L
                            + (long) values[foundIndex] * 31L + foundIndex;
                } else {
                    misses = misses + 1L;
                    signature = signature * 1000037L
                            ^ ((long) target * 17L + probeComparisons);
                }
            }
        }

        return new SortSearchResult(length, rounds, queries, hits, misses,
                comparisons, signature);
    }

    private static long binarySearch(int[] values, int target) {
        int low = 0;
        int high = values.length - 1;
        int comparisons = 0;
        while (low <= high) {
            int middle = (low + high) >>> 1;
            int value = values[middle];
            comparisons = comparisons + 1;
            if (value < target) {
                low = middle + 1;
            } else if (value > target) {
                high = middle - 1;
            } else {
                return ((long) comparisons << 32) | ((long) middle & 4294967295L);
            }
        }
        return ((long) comparisons << 32) | 4294967295L;
    }
}
