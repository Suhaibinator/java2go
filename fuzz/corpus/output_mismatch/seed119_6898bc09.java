// seed: 119
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    public static void main(String[] args) {
        int i0 = 0x7FFFFFFF;
        if ((true || false) || (i0 == i0)) {
            if (!false || (i0 > i0)) {
            }
        }
        int[] arr1 = {-7, i0, 1, 0};
        int i2 = 0;
        for (int q3 = 0; q3 < arr1.length; q3++) { i2 += arr1[q3]; }
        char c4 = (char) ((((0 <= i0) ? i2 : i0) * (i2 >>> 16)) & i0);
        double d5 = (((3.14 / -2.5) + (-2.5 / 0.1)) * ((3.14 - 0.0) + (-84 / 0.5))) / -1.0;
        StringBuilder sb7 = new StringBuilder();
        String s6 = sb7.toString();
        int i8 = 0;
        while (i8 < 1) {
            i8++;
        }
        for (int it9 = 2; it9 < 5; it9++) {
            i0 += c4 + i2;
        }
        int i10 = (~(++i0) + ((int) ((long) 0))) % 1;
        System.out.println(new Box(2).scale(4));
        System.out.println(new Box(3).kind());
        System.out.println(i0);
        System.out.println(i10);
        System.out.println(d5);
        System.out.println(s6);
    }
}
