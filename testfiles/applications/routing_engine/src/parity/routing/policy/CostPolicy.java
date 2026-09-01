package parity.routing.policy;

import parity.routing.model.Edge;

public interface CostPolicy {
    int cost(Edge edge);
    String name();
}
