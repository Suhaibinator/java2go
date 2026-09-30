package repro;
public class Main {
    static int value;
    static void assign(int value) { Main.value = value; }
    public static void main(String[] args) {
        assign(7);
        System.out.println(Main.value);
    }
}
