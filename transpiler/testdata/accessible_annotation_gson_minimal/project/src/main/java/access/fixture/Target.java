package access.fixture;
public final class Target {
    public int publicValue = 11;
    private int privateValue = 17;
    public static int publicStatic = 23;
    private Target() {}
    public static Target fresh() { return new Target(); }
    public int publicRead() { return publicValue; }
    private int privateRead() { return privateValue; }
}
