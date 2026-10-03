public class VolatileField {
    volatile int count;
    public int value() {
        count += java.time.Year.MAX_VALUE;
        return count;
    }
}
