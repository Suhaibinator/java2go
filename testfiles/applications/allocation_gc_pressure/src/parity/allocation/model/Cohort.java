package parity.allocation.model;

public class Cohort {
    private final int generation;
    private final AllocationNode[] nodes;

    public Cohort(int generation, int nodeCount, int payloadWords) {
        this.generation = generation;
        this.nodes = new AllocationNode[nodeCount];

        for (int index = 0; index < nodeCount; index++) {
            this.nodes[index] = new AllocationNode(index, generation, payloadWords);
        }
        for (int index = 0; index < nodeCount; index++) {
            int nextIndex = (index + 1) % nodeCount;
            int skipIndex = (index * 17 + generation * 13 + 7) % nodeCount;
            this.nodes[index].connect(this.nodes[nextIndex], this.nodes[skipIndex]);
        }
    }

    public int generation() {
        return this.generation;
    }

    public int nodeCount() {
        return this.nodes.length;
    }

    public long traverseAndMutate(int steps, int salt) {
        AllocationNode cursor = this.nodes[(this.generation * 19 + salt) % this.nodes.length];
        long checksum = this.generation + 1L;
        for (int step = 0; step < steps; step++) {
            long sampled = cursor.sampleAndMutate(step, salt);
            checksum = checksum * 1000003L + sampled + cursor.id();
            if ((step + salt) % 5 == 0) {
                cursor = cursor.skip();
            } else {
                cursor = cursor.next();
            }
        }
        return checksum;
    }

    public long checksum() {
        long checksum = this.generation + 1L;
        for (int index = 0; index < this.nodes.length; index++) {
            checksum = checksum * 65537L + this.nodes[index].checksum();
        }
        return checksum;
    }
}
