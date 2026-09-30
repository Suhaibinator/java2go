package parity.routing.model;

public class Edge {
    private int from;
    private int to;
    private int baseCost;
    private boolean toll;
    private int risk;

    public Edge(int from, int to, int baseCost, boolean toll, int risk) {
        this.from = from;
        this.to = to;
        this.baseCost = baseCost;
        this.toll = toll;
        this.risk = risk;
    }

    public int from() {
        return this.from;
    }

    public int to() {
        return this.to;
    }

    public int baseCost() {
        return this.baseCost;
    }

    public boolean toll() {
        return this.toll;
    }

    public int risk() {
        return this.risk;
    }
}
