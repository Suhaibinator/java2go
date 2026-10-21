package app;
import java.util.List;
import java.util.ArrayList;
import bridge.Box;
import model.Foo;

public class Main {
    static int receivers, arguments, cleanups;
    static String trace = "";

    static Box<List<Foo>> receiver(Box<List<Foo>> value) {
        receivers++;
        trace += "R";
        return value;
    }

    static List<Foo> argument(List<Foo> value) {
        arguments++;
        trace += "A";
        return value;
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Box<List<Foo>> box = new Box<List<Foo>>(seed);
        List<Foo> values = new ArrayList<Foo>();
        Foo first = new Foo(seed);
        values.add(first);
        values.add(new Foo(seed + 1));
        List<Foo> seen = receiver(box).identity(argument(values));
        seen.add(new Foo(seed + 2));
        System.out.println("seed=" + seed);
        System.out.println("identity=" + (seen == values) + ":" + (seen.get(0) == first) + ":" + values.size() + ":" + seen.get(0).id + ":" + values.get(2).id + ":" + receivers + ":" + arguments + ":" + box.calls + ":" + trace + ":" + box.trace);

        List<Foo> nullable = receiver(box).identity(argument(null));
        System.out.println("null-value=" + (nullable == null) + ":" + receivers + ":" + arguments + ":" + box.calls + ":" + trace + ":" + box.trace);

        boolean nullReceiver = false;
        try {
            List<Foo> ignored = receiver(null).identity(argument(values));
            trace += "X";
        } catch (NullPointerException expected) {
            nullReceiver = true;
            trace += "C";
        } finally {
            cleanups++;
            trace += "F";
        }
        System.out.println("null-receiver=" + nullReceiver + ":" + receivers + ":" + arguments + ":" + box.calls + ":" + cleanups + ":" + trace + ":" + box.trace);

        List<List<Foo>> outer = new ArrayList<List<Foo>>();
        outer.add(values);
        Box<List<List<Foo>>> nestedBox = new Box<List<List<Foo>>>(seed);
        List<List<Foo>> nested = nestedBox.identity(outer);
        System.out.println("nested=" + (nested == outer) + ":" + (nested.get(0) == values) + ":" + nested.get(0).get(1).id + ":" + nestedBox.calls + ":" + nestedBox.trace);
        System.out.println("original-call=" + (Box.call(box, values) == values) + ":" + box.calls + ":" + box.trace);

        Object wide = box.<List<Foo>>polluted(values);
        System.out.println("wide-pollution=" + (((Integer)wide).intValue() == seed) + ":" + box.calls + ":" + box.trace);
        boolean narrowCast = false;
        try {
            List<Foo> ignored = receiver(box).polluted(argument(values));
            trace += "X";
        } catch (ClassCastException expected) {
            narrowCast = true;
            trace += "C";
        } finally {
            cleanups++;
            trace += "F";
        }
        System.out.println("narrow-pollution=" + narrowCast + ":" + receivers + ":" + arguments + ":" + box.calls + ":" + cleanups + ":" + trace + ":" + box.trace);

        Integer numeric = box.<Integer>numeric(Integer.valueOf(seed + 3));
        boolean numberCast = false;
        try {
            Number ignored = box.numeric(first);
        } catch (ClassCastException expected) {
            numberCast = true;
        }
        Foo declared = box.<Foo>declared(first);
        boolean sourceCast = false;
        try {
            Foo ignored = box.declared(Integer.valueOf(seed));
        } catch (ClassCastException expected) {
            sourceCast = true;
        }
        System.out.println("declared-bounds=" + numeric.intValue() + ":" + numberCast + ":" + (declared == first) + ":" + sourceCast + ":" + box.calls + ":" + box.trace);

        shadow.List<Foo> source = new shadow.List<Foo>(seed + 4);
        Box<shadow.List<Foo>> sourceBox = new Box<shadow.List<Foo>>(seed);
        shadow.List<Foo> sourceSeen = sourceBox.identity(source);
        System.out.println("source-list=" + (sourceSeen == source) + ":" + sourceSeen.code + ":" + sourceBox.calls + ":" + sourceBox.trace);
    }
}
