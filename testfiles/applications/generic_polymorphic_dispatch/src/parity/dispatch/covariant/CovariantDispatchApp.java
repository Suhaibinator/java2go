package parity.dispatch.covariant;

public class CovariantDispatchApp {
    interface Value {}
    static class Item implements Value { int id; Item(int id) { this.id = id; } int read() { return id; } }
    static class Special extends Item { Special(int id) { super(id); } int read() { return id + 10; } }
    static class Other extends Item { Other() { super(9); } }
    static int bodies;
    static class Store<T extends Value> { Item exchange(T value) { bodies++; return new Item(1); } }
    static class Specialized extends Store<Special> {
        Special exchange(Special value) { bodies += 10; return value; }
    }
    public static String run() {
        Specialized store = new Specialized();
        Store<Special> base = store;
        Store raw = store;
        Item result = base.exchange(new Special(3));
        String outcome;
        try { raw.exchange(new Other()); outcome = "missing"; }
        catch (ClassCastException expected) { outcome = "cast"; }
        boolean empty = base.exchange(null) == null;
        return result.read() + ":" + bodies + ":" + outcome + ":" + empty;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
