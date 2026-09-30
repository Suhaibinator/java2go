package parity.routing.app;

import parity.routing.algorithm.RoutePlanner;
import parity.routing.model.Graph;
import parity.routing.model.NetworkFactory;
import parity.routing.model.Route;
import parity.routing.policy.CostPolicy;
import parity.routing.policy.RiskAdjustedCost;
import parity.routing.policy.StandardCost;

public class RoutingApplication {
    private static Route printRoute(Graph graph, int start, int target, CostPolicy policy) {
        Route route = RoutePlanner.find(graph, start, target, policy);
        System.out.println(policy.name() + "|" + route.summary());
        return route;
    }

    public static void main(String[] args) {
        Graph graph = NetworkFactory.sampleNetwork();
        System.out.println("ROUTING REPORT");
        System.out.println(graph.describe());

        Route standard = printRoute(graph, 0, 5, new StandardCost());
        Route riskAdjusted = printRoute(graph, 0, 5, new RiskAdjustedCost());
        Route unreachable = printRoute(graph, 5, 0, new StandardCost());

        int checksum = standard.cost() + standard.hops()
                + riskAdjusted.cost() + riskAdjusted.hops()
                + unreachable.cost() + unreachable.hops();
        System.out.println("checksum=" + checksum);
    }
}
