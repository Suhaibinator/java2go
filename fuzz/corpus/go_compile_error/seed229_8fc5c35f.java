// seed: 229
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    public static void main(String[] args) {
        int[] arr0 = {7, -2, 7};
        arr0[0] = arr0[arr0.length - 1];
        int i1 = 0;
        for (int q2 = 0; q2 < arr0.length; q2++) { i1 += arr0[q2]; }
        int i3 = ~((int) ((long) (i1 << 8)));
        i1 = --i3;
        i3 = ((true ? 100 : -2) >>> 64) * ((-2 == -146) ? 256 : i1);
        String s4 = "AaBbCc";
        int i5 = s4.length();
        int i6 = s4.indexOf("");
        if (i3-- <= i3) {
            i5--;
        } else {
            s4 += "val";
        }
        i5 ^= i1 >>> 8;
        int i7 = 0;
        while (i7 < 1) {
            i1 -= i6;
            i3 ^= i1 >> 40;
            i7++;
        }
        int i8 = 0;
        while (i8 < 4) {
            i1 *= -2147483648;
            i8++;
        }
        i3 -= -(1000000);
        int[] arr9 = {i8, 366};
        arr9[0] = arr9[arr9.length - 1];
        int i10 = 0;
        for (int q11 = 0; q11 < arr9.length; q11++) { i10 += arr9[q11]; }
        i5--;
        i10 >>>= 31;
        System.out.println(new Box(7).scale(4));
        System.out.println(new Box(3).kind());
        System.out.println(i1);
        System.out.println(i3);
        System.out.println(i5);
        System.out.println(i6);
        System.out.println(i7);
        System.out.println(i8);
        System.out.println(i10);
        System.out.println(s4);
    }
}
