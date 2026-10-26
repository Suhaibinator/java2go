package probe.shadow;

// Structural lookalike with no ancestry to java.lang.Enum.
public final class Enum {
    public String name() { return "IMPOSTER"; }
    public int ordinal() { return 0; }
    public Class<?> getDeclaringClass() { return Enum.class; }
}
