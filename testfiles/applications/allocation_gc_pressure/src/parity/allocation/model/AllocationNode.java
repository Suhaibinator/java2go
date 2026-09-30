package parity.allocation.model;

public class AllocationNode {
    private final int id;
    private final int[] payload;
    private AllocationNode next;
    private AllocationNode skip;

    public AllocationNode(int id, int generation, int payloadWords) {
        this.id = id;
        this.payload = new int[payloadWords];

        int state = (id * 97 + generation * 193 + 17) % 65521;
        for (int index = 0; index < payloadWords; index++) {
            state = (state * 25173 + 13849) % 65521;
            this.payload[index] = state + (id + index) % 31;
        }
    }

    public void connect(AllocationNode next, AllocationNode skip) {
        this.next = next;
        this.skip = skip;
    }

    public int id() {
        return this.id;
    }

    public AllocationNode next() {
        return this.next;
    }

    public AllocationNode skip() {
        return this.skip;
    }

    public long sampleAndMutate(int step, int salt) {
        int firstIndex = (step * 13 + this.id * 7 + salt) % this.payload.length;
        int secondIndex = (firstIndex + 17) % this.payload.length;
        int updated = (this.payload[firstIndex] + step + salt + this.id) % 1000003;
        this.payload[firstIndex] = updated;
        return updated * 257L + this.payload[secondIndex];
    }

    public long checksum() {
        long checksum = this.id + 1L;
        for (int index = 0; index < this.payload.length; index++) {
            checksum = checksum * 1000003L
                    + this.payload[index] * 31L
                    + index;
        }
        return checksum;
    }
}
