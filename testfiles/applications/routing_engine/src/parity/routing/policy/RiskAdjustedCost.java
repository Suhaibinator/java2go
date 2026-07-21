package parity.routing.policy;

import parity.routing.model.Edge;

public class RiskAdjustedCost implements CostPolicy {
    public int cost(Edge edge) {
        int tollPenalty = edge.toll() ? 5 : 0;
        return edge.baseCost() + tollPenalty + edge.risk();
    }

    public String name() {
        return "risk-adjusted";
    }
}
