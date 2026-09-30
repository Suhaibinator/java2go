// seed: 207
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    static class DoubleBox extends Box {
        DoubleBox(int v) { super(v); }
        int scale(int k) { return v * k * 2; }
        int kind() { return 2; }
    }
    public static void main(String[] args) {
        int[] arr0 = {-100, -2147483648, -7, 0x7FFFFFFF};
        arr0[0] = arr0[arr0.length - 1];
        int i1 = 0;
        for (int q2 = 0; q2 < arr0.length; q2++) { i1 += arr0[q2]; }
        int[] arr3 = {i1, i1};
        arr3[0] = arr3[arr3.length - 1];
        int i4 = 0;
        for (int q5 = 0; q5 < arr3.length; q5++) { i4 += arr3[q5]; }
        long l6 = (((-1L % 10000000000L) + ((long) 0)) | 0L) * ((-(9223372036854775807L + 4294967296L)) + ((-10000000000L * 1L) * ((long) 2)));
        String s7 = "AaBbCc";
        int i8 = s7.length();
        boolean b9 = s7.startsWith("Java");
        int[] arr10 = {i1, 77, 7};
        arr10[0] = arr10[arr10.length - 1];
        int i11 = 0;
        for (int q12 = 0; q12 < arr10.length; q12++) { i11 += arr10[q12]; }
        int i13 = 0;
        while (i13 < 3) {
            i4 += 1000000;
            i13++;
        }
        long l14 = (-((4294967296L >> 32) >> 0)) % -1L;
        i4 = ((-1L < 1000000000000L) ? 856 : i13) - ((int) (-(100L) / (l14 | 1L)));
        boolean b15 = (--i1 == i13) && ((-(10000000000L / -9223372036854775808L)) == l6);
        System.out.println(new Box(7).scale(3));
        System.out.println(new Box(3).kind());
        System.out.println(new DoubleBox(7).scale(3));
        System.out.println(((Box) new DoubleBox(5)).kind());
        System.out.println(i1);
        System.out.println(i4);
        System.out.println(i8);
        System.out.println(i11);
        System.out.println(i13);
        System.out.println(l6);
        System.out.println(l14);
        System.out.println(b9);
        System.out.println(b15);
        System.out.println(s7);
    }
}
