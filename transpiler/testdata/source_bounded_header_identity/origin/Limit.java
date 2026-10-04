package origin;
public class Limit {
    public int seed;
    public int calls;
    public Limit(int seed) { this.seed = seed; }
    public int read() { return seed + ++calls; }
}
