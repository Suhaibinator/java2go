package parity.analytics.pipeline;

import parity.analytics.model.Event;

public abstract class ScorePolicy {
    private int minimumScore;

    public ScorePolicy(int minimumScore) {
        this.minimumScore = minimumScore;
    }

    protected int clamp(int value) {
        return Math.max(this.minimumScore, value);
    }

    public abstract int score(Event event);
}
