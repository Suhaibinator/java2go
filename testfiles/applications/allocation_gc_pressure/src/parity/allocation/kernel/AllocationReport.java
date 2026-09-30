package parity.allocation.kernel;

public class AllocationReport {
    private final int ephemeralRecords;
    private final int releasedCohorts;
    private final int retainedNodes;
    private final int retainedGenerationSum;
    private final long ephemeralChecksum;
    private final long traversalChecksum;
    private final long retainedChecksum;
    private final long combinedChecksum;

    public AllocationReport(
            int ephemeralRecords,
            int releasedCohorts,
            int retainedNodes,
            int retainedGenerationSum,
            long ephemeralChecksum,
            long traversalChecksum,
            long retainedChecksum,
            long combinedChecksum) {
        this.ephemeralRecords = ephemeralRecords;
        this.releasedCohorts = releasedCohorts;
        this.retainedNodes = retainedNodes;
        this.retainedGenerationSum = retainedGenerationSum;
        this.ephemeralChecksum = ephemeralChecksum;
        this.traversalChecksum = traversalChecksum;
        this.retainedChecksum = retainedChecksum;
        this.combinedChecksum = combinedChecksum;
    }

    public int ephemeralRecords() {
        return this.ephemeralRecords;
    }

    public int releasedCohorts() {
        return this.releasedCohorts;
    }

    public int retainedNodes() {
        return this.retainedNodes;
    }

    public int retainedGenerationSum() {
        return this.retainedGenerationSum;
    }

    public long ephemeralChecksum() {
        return this.ephemeralChecksum;
    }

    public long traversalChecksum() {
        return this.traversalChecksum;
    }

    public long retainedChecksum() {
        return this.retainedChecksum;
    }

    public long combinedChecksum() {
        return this.combinedChecksum;
    }
}
