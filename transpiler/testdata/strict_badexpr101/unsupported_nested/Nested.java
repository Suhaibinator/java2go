public class Nested {
    public static int value() {
        int count = 0;
        return 1 + (count += java.time.Year.MAX_VALUE);
    }
}
