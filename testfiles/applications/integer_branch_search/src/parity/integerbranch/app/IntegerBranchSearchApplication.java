package parity.integerbranch.app;

import parity.integerbranch.model.SearchResult;
import parity.integerbranch.model.SieveResult;
import parity.integerbranch.model.SortSearchResult;
import parity.integerbranch.model.WalkResult;
import parity.integerbranch.search.BranchAndBoundQueens;
import parity.integerbranch.sieve.PrimeSieve;
import parity.integerbranch.sorting.DeterministicSortSearch;
import parity.integerbranch.walk.IrregularMemoryWalker;

public class IntegerBranchSearchApplication {
    private static final int BOARD_SIZE = 13;
    private static final int SEARCH_VARIANTS = 4;
    private static final int SEARCH_ROUNDS = 20;
    private static final int SIEVE_LIMIT = 5000000;
    private static final int SIEVE_ROUNDS = 12;
    private static final int SORT_LENGTH = 262144;
    private static final int SORT_ROUNDS = 16;
    private static final int SORT_QUERIES_PER_ROUND = 65536;
    private static final int WALK_TABLE_BITS = 20;
    private static final int WALK_STEPS = 12000000;
    private static final int WALK_ROUNDS = 18;
    private static final int WALK_SEED = 324508639;

    public static void main(String[] args) {
        long searchChecksum = 0L;
        long totalSolutions = 0L;
        long totalNodes = 0L;
        long totalDeadEnds = 0L;

        System.out.println("application=integer_branch_search");
        System.out.println("board_size=" + BOARD_SIZE);
        for (int variant = 0; variant < SEARCH_VARIANTS; variant++) {
            long variantSolutions = 0L;
            long variantNodes = 0L;
            long variantBranches = 0L;
            long variantDeadEnds = 0L;
            long variantSignature = 1469598103934665603L + variant;
            for (int round = 0; round < SEARCH_ROUNDS; round++) {
                SearchResult current = new BranchAndBoundQueens(BOARD_SIZE, variant).solve();
                variantSolutions = variantSolutions + current.solutions();
                variantNodes = variantNodes + current.nodes();
                variantBranches = variantBranches + current.branches();
                variantDeadEnds = variantDeadEnds + current.deadEnds();
                variantSignature = variantSignature * 1000003L
                        + current.signature() + (long) (round + 1) * 97L;
            }
            SearchResult result = new SearchResult(variant, variantSolutions, variantNodes,
                    variantBranches, variantDeadEnds, variantSignature);
            totalSolutions = totalSolutions + result.solutions();
            totalNodes = totalNodes + result.nodes();
            totalDeadEnds = totalDeadEnds + result.deadEnds();
            searchChecksum = searchChecksum * 1000003L
                    + result.signature() + result.nodes() * 31L + result.branches();
            System.out.println("search_" + result.variant()
                    + "=rounds:" + SEARCH_ROUNDS
                    + ",solutions:" + result.solutions()
                    + ",nodes:" + result.nodes()
                    + ",branches:" + result.branches()
                    + ",dead_ends:" + result.deadEnds()
                    + ",signature:" + result.signature());
        }

        int sievePrimeCount = 0;
        int sieveLastPrime = 0;
        long sievePrimeSum = 0L;
        long sieveSignature = 1099511628211L;
        for (int round = 0; round < SIEVE_ROUNDS; round++) {
            SieveResult current = PrimeSieve.run(SIEVE_LIMIT);
            sievePrimeCount = sievePrimeCount + current.primeCount();
            sieveLastPrime = current.lastPrime();
            sievePrimeSum = sievePrimeSum + current.primeSum();
            sieveSignature = sieveSignature * 16777619L
                    + current.signature() + (long) (round + 1) * 193L;
        }
        SieveResult sieve = new SieveResult(SIEVE_LIMIT, sievePrimeCount,
                sieveLastPrime, sievePrimeSum, sieveSignature);

        SortSearchResult sort = DeterministicSortSearch.run(SORT_LENGTH, SORT_ROUNDS,
                SORT_QUERIES_PER_ROUND, 610839777);

        int walkFinalIndex = 0;
        long walkHotBranches = 0L;
        long walkColdBranches = 0L;
        long walkAccumulator = 0L;
        long walkSignature = -3750763034362895579L;
        for (int round = 0; round < WALK_ROUNDS; round++) {
            int roundSeed = WALK_SEED + round * 104729;
            WalkResult current = IrregularMemoryWalker.run(WALK_TABLE_BITS, WALK_STEPS, roundSeed);
            walkFinalIndex = current.finalIndex();
            walkHotBranches = walkHotBranches + current.hotBranches();
            walkColdBranches = walkColdBranches + current.coldBranches();
            walkAccumulator = walkAccumulator + current.accumulator();
            walkSignature = walkSignature * 1000033L
                    ^ (current.signature() + (long) (round + 1) * 389L);
        }
        WalkResult walk = new WalkResult(walkFinalIndex, walkHotBranches,
                walkColdBranches, walkAccumulator, walkSignature);

        long combinedChecksum = searchChecksum * 31L
                + sieve.signature() * 17L
                + sort.signature() * 19L
                + walk.signature() * 13L
                + totalSolutions * 7L
                + totalNodes * 5L
                + totalDeadEnds;

        System.out.println("search_totals=solutions:" + totalSolutions
                + ",nodes:" + totalNodes
                + ",dead_ends:" + totalDeadEnds
                + ",checksum:" + searchChecksum);
        System.out.println("sieve=rounds:" + SIEVE_ROUNDS
                + ",limit:" + sieve.limit()
                + ",primes:" + sieve.primeCount()
                + ",last:" + sieve.lastPrime()
                + ",sum:" + sieve.primeSum()
                + ",signature:" + sieve.signature());
        System.out.println("sort=length:" + sort.length()
                + ",rounds:" + sort.rounds()
                + ",queries:" + sort.queries()
                + ",hits:" + sort.hits()
                + ",misses:" + sort.misses()
                + ",comparisons:" + sort.comparisons()
                + ",signature:" + sort.signature());
        System.out.println("walk=rounds:" + WALK_ROUNDS
                + ",table_bits:" + WALK_TABLE_BITS
                + ",steps:" + WALK_STEPS
                + ",final_index:" + walk.finalIndex()
                + ",hot:" + walk.hotBranches()
                + ",cold:" + walk.coldBranches()
                + ",accumulator:" + walk.accumulator()
                + ",signature:" + walk.signature());
        System.out.println("combined_checksum=" + combinedChecksum);
    }
}
