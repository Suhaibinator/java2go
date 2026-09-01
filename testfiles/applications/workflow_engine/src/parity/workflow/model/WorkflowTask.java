package parity.workflow.model;

import java.util.ArrayList;
import java.util.List;

public class WorkflowTask<T> {
    private final String id;
    private final Priority priority;
    private final int effort;
    private final T payload;
    private final List<String> dependencies;
    private final List<String> history;
    private TaskState state;
    private int attempts;

    public WorkflowTask(String id, Priority priority, int effort, T payload) {
        this.id = id;
        this.priority = priority;
        this.effort = effort;
        this.payload = payload;
        this.dependencies = new ArrayList<String>();
        this.history = new ArrayList<String>();
        this.state = TaskState.REGISTERED;
        this.attempts = 0;
        history.add(state.name());
    }

    public WorkflowTask<T> dependsOn(String dependencyId) {
        dependencies.add(dependencyId);
        return this;
    }

    public String getId() {
        return id;
    }

    public Priority getPriority() {
        return priority;
    }

    public int getEffort() {
        return effort;
    }

    public T getPayload() {
        return payload;
    }

    public List<String> getDependencies() {
        return dependencies;
    }

    public TaskState getState() {
        return state;
    }

    public int getAttempts() {
        return attempts;
    }

    public void markReady() {
        transition(TaskState.READY);
    }

    public void startAttempt() {
        attempts++;
        transition(TaskState.RUNNING);
    }

    public void markSucceeded() {
        transition(TaskState.SUCCEEDED);
    }

    public void markRetry() {
        transition(TaskState.READY);
    }

    public void markFailed() {
        transition(TaskState.FAILED);
    }

    public void markSkipped() {
        transition(TaskState.SKIPPED);
    }

    private void transition(TaskState next) {
        state = next;
        history.add(next.name());
    }

    public String historyText() {
        StringBuilder text = new StringBuilder();
        for (int i = 0; i < history.size(); i++) {
            if (i > 0) {
                text.append(">");
            }
            text.append(history.get(i));
        }
        return text.toString();
    }
}
