record Point(int x, int y) {
    int manhattan() {
        return Math.abs(x) + Math.abs(y);
    }
}

record Named(String name, int age) {
}

public class Records {
    public static void main(String[] args) {
        Point p = new Point(3, -4);
        System.out.println(p.x());
        System.out.println(p.y());
        System.out.println(p.manhattan());

        Named n = new Named("Ann", 30);
        System.out.println(n.name() + " " + n.age());

        Point q = new Point(3, -4);
        System.out.println(p.equals(q));
    }
}
