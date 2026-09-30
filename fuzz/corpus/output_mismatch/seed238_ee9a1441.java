// seed: 238
public class Gen {
    static class Box {
        int v;
        Box(int v) { this.v = v; }
        int scale(int k) { return v * k; }
        int kind() { return 1; }
    }
    public static void main(String[] args) {
        int i0 = (-296 <= 924) ? 0 : -775;
        int i1 = true ? 2 : i0;
        int i2 = 0;
        while (i2 < 4) {
            i1 -= 255;
            i1 ^= i1 + i0;
            i2++;
        }
        i0 >>= 16;
        char c3 = '0';
        String s4 = "k" + (-252 + i2);
        c3 *= (char) (100);
        StringBuilder sb6 = new StringBuilder();
        sb6.append(i2);
        sb6.append(true);
        String s5 = sb6.toString();
        s5 += "n=" + (i1 + i0);
        for (int it7 = 1; it7 < 2; it7++) {
            i1 -= -1000000;
        }
        for (int it8 = 1; it8 < 2; it8++) {
            i0 *= 196;
        }
        s5 = "::" + c3 + false;
        System.out.println(new Box(6).scale(5));
        System.out.println(new Box(3).kind());
        System.out.println(i0);
        System.out.println(i1);
        System.out.println(i2);
        System.out.println(c3);
        System.out.println((int) c3);
        System.out.println(s4);
        System.out.println(s5);
    }
}
