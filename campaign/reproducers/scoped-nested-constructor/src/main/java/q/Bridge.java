package q;
import p.Outer;
import java.util.function.Function;
public class Bridge {
    public static String suffix() { return "c"; }
    public static void main(String[] args) {
        Outer.Inner codec = new Outer.Inner();
        Function<String, String> reference = codec::choose;
        String first = reference.apply("a");
        String second = codec.choose("b", 1);
        System.out.println(first + second + Outer.suffix());
    }
}
