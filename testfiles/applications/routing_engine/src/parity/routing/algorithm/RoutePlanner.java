package parity.routing.algorithm;

import parity.routing.model.Edge;
import parity.routing.model.Graph;
import parity.routing.model.Route;
import parity.routing.policy.CostPolicy;

public class RoutePlanner {
    private static final int UNREACHABLE = 1000000000;

    public static Route find(Graph graph, int start, int target, CostPolicy policy) {
        int[] distances = new int[graph.nodeCount()];
        int[] hops = new int[graph.nodeCount()];
        String[] paths = new String[graph.nodeCount()];

        for (int i = 0; i < graph.nodeCount(); i++) {
            distances[i] = UNREACHABLE;
            hops[i] = 0;
            paths[i] = "";
        }
        distances[start] = 0;
        paths[start] = "" + start;

        for (int pass = 0; pass < graph.nodeCount() - 1; pass++) {
            boolean changed = false;
            Edge[] edges = graph.edges();
            for (int i = 0; i < edges.length; i++) {
                Edge edge = edges[i];
                int from = edge.from();
                int to = edge.to();
                if (distances[from] < UNREACHABLE) {
                    int candidateCost = distances[from] + policy.cost(edge);
                    int candidateHops = hops[from] + 1;
                    String candidatePath = paths[from] + "->" + to;
                    boolean lowerCost = candidateCost < distances[to];
                    boolean stableTie = candidateCost == distances[to]
                            && (paths[to].equals("") || candidatePath.compareTo(paths[to]) < 0);
                    if (lowerCost || stableTie) {
                        distances[to] = candidateCost;
                        hops[to] = candidateHops;
                        paths[to] = candidatePath;
                        changed = true;
                    }
                }
            }
            if (!changed) {
                break;
            }
        }

        if (distances[target] == UNREACHABLE) {
            return new Route("unreachable", -1, 0);
        }
        return new Route(paths[target], distances[target], hops[target]);
    }
}
