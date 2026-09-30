// seed: 278
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
        int i0 = ((false && false) ? 10 : 256) * (((10 + -1000000) * ('0' + 65535)) * (~('~' + -7)));
        if (false) {
            i0--;
        }
        char c1 = (char) (('9' + -1000000) - ((100 * i0) / (i0 | 1)));
        int i2 = (-(c1 + i0)) * (!true ? 3 : 1);
        boolean b3 = (((i0 / (i0 | 1)) - (-193 + 424)) * (-(0x7FFFFFFF) << 64)) == 65536;
        double d4 = 1000000 / -2.5;
        i2 |= i2 & 2;
        int i5 = 0;
        while (i5 < 2) {
            i2 *= 10;
            i5++;
        }
        System.out.println(new Box(4).scale(4));
        System.out.println(new Box(3).kind());
        System.out.println(new DoubleBox(4).scale(4));
        System.out.println(((Box) new DoubleBox(5)).kind());
        System.out.println(i0);
        System.out.println(i2);
        System.out.println(i5);
        System.out.println(d4);
        System.out.println(b3);
        System.out.println(c1);
        System.out.println((int) c1);
    }
}
