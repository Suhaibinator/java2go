import static java.util.Objects.hash;
public final class ImportedObjectsHashProbe {
    static final class Objects {
        static int hash(int value) { return 700 + value; }
    }
    public static String run() {
        Object[] values = new Object[] {Integer.valueOf(2), null, "Ω😀"};
        return hash(1, 2) + ":" + (hash(values) == java.util.Objects.hash(2, (Object) null, "Ω😀")) + ":" + hash() + ":" + hash((Object[]) null) + ":" + Objects.hash(3) + ":" + java.util.Objects.hash(3);
    }
}
