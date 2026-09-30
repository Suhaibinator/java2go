package parity.dispatch.factory;

public class FactoryDispatchApp {
    interface Factory { Base make(boolean empty); }
    interface Operation { Base get(boolean empty); }
    static class Base implements Factory {
        public Base make(boolean empty) { bodies++; return empty ? null : this; }
        int kind() { return 1; }
    }
    static class Child extends Base {
        public Child make(boolean empty) { bodies += 10; return empty ? null : this; }
        int kind() { return 2; }
    }
    static int bodies;
    public static String run() {
        Child child = new Child();
        Base base = child;
        Factory factory = child;
        Operation bound = base::make;
        Base one = base.make(false);
        Child two = child.make(false);
        Base three = factory.make(false);
        Base four = bound.get(false);
        Object objectOne = one;
        Object objectTwo = two;
        Object objectThree = three;
        boolean same = one == two && two == three && three == four
                && objectOne.hashCode() == objectTwo.hashCode() && objectTwo.hashCode() == objectThree.hashCode();
        boolean empty = base.make(true) == null && factory.make(true) == null;
        return one.kind() + ":" + same + ":" + empty + ":" + bodies;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
