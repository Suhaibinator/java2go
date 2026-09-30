// seed: 208
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    public static void main(String[] args) {
        int i0 = (((32 & 3) >> 8) % 256) / 1;
        i0--;
        for (int it1 = 0; it1 < 3; it1++) {
            i0 += i0;
            if (!(i0 < i0)) {
                i0 = ('~' + i0) / (i0 | 1);
            } else {
                i0++;
            }
        }
        int[] arr2 = {i0, i0};
        arr2[0] = arr2[arr2.length - 1];
        int i3 = 0;
        for (int q4 = 0; q4 < arr2.length; q4++) { i3 += arr2[q4]; }
        i0--;
        long l5 = -((long) ((i3 | i0) & i3));
        i0--;
        i3--;
        int i6 = 0;
        while (i6 < 4) {
            i0 += i0;
            i6++;
        }
        l5 = ((-(10000000000L / -1L)) - (-(1L) >>> 0)) ^ l5;
        String s7 = "Java";
        int i8 = s7.length();
        String s9 = s7.length() > 1 ? s7.substring(1) : s7;
        int i10 = 0;
        while (i10 < 4) {
            i3 += i8;
            i10++;
        }
        int i11 = i6++;
        i8 -= -(i8);
        i6--;
        i11 >>>= 0;
        System.out.println(new Box(2).scale(4));
        System.out.println(new Box(3).kind());
        System.out.println(i0);
        System.out.println(i3);
        System.out.println(i6);
        System.out.println(i8);
        System.out.println(i10);
        System.out.println(i11);
        System.out.println(l5);
        System.out.println(s7);
        System.out.println(s9);
    }
}
