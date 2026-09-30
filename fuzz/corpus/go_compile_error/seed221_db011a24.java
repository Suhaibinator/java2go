// seed: 221
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    public static void main(String[] args) {
        int[] arr0 = {586, 0xFFFFFFFF, -914, -7};
        arr0[0] = arr0[arr0.length - 1];
        int i1 = 0;
        for (int q2 = 0; q2 < arr0.length; q2++) { i1 += arr0[q2]; }
        String s3 = "n=" + i1;
        i1 = (int) (((9223372036854775807L & 100L) << 32) >>> 64);
        double d4 = 636 / 0.1;
        i1--;
        int i5 = ((((1000000000000L < 4294967296L) ? i1 : i1) - ((true || false) ? -100 : -930)) + ((i1 + i1) | i1)) * ((true ? 1 : i1) - (-('Z' + i1)));
        int[] arr6 = {i5, 255};
        arr6[0] = arr6[arr6.length - 1];
        int i7 = 0;
        for (int q8 = 0; q8 < arr6.length; q8++) { i7 += arr6[q8]; }
        int i9 = (i1-- / (i5 | 1)) / 256;
        i9--;
        i9 >>>= 8;
        char c10 = 'z';
        d4 = i9 / -1.0;
        if (!true) {
            c10++;
        }
        i5--;
        StringBuilder sb12 = new StringBuilder();
        sb12.append("val");
        sb12.append(-681);
        sb12.append(i9);
        String s11 = sb12.toString();
        System.out.println(new Box(4).scale(1));
        System.out.println(new Box(3).kind());
        System.out.println(i1);
        System.out.println(i5);
        System.out.println(i7);
        System.out.println(i9);
        System.out.println(d4);
        System.out.println(c10);
        System.out.println((int) c10);
        System.out.println(s3);
        System.out.println(s11);
    }
}
