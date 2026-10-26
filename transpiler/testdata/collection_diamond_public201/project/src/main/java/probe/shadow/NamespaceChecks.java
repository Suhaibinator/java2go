package probe.shadow;

public final class NamespaceChecks {
    public static int run(int seed) {
        ArrayList list = new ArrayList(seed);
        Collection marker = new Collection(seed);
        return list.value() + marker.value();
    }
}
