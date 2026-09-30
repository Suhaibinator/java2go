package parity.routing.policy;

import parity.routing.model.Edge;

public class StandardCost implements CostPolicy {
    public int cost(Edge edge) {
        return edge.baseCost();
    }

    public String name() {
        return "standard";
    }
}
