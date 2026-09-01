public class NumericEdge {
    public static void main(String[] args) {
        // int overflow wraps (32-bit two's complement)
        int maxInt = 2147483647;
        System.out.println(maxInt + 1);
        int minInt = -2147483648;
        System.out.println(minInt - 1);
        System.out.println(maxInt * 2);

        // integer division truncates toward zero
        System.out.println(7 / 2);
        System.out.println(-7 / 2);
        System.out.println(7 / -2);
        System.out.println(-7 % 3);
        System.out.println(7 % -3);

        // long arithmetic
        long bigL = 10000000000L;
        System.out.println(bigL * 2);

        // shifts
        System.out.println(1 << 4);
        System.out.println(-8 >> 1);
        System.out.println(-8 >>> 28);
        System.out.println(256 >>> 2);
        int neg = -1;
        System.out.println(neg >>> 0);

        // long shift distance masking is 6 bits for long, 5 bits for int
        System.out.println(1 << 32);
        System.out.println(1L << 32);

        // char arithmetic promotes to int
        char c = 'A';
        System.out.println(c + 1);
        System.out.println((char) (c + 1));
        System.out.println((int) c);

        // string + number concatenation, left-to-right
        System.out.println("sum=" + 1 + 2);
        System.out.println("sum=" + (1 + 2));
        System.out.println(1 + 2 + "=sum");

        // bitwise ops
        System.out.println(0xF0 & 0x0F);
        System.out.println(0xF0 | 0x0F);
        System.out.println(0xFF ^ 0x0F);
        System.out.println(~0);
    }
}
