// seed: 212
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
        int i0 = 0;
        for (int it1 = 0; it1 < 4; it1++) {
            i0 *= i0;
        }
        StringBuilder sb3 = new StringBuilder();
        sb3.append(i0);
        String s2 = sb3.toString();
        i0--;
        s2 += "out";
        char c4 = 'a';
        int i5 = 0;
        while (i5 < 3) {
            i0 -= 513;
            c4 <<= 0;
            i5++;
        }
        String s6 = "x";
        int i7 = s6.length();
        String s8 = s6.length() > 1 ? s6.substring(1) : s6;
        for (int it9 = 2; it9 < 7; it9++) {
            i7 += i7;
        }
        for (int it10 = 2; it10 < 5; it10++) {
            i0 += -863;
            i5 |= i5 | 10;
        }
        for (int it11 = 0; it11 < 3; it11++) {
            i5 += i0;
            i5++;
        }
        for (int it12 = 2; it12 < 6; it12++) {
            i7 += i5;
            for (int it13 = 2; it13 < 7; it13++) {
                i5 ^= -747;
            }
        }
        if (!(true || true)) {
            i7 = (((int) (1L / 100L)) << 0) + ((668 >= -534) ? i0 : i0);
        } else {
            i0 += (false && false) ? 65535 : -452;
        }
        s6 += 65536 + i5 + "alpha";
        StringBuilder sb15 = new StringBuilder();
        sb15.append(false);
        String s14 = sb15.toString();
        int i16 = 0;
        while (i16 < 1) {
            i0 += i16;
            for (int it17 = 1; it17 < 5; it17++) {
                i0 -= -2;
            }
            i16++;
        }
        System.out.println(new Box(4).scale(2));
        System.out.println(new Box(3).kind());
        System.out.println(new DoubleBox(4).scale(2));
        System.out.println(((Box) new DoubleBox(5)).kind());
        System.out.println(i0);
        System.out.println(i5);
        System.out.println(i7);
        System.out.println(i16);
        System.out.println(c4);
        System.out.println((int) c4);
        System.out.println(s2);
        System.out.println(s6);
        System.out.println(s8);
        System.out.println(s14);
    }
}
