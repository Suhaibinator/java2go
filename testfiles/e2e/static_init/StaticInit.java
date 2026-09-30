public class StaticInit {
    static int a = init("a", 1);
    static int b;

    static {
        System.out.println("block1");
        b = init("b", 2);
    }

    static int c = init("c", 3);

    static {
        System.out.println("block2 sees a=" + a + " b=" + b + " c=" + c);
    }

    static int init(String name, int value) {
        System.out.println("init " + name);
        return value;
    }

    public static void main(String[] args) {
        System.out.println("main");
        System.out.println(a + b + c);
    }
}
