class Box<T> {
    private T value;

    Box(T value) {
        this.value = value;
    }

    T get() {
        return value;
    }

    void set(T value) {
        this.value = value;
    }
}

class Pair<A, B> {
    private A first;
    private B second;

    Pair(A first, B second) {
        this.first = first;
        this.second = second;
    }

    A getFirst() {
        return first;
    }

    B getSecond() {
        return second;
    }
}

public class Generics {
    static <T> T identity(T value) {
        return value;
    }

    public static void main(String[] args) {
        Box<String> sb = new Box<String>("hello");
        System.out.println(sb.get());
        sb.set("world");
        System.out.println(sb.get());

        Box<Integer> ib = new Box<Integer>(42);
        System.out.println(ib.get());

        Pair<String, Integer> p = new Pair<String, Integer>("age", 30);
        System.out.println(p.getFirst() + "=" + p.getSecond());

        System.out.println(identity("direct"));
    }
}
