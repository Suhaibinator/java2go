// seed: 225
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    public static void main(String[] args) {
        if (false) {
            int i0 = 0;
            for (int it1 = 1; it1 < 2; it1++) {
                i0 *= 3;
            }
        } else {
            int i2 = 0;
            for (int it3 = 2; it3 < 4; it3++) {
                i2 *= 65536;
            }
        }
        long l4 = ((((long) -100) | 100L) / -10000000000L) / 1L;
        String s5 = "Hello";
        int i6 = s5.length();
        boolean b7 = s5.startsWith("AaBbCc");
        double d8 = ((i6 / -2.5) * (i6 / 1.0)) * ((-975 / 10.0) + ((0xFFFFFFFF / 2.0) / 0.1));
        char c9 = '9';
        System.out.println(new Box(2).scale(1));
        System.out.println(new Box(3).kind());
        System.out.println(i6);
        System.out.println(l4);
        System.out.println(d8);
        System.out.println(b7);
        System.out.println(c9);
        System.out.println((int) c9);
        System.out.println(s5);
    }
}
