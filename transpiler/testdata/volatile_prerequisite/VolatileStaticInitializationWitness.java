public final class VolatileStaticInitializationWitness {
    static int events;
    static int record(int digit) { events = events * 10 + digit; return digit; }
    static class Base { static volatile int field = record(3); }
    static class Child extends Base { static { record(4); } }
    static class CompoundBase { static volatile int field = record(3); }
    static class CompoundChild extends CompoundBase { static { record(4); } }
    static Child qualifier() { record(1); return null; }
    static CompoundChild compoundQualifier() { record(1); return null; }
    static int rhs() { record(2); return 7; }
    public static void main(String[] args) {
        int simple = qualifier().field = rhs();
        System.out.println("simple|" + simple + "|" + events + "|" + Child.field);
        events = 0;
        int compound = compoundQualifier().field += rhs();
        System.out.println("compound|" + compound + "|" + events + "|" + CompoundChild.field);
    }
}
