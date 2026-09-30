public class Gen {
    static class Box { int kind() { return 1; } }
    static class DoubleBox extends Box { int kind() { return 2; } }
    public static void main(String[] args) {
        System.out.println(((Box) new DoubleBox()).kind());
    }
}
