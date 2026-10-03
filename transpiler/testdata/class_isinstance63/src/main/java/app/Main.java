package app;

public class Main {
    interface Marker {}
    static class Parent implements Marker {}
    static class Child extends Parent {}
    static class Other {}
    static class Class { int isInstance(Object value) { return 7; } }
    static int effects;
    static java.lang.Class<?> receiver() { effects = effects * 10 + 1; return null; }
    static Object argument() { effects = effects * 10 + 2; return new Child(); }
    public static void main(String[] args) {
        Object leaf = new Child();
        Object nil = null;
        Child typedNil = null;
        System.out.println("nominal:" + Child.class.isInstance(leaf) + ":" + Parent.class.isInstance(leaf) + ":" + Marker.class.isInstance(leaf) + ":" + Other.class.isInstance(leaf));
        System.out.println("null:" + Parent.class.isInstance(nil) + ":" + Parent.class.isInstance(typedNil));
        System.out.println("boxing:" + Integer.class.isInstance(3) + ":" + Number.class.isInstance(3) + ":" + Integer.class.isInstance(3L));
        System.out.println("primitive:" + int.class.isInstance(3) + ":" + void.class.isInstance(leaf));
        Object strings = new String[0];
        Object ints = new int[0];
        System.out.println("arrays:" + Object[].class.isInstance(strings) + ":" + Object[].class.isInstance(ints) + ":" + int[].class.isInstance(ints) + ":" + long[].class.isInstance(ints));
        System.out.println("array-interfaces:" + Object.class.isInstance(ints) + ":" + Cloneable.class.isInstance(ints) + ":" + java.io.Serializable.class.isInstance(ints));
        System.out.println("shadow:" + new Class().isInstance(leaf));
        try { receiver().isInstance(argument()); } catch (NullPointerException expected) { System.out.println("null-receiver-effects:" + effects); }
    }
}
