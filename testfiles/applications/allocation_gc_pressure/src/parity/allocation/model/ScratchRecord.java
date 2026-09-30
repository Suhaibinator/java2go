package parity.allocation.model;

public class ScratchRecord {
    private final int tag;
    private final int[] left;
    private final int[] right;

    public ScratchRecord(int phase, int batch, int item, int payloadWords) {
        this.tag = phase * 4099 + batch * 257 + item;
        this.left = new int[payloadWords];
        this.right = new int[payloadWords];

        int state = (phase * 733 + batch * 193 + item * 17 + 29) % 65521;
        for (int index = 0; index < payloadWords; index++) {
            state = (state * 25173 + 13849) % 65521;
            this.left[index] = state;
            state = (state * 25173 + 13849) % 65521;
            this.right[index] = state;
        }
    }

    public long consume() {
        long checksum = this.tag + 1L;
        for (int index = 0; index < this.left.length; index++) {
            int mirror = this.right.length - index - 1;
            int mixed = (this.left[index] * 17 + this.right[mirror] * 31 + this.tag) % 1000003;
            this.left[index] = mixed;
            checksum = checksum * 65537L + mixed + this.right[index];
        }
        return checksum;
    }
}
