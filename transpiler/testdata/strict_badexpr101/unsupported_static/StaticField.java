public class StaticField {
    static int count;
    public static int value() {
        count += java.time.Year.MAX_VALUE;
        return count;
    }
}
