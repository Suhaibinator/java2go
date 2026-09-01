package parity.routing.model;

public class NetworkFactory {
    public static Graph sampleNetwork() {
        Edge[] edges = new Edge[10];
        edges[0] = new Edge(0, 1, 4, false, 2);
        edges[1] = new Edge(0, 2, 2, true, 1);
        edges[2] = new Edge(2, 1, 1, false, 3);
        edges[3] = new Edge(1, 3, 5, false, 1);
        edges[4] = new Edge(2, 3, 8, false, 0);
        edges[5] = new Edge(2, 4, 10, true, 4);
        edges[6] = new Edge(3, 4, 2, false, 2);
        edges[7] = new Edge(3, 5, 6, true, 2);
        edges[8] = new Edge(4, 5, 3, false, 1);
        edges[9] = new Edge(1, 5, 15, false, 0);
        return new Graph(7, edges);
    }
}
