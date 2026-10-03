public class Shadow {
    static class Year { static final int MAX_VALUE = 11; }
    public static int value() {
        int count = 0;
        count += Year.MAX_VALUE;
        return count;
    }
}
