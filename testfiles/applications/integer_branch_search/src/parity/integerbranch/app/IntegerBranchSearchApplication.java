package parity.integerbranch.app;

import parity.integerbranch.model.SearchResult;
import parity.integerbranch.model.SieveResult;
import parity.integerbranch.model.WalkResult;
import parity.integerbranch.search.BranchAndBoundQueens;
import parity.integerbranch.sieve.PrimeSieve;
import parity.integerbranch.walk.IrregularMemoryWalker;

public class IntegerBranchSearchApplication {
    private static final int BOARD_SIZE = 13;
    private static final int SEARCH_VARIANTS = 4;
    private static final int SIEVE_LIMIT = 5000000;
    private static final int WALK_TABLE_BITS = 20;
    private static final int WALK_STEPS = 12000000;

    public static void main(String[] args) {
        long searchChecksum = 0L;
        long totalSolutions = 0L;
        long totalNodes = 0L;
        long totalDeadEnds = 0L;

        System.out.println("application=integer_branch_search");
        System.out.println("board_size=" + BOARD_SIZE);
        for (int variant = 0; variant < SEARCH_VARIANTS; variant++) {
            SearchResult result = new BranchAndBoundQueens(BOARD_SIZE, variant).solve();
            totalSolutions = totalSolutions + result.solutions();
            totalNodes = totalNodes + result.nodes();
            totalDeadEnds = totalDeadEnds + result.deadEnds();
            searchChecksum = searchChecksum * 1000003L
                    + result.signature() + result.nodes() * 31L + result.branches();
            System.out.println("search_" + result.variant()
                    + "=solutions:" + result.solutions()
                    + ",nodes:" + result.nodes()
                    + ",branches:" + result.branches()
                    + ",dead_ends:" + result.deadEnds()
                    + ",signature:" + result.signature());
        }

        SieveResult sieve = PrimeSieve.run(SIEVE_LIMIT);
        WalkResult walk = IrregularMemoryWalker.run(WALK_TABLE_BITS, WALK_STEPS, 324508639);
        long combinedChecksum = searchChecksum * 31L
                + sieve.signature() * 17L
                + walk.signature() * 13L
                + totalSolutions * 7L
                + totalNodes * 5L
                + totalDeadEnds;

        System.out.println("search_totals=solutions:" + totalSolutions
                + ",nodes:" + totalNodes
                + ",dead_ends:" + totalDeadEnds
                + ",checksum:" + searchChecksum);
        System.out.println("sieve=limit:" + sieve.limit()
                + ",primes:" + sieve.primeCount()
                + ",last:" + sieve.lastPrime()
                + ",sum:" + sieve.primeSum()
                + ",signature:" + sieve.signature());
        System.out.println("walk=table_bits:" + WALK_TABLE_BITS
                + ",steps:" + WALK_STEPS
                + ",final_index:" + walk.finalIndex()
                + ",hot:" + walk.hotBranches()
                + ",cold:" + walk.coldBranches()
                + ",accumulator:" + walk.accumulator()
                + ",signature:" + walk.signature());
        System.out.println("combined_checksum=" + combinedChecksum);
    }
}
