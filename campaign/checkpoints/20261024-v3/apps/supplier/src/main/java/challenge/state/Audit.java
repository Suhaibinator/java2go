package challenge.state;

public final class Audit {
    public final IllegalArgumentException failure = new IllegalArgumentException("mapper-stop");
    public int realCalls;
    public int lambdaCalls;
    public int localCalls;
    public int sourceRoutes;
    public int objectRoutes;
    public int canonicalBodyDonors;
    public int classifiers;
    public int values;
}
