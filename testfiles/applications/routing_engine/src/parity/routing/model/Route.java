package parity.routing.model;

public class Route {
    private String path;
    private int cost;
    private int hops;

    public Route(String path, int cost, int hops) {
        this.path = path;
        this.cost = cost;
        this.hops = hops;
    }

    public int cost() {
        return this.cost;
    }

    public int hops() {
        return this.hops;
    }

    public String summary() {
        return this.path + "|cost=" + this.cost + "|hops=" + this.hops;
    }
}
