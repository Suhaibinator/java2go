package parity.routing.model;

public class Graph {
    private int nodeCount;
    private Edge[] edges;
    private int edgeCount;

    public Graph(int nodeCount, Edge[] edges) {
        this.nodeCount = nodeCount;
        this.edges = edges;
        this.edgeCount = edges.length;
    }

    public int nodeCount() {
        return this.nodeCount;
    }

    public Edge[] edges() {
        return this.edges;
    }

    public String describe() {
        return "nodes=" + this.nodeCount + ",edges=" + this.edgeCount;
    }
}
