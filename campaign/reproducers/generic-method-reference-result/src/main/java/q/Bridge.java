package q;
import p.Outer;
import p.Outer.Inner;
import java.util.function.Function;
public class Bridge {
    public static String suffix() { return "c"; }
    public static void main(String[] args) {
        Inner codec = new Inner();
        Function<String, String> reference = codec::choose;
        String first = reference.apply("a");
        String second = codec.choose("b", 1);
        System.out.println(first + second + Outer.suffix());
    }
}
