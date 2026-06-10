public class VarSimple {
    public static void main(String[] args) {
        var n = 42;
        var name = "hello";
        var d = 3.5;
        var flag = true;
        var sum = n + 8;

        System.out.println(n);
        System.out.println(name);
        System.out.println(d);
        System.out.println(flag);
        System.out.println(sum);

        for (var i = 0; i < 3; i++) {
            System.out.println("i=" + i);
        }

        var total = 0;
        for (var j = 1; j <= 4; j++) {
            total += j;
        }
        System.out.println("total=" + total);

        var greeting = name + ", world";
        System.out.println(greeting.length());
    }
}
