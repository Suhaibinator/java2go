// seed: 309
public class Gen {
    public static void main(String[] args) {
        String s0 = "x";
        int i1 = s0.length();
        boolean b2 = s0.startsWith("Java");
        long l3 = ((-(1000000000000L >> 32)) % -10000000000L) ^ -1L;
        char c4 = (char) ('z' + i1);
        i1++;
        StringBuilder sb6 = new StringBuilder();
        sb6.append(i1);
        String s5 = sb6.toString();
        String s7 = "beta" + i1;
        long l8 = (((l3 + l3) * (-10000000000L * 1L)) + ((2147483648L - l3) >> 64)) << 100;
        boolean b9 = ((((long) i1) / (l8 | 1L)) > 100L) && (--i1 < i1);
        int i10 = (int) (((4294967296L >>> 33) % (l8 | 1L)) * ((0L << 1) >> 1));
        System.out.println(i1);
        System.out.println(i10);
        System.out.println(l3);
        System.out.println(l8);
        System.out.println(b2);
        System.out.println(b9);
        System.out.println(c4);
        System.out.println((int) c4);
        System.out.println(s0);
        System.out.println(s5);
        System.out.println(s7);
    }
}
