public class Exceptions {
    static int divide(int a, int b) {
        try {
            return a / b;
        } catch (ArithmeticException e) {
            return -1;
        }
    }

    static String classify(int n) {
        try {
            if (n < 0) {
                throw new IllegalArgumentException("negative: " + n);
            }
            if (n == 0) {
                throw new IllegalStateException("zero");
            }
            return "ok " + n;
        } catch (IllegalArgumentException e) {
            return "iae " + e.getMessage();
        } catch (RuntimeException e) {
            return "rte " + e.getMessage();
        } finally {
            System.out.println("classify done for " + n);
        }
    }

    public static void main(String[] args) {
        System.out.println(divide(10, 2));
        System.out.println(divide(10, 0));
        System.out.println(classify(5));
        System.out.println(classify(-3));
        System.out.println(classify(0));
    }
}
