package parity.allocation.kernel;

import parity.allocation.model.Cohort;
import parity.allocation.model.ScratchRecord;

public class AllocationGcKernel {
    public static AllocationReport run(
            int phases,
            int ephemeralBatches,
            int batchSize,
            int scratchWords,
            int retainedSlots,
            int cohortNodes,
            int cohortPayloadWords,
            int traversalSteps) {
        Cohort[] retained = new Cohort[retainedSlots];
        long ephemeralChecksum = 1L;
        long traversalChecksum = 1L;
        int ephemeralRecords = 0;

        for (int phase = 0; phase < phases; phase++) {
            for (int batchIndex = 0; batchIndex < ephemeralBatches; batchIndex++) {
                ScratchRecord[] batch = new ScratchRecord[batchSize];
                for (int item = 0; item < batchSize; item++) {
                    batch[item] = new ScratchRecord(phase, batchIndex, item, scratchWords);
                }
                long batchChecksum = batchIndex + 1L;
                for (int item = 0; item < batch.length; item++) {
                    batchChecksum = batchChecksum * 1000003L + batch[item].consume();
                }
                ephemeralChecksum = ephemeralChecksum * 65537L + batchChecksum;
                ephemeralRecords += batch.length;
            }

            int replacementSlot = phase % retainedSlots;
            retained[replacementSlot] = new Cohort(phase, cohortNodes, cohortPayloadWords);

            for (int slot = 0; slot < retained.length; slot++) {
                Cohort cohort = retained[slot];
                if (cohort != null) {
                    long traversed = cohort.traverseAndMutate(traversalSteps, phase + slot * 31);
                    traversalChecksum = traversalChecksum * 1000003L + traversed;
                }
            }
        }

        int retainedNodes = 0;
        int retainedGenerationSum = 0;
        long retainedChecksum = 1L;
        for (int slot = 0; slot < retained.length; slot++) {
            Cohort cohort = retained[slot];
            retainedNodes += cohort.nodeCount();
            retainedGenerationSum += cohort.generation();
            retainedChecksum = retainedChecksum * 65537L + cohort.checksum();
        }

        int releasedCohorts = phases - retainedSlots;
        long combinedChecksum = ephemeralChecksum * 31L
                + traversalChecksum * 17L
                + retainedChecksum * 13L
                + retainedGenerationSum;
        return new AllocationReport(
                ephemeralRecords,
                releasedCohorts,
                retainedNodes,
                retainedGenerationSum,
                ephemeralChecksum,
                traversalChecksum,
                retainedChecksum,
                combinedChecksum);
    }
}
