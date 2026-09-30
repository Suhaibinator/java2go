// seed: 330
public class Gen {
    public static void main(String[] args) {
        double d0 = -(((0.5 * 0.1) / 2.0) + ((10 / 1.0) / 3.14));
        int i2 = 0;
        int i1 = 0;
        while (i1 < 1) {
            i2 ^= i1;
            i1++;
        }
        int i3 = 'a' + i2;
        i3 &= (true && false) ? i3 : 2147483647;
        int i4 = ((int) ((-10000000000L - 0L) / 9223372036854775807L)) + ((int) ((100L % 1L) << 33));
        System.out.println(i1);
        System.out.println(i2);
        System.out.println(i3);
        System.out.println(i4);
        System.out.println(d0);
    }
}
