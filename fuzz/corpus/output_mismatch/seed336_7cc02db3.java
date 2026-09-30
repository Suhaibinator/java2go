// seed: 336
public class Gen {
    public static void main(String[] args) {
        String s0 = "::";
        s0 += -2 + 0xFFFFFFFF + "sum=";
        s0 += -2147483648 + 0xFFFFFFFF + "out";
        StringBuilder sb2 = new StringBuilder();
        sb2.append(-7);
        sb2.append("alpha");
        sb2.append("val");
        String s1 = sb2.toString();
        System.out.println(s0);
        System.out.println(s1);
    }
}
