package parity.integerbranch.search;

import parity.integerbranch.model.SearchResult;

public class BranchAndBoundQueens {
    private final int size;
    private final int variant;
    private final long fullMask;
    private final long[] allowedRows;
    private long solutions;
    private long nodes;
    private long branches;
    private long deadEnds;
    private long signature;

    public BranchAndBoundQueens(int size, int variant) {
        this.size = size;
        this.variant = variant;
        this.fullMask = (1L << size) - 1L;
        this.allowedRows = ConstraintBoard.allowedRows(size, variant);
        this.solutions = 0L;
        this.nodes = 0L;
        this.branches = 0L;
        this.deadEnds = 0L;
        this.signature = 1469598103934665603L;
    }

    public SearchResult solve() {
        search(0, 0L, 0L, 0L, 7809847782465536322L + this.variant);
        return new SearchResult(this.variant, this.solutions, this.nodes,
                this.branches, this.deadEnds, this.signature);
    }

    private void search(int row, long occupiedColumns, long diagonalLeft,
                        long diagonalRight, long pathSignature) {
        this.nodes = this.nodes + 1L;
        if (row == this.size) {
            this.solutions = this.solutions + 1L;
            this.signature = this.signature * 1000003L + pathSignature;
            return;
        }

        long attacked = occupiedColumns | diagonalLeft | diagonalRight;
        long choices = this.allowedRows[row] & this.fullMask & ~attacked;
        if (choices == 0L) {
            this.deadEnds = this.deadEnds + 1L;
            return;
        }

        while (choices != 0L) {
            long selected = choices & -choices;
            choices = choices ^ selected;
            this.branches = this.branches + 1L;

            long nextLeft = ((diagonalLeft | selected) << 1) & this.fullMask;
            long nextRight = (diagonalRight | selected) >> 1;
            long nextSignature = pathSignature * 1315423911L
                    + selected * 31L + (long) (row + 1) * 17L;
            search(row + 1, occupiedColumns | selected,
                    nextLeft, nextRight, nextSignature);
        }
    }
}
