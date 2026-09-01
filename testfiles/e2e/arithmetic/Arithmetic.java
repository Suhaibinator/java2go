public class Arithmetic {
    public static void main(String[] args) {
        int a = 17;
        int b = 5;
        System.out.println(a + b);
        System.out.println(a - b);
        System.out.println(a * b);
        System.out.println(a / b);
        System.out.println(a % b);
        System.out.println(-a);
        System.out.println((a + b) * 2 - b);

        int x = 1;
        x += 10;
        x -= 3;
        x *= 2;
        System.out.println(x);

        int counter = 0;
        System.out.println(counter++);
        System.out.println(counter);
        System.out.println(++counter);

        boolean p = true;
        boolean q = false;
        System.out.println(p && q);
        System.out.println(p || q);
        System.out.println(!p);
        System.out.println(a > b);
        System.out.println(a == 17);
    }
}
