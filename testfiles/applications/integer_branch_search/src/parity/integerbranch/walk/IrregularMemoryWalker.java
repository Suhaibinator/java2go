package parity.integerbranch.walk;

import parity.integerbranch.model.WalkResult;

public class IrregularMemoryWalker {
    public static WalkResult run(int tableBits, int steps, int seed) {
        int tableSize = 1 << tableBits;
        int mask = tableSize - 1;
        int[] next = new int[tableSize];
        int[] values = new int[tableSize];

        int generated = seed;
        for (int i = 0; i < tableSize; i++) {
            generated = generated * 1664525 + 1013904223;
            values[i] = generated ^ (i * 1103515245);
            next[i] = (i * 1048573 + 8191) & mask;
        }

        int index = seed & mask;
        long hotBranches = 0L;
        long coldBranches = 0L;
        long accumulator = 0L;
        long signature = -3750763034362895579L;

        for (int step = 0; step < steps; step++) {
            index = next[index];
            int value = values[index];
            int selector = (value ^ (value >> 13) ^ step) & 15;
            if (selector < 9) {
                hotBranches = hotBranches + 1L;
                accumulator = accumulator + (long) (value ^ (index * 31));
                signature = signature * 1000003L + accumulator + selector;
            } else {
                coldBranches = coldBranches + 1L;
                accumulator = accumulator - (long) (value + index * 17);
                signature = signature * 1000033L ^ (accumulator - selector);
            }

            int mixedIndex = (index ^ (value >> 11) ^ (step * 97)) & mask;
            index = next[mixedIndex];
        }
        return new WalkResult(index, hotBranches, coldBranches, accumulator, signature);
    }
}
