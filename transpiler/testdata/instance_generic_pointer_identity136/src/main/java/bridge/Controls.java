package bridge;
import model.Foo;
public class Controls {
    public static int receivers;
    public static int arguments;
    public static String trace = "";
    public static Box<Foo> receiver(Box<Foo> value) { receivers++; trace += "R"; return value; }
    public static Foo argument(Foo value) { arguments++; trace += "A"; return value; }
    public static int primitive(int value) { arguments++; trace += "N"; return value; }
    public static Foo failingArgument(Foo value) { arguments++; trace += "X"; throw new IllegalStateException("argument"); }
}
