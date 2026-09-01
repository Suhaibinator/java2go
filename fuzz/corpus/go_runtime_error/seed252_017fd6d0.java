// seed: 252
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
        StringBuilder sb1 = new StringBuilder();
        sb1.append("x");
        String s0 = sb1.toString();
        s0 += "x" + (-2 + -833);
        if ((2147483647 < -1) || !false) {
            s0 += "x";
        }
        System.out.println(new Box(3).scale(5));
        System.out.println(new Box(3).kind());
        System.out.println(new DoubleBox(3).scale(5));
        System.out.println(((Box) new DoubleBox(5)).kind());
        System.out.println(s0);
    }
}
