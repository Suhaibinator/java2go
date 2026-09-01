public class Overloading {
    static String describe(int x) {
        return "int:" + x;
    }

    static String describe(long x) {
        return "long:" + x;
    }

    static String describe(double x) {
        return "double:" + x;
    }

    static String describe(String x) {
        return "string:" + x;
    }

    static int sum(int a, int b) {
        return a + b;
    }

    static int sum(int a, int b, int c) {
        return a + b + c;
    }

    public static void main(String[] args) {
        System.out.println(describe(5));
        System.out.println(describe(5L));
        System.out.println(describe(5.0));
        System.out.println(describe("five"));

        // int literal widens to long when only long overload would differ;
        // here exact int overload is chosen
        int i = 7;
        long l = 7L;
        System.out.println(describe(i));
        System.out.println(describe(l));

        System.out.println(sum(1, 2));
        System.out.println(sum(1, 2, 3));
    }
}
