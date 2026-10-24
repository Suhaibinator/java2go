package app;
import bridge.Box;
import bridge.Controls;
import model.Foo;
import model.Cleanup;

public class Main {
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Box<Foo> box = new Box<Foo>(seed);
        Foo value = new Foo(seed);
        System.out.println("seed=" + seed);
        Foo first = Controls.receiver(box).identity(Controls.argument(value));
        first.id++;
        System.out.println("pointer=" + (first == value) + ":" + value.id + ":" + Controls.receivers + ":" + Controls.arguments + ":" + box.calls + ":" + Controls.trace + ":" + box.trace);
        Foo empty = Controls.receiver(box).identity(Controls.argument(null));
        System.out.println("nullable=" + (empty == null) + ":" + Controls.receivers + ":" + Controls.arguments + ":" + box.calls + ":" + Controls.trace + ":" + box.trace);
        Foo relayed = Box.relay(box, value);
        Foo privateValue = box.throughPrivate(value);
        Foo staticValue = Box.throughStatic(value);
        System.out.println("typed-controls=" + (relayed == value) + ":" + (privateValue == value) + ":" + (staticValue == value) + ":" + box.calls + ":" + Box.staticCalls + ":" + box.trace + ":" + Box.staticTrace);

        Integer wrapper = Integer.valueOf(seed + 1);
        Integer kept = box.identity(wrapper);
        int unboxed = box.identity(wrapper);
        Integer autoboxed = Controls.receiver(box).identity(Controls.primitive(seed + 2));
        System.out.println("integer=" + (kept == wrapper) + ":" + unboxed + ":" + autoboxed.intValue() + ":" + Controls.receivers + ":" + Controls.arguments + ":" + box.calls + ":" + Controls.trace + ":" + box.trace);
        boolean bool = box.identity(true);
        byte small = box.identity((byte)7);
        short medium = box.identity((short)(seed + 3));
        char character = box.identity((char)('A' + seed % 5));
        long wide = box.identity((long)seed + 4L);
        float fractional = box.identity((float)seed + 0.5f);
        double precise = box.identity((double)seed + 0.25d);
        System.out.println("primitives=" + bool + ":" + small + ":" + medium + ":" + character + ":" + wide + ":" + fractional + ":" + precise + ":" + box.calls + ":" + box.trace);
        Boolean booleanObject = Boolean.valueOf(true);
        Byte byteObject = Byte.valueOf((byte)7);
        Short shortObject = Short.valueOf((short)(seed + 3));
        Character charObject = Character.valueOf((char)('A' + seed % 5));
        Long longObject = Long.valueOf((long)seed + 4L);
        Float floatObject = Float.valueOf((float)seed + 0.5f);
        Double doubleObject = Double.valueOf((double)seed + 0.25d);
        System.out.println("wrapper-identity=" + (box.identity(booleanObject) == booleanObject) + ":" + (box.identity(byteObject) == byteObject) + ":" + (box.identity(shortObject) == shortObject) + ":" + (box.identity(charObject) == charObject) + ":" + (box.identity(longObject) == longObject) + ":" + (box.identity(floatObject) == floatObject) + ":" + (box.identity(doubleObject) == doubleObject) + ":" + box.calls + ":" + box.trace);
        Integer nullable = box.identity((Integer)null);
        System.out.println("null-wrapper=" + (nullable == null) + ":" + box.calls + ":" + box.trace);
        try { int wrong = box.identity((Integer)null); System.out.println("wrong-unbox=" + wrong); }
        catch (NullPointerException expected) { Controls.trace += "U"; System.out.println("unbox-null=" + box.calls + ":" + Controls.trace + ":" + box.trace); }
        finally { Controls.trace += "F"; }

        try (Cleanup left = new Cleanup("A"); Cleanup right = new Cleanup("B")) {
            Foo wrong = Controls.receiver(null).identity(Controls.argument(value));
            System.out.println("wrong-null-receiver=" + wrong.id);
        } catch (NullPointerException expected) {
            Controls.trace += "C";
            System.out.println("receiver-null=" + Controls.receivers + ":" + Controls.arguments + ":" + box.calls + ":" + Controls.trace + ":" + Cleanup.opens + ":" + Cleanup.closes + ":" + Cleanup.trace + ":" + box.trace);
        } finally { Controls.trace += "F"; }
        try { Foo wrong = Controls.receiver(null).identity(Controls.failingArgument(value)); System.out.println("wrong-argument=" + wrong.id); }
        catch (IllegalStateException expected) { Controls.trace += "Q"; System.out.println("argument-before-null=" + expected.getMessage() + ":" + Controls.receivers + ":" + Controls.arguments + ":" + box.calls + ":" + Controls.trace + ":" + box.trace); }
        finally { Controls.trace += "F"; }

        Object broader = Controls.receiver(box).polluted(Controls.argument(value));
        System.out.println("wide-pollution=" + (broader instanceof foreign.Foo) + ":" + ((foreign.Foo)broader).id + ":" + box.calls + ":" + Controls.trace + ":" + box.trace);
        try (Cleanup left = new Cleanup("D"); Cleanup right = new Cleanup("E")) {
            Foo wrong = Controls.receiver(box).polluted(Controls.argument(value));
            System.out.println("wrong-pollution=" + wrong.id);
        } catch (ClassCastException expected) {
            Controls.trace += "C";
            System.out.println("narrow-pollution=" + Controls.receivers + ":" + Controls.arguments + ":" + box.calls + ":" + Controls.trace + ":" + Cleanup.opens + ":" + Cleanup.closes + ":" + Cleanup.trace + ":" + box.trace);
        } finally { Controls.trace += "F"; }
        try { int wrong = box.wrapperPolluted(wrapper); System.out.println("wrong-wrapper=" + wrong); }
        catch (ClassCastException expected) { System.out.println("wrapper-pollution=" + box.calls + ":" + box.trace); }

        Foo declared = box.<Foo>declared(value);
        Foo declaredNull = box.<Foo>declared(null);
        System.out.println("declared=" + (declared == value) + ":" + (declaredNull == null) + ":" + box.calls + ":" + box.trace);
        try (Cleanup left = new Cleanup("G"); Cleanup right = new Cleanup("H")) {
            Foo wrong = box.<Foo>declared(new foreign.Foo(seed));
            System.out.println("wrong-declared=" + wrong.id);
        } catch (ClassCastException expected) {
            System.out.println("body-cast=" + box.calls + ":" + Cleanup.opens + ":" + Cleanup.closes + ":" + Cleanup.trace + ":" + box.trace);
        } finally { Cleanup.trace += "F"; }
        try (Cleanup left = new Cleanup("J"); Cleanup right = new Cleanup("K")) {
            Foo wrong = box.throwing(value);
            System.out.println("wrong-body=" + wrong.id);
        } catch (IllegalStateException expected) {
            System.out.println("body-exception=" + expected.getMessage() + ":" + box.calls + ":" + Cleanup.opens + ":" + Cleanup.closes + ":" + Cleanup.trace + ":" + box.trace);
        } finally { Cleanup.trace += "F"; }
        System.out.println("final-state=" + value.id + ":" + box.marker + ":" + box.calls + ":" + Box.staticCalls + ":" + Controls.receivers + ":" + Controls.arguments + ":" + Controls.trace + ":" + Cleanup.opens + ":" + Cleanup.closes + ":" + Cleanup.trace + ":" + box.trace + ":" + Box.staticTrace);
    }
}
