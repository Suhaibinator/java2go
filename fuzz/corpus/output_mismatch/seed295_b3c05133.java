// seed: 295
public class Gen {
    public static void main(String[] args) {
        long l0 = (((long) (1 + 394)) >> 100) >> 63;
        int i1 = -225;
        int i2 = -(('~' + i1) | 10);
        int i3 = (((-1 * i2) + (i2 - i1)) - ((763 >> 8) + (i1 - i1))) << 31;
        int i4 = '!' + 256;
        i1 &= ++i1;
        i4 = (('z' + -2) / (i4 | 1)) * (('z' + 2) >>> 33);
        i4++;
        if (!(9223372036854775807L > l0)) {
            for (int it5 = 1; it5 < 3; it5++) {
                i2 ^= -1;
            }
        } else {
            if ((-1 % 100) < -786) {
                l0 = ((long) ((7 - i3) >>> 33)) % (l0 | 1L);
            }
        }
        i4++;
        int i6 = ~((('9' + i3) << 1) << 64);
        int i7 = 0;
        while (i7 < 3) {
            i1 -= 0;
            i7++;
        }
        long l8 = -(((l0 - 4294967296L) + (l0 ^ l0)) & 9223372036854775807L);
        int i9 = 0;
        while (i9 < 2) {
            i1 ^= i6;
            i9++;
        }
        System.out.println(i1);
        System.out.println(i2);
        System.out.println(i3);
        System.out.println(i4);
        System.out.println(i6);
        System.out.println(i7);
        System.out.println(i9);
        System.out.println(l0);
        System.out.println(l8);
    }
}
