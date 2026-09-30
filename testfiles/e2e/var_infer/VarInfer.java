public class VarInfer {
    public static void main(String[] args) {
        var n = 42;
        var name = "hello";
        var d = 3.5;
        var sum = n + 8;
        System.out.println(n);
        System.out.println(name);
        System.out.println(d);
        System.out.println(sum);

        for (var i = 0; i < 3; i++) {
            System.out.println("i=" + i);
        }

        var total = 0;
        int[] xs = new int[] { 10, 20, 30 };
        for (var x : xs) {
            total += x;
        }
        System.out.println("total=" + total);
    }
}
