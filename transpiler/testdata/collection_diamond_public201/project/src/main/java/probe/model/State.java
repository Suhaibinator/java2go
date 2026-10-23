package probe.model;

public final class State {
    private final int seed;
    private int count;
    private int checksum;
    private String last;
    public State(int seed) { this.seed = seed; this.checksum = seed; }
    public void accept(String text) {
        checksum = checksum * 31 + text.length() + seed;
        count++;
        last = text;
    }
    public int count() { return count; }
    public int checksum() { return checksum; }
    public String last() { return last; }
}
