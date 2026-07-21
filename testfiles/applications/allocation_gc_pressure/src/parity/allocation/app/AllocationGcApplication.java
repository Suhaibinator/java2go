package parity.allocation.app;

import parity.allocation.kernel.AllocationGcKernel;
import parity.allocation.kernel.AllocationReport;

public class AllocationGcApplication {
    private static final int PHASES = 72;
    private static final int EPHEMERAL_BATCHES = 10;
    private static final int BATCH_SIZE = 1024;
    private static final int SCRATCH_WORDS = 28;
    private static final int RETAINED_SLOTS = 6;
    private static final int COHORT_NODES = 1536;
    private static final int COHORT_PAYLOAD_WORDS = 48;
    private static final int TRAVERSAL_STEPS = 12000;

    public static void main(String[] args) {
        AllocationReport report = AllocationGcKernel.run(
                PHASES,
                EPHEMERAL_BATCHES,
                BATCH_SIZE,
                SCRATCH_WORDS,
                RETAINED_SLOTS,
                COHORT_NODES,
                COHORT_PAYLOAD_WORDS,
                TRAVERSAL_STEPS);

        System.out.println("application=allocation_gc_pressure");
        System.out.println("phases=" + PHASES);
        System.out.println("ephemeral_batches=" + EPHEMERAL_BATCHES);
        System.out.println("batch_size=" + BATCH_SIZE);
        System.out.println("scratch_words=" + SCRATCH_WORDS);
        System.out.println("retained_slots=" + RETAINED_SLOTS);
        System.out.println("cohort_nodes=" + COHORT_NODES);
        System.out.println("cohort_payload_words=" + COHORT_PAYLOAD_WORDS);
        System.out.println("traversal_steps=" + TRAVERSAL_STEPS);
        System.out.println("ephemeral_records=" + report.ephemeralRecords());
        System.out.println("released_cohorts=" + report.releasedCohorts());
        System.out.println("retained_nodes=" + report.retainedNodes());
        System.out.println("retained_generation_sum=" + report.retainedGenerationSum());
        System.out.println("ephemeral_checksum=" + report.ephemeralChecksum());
        System.out.println("traversal_checksum=" + report.traversalChecksum());
        System.out.println("retained_checksum=" + report.retainedChecksum());
        System.out.println("combined_checksum=" + report.combinedChecksum());
    }
}
