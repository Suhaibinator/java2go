package parity.workflow.engine;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import parity.workflow.model.ExecutionResult;
import parity.workflow.model.ResultType;
import parity.workflow.model.TaskState;
import parity.workflow.model.WorkflowTask;
import parity.workflow.rules.RuleResult;
import parity.workflow.rules.TaskRule;

public class WorkflowEngine<T> {
    private final String name;
    private final int maximumRetries;
    private final List<WorkflowTask<T>> tasks;
    private final Map<String, WorkflowTask<T>> tasksById;
    private final List<TaskRule<T>> rules;
    private final Map<String, Integer> metrics;

    public WorkflowEngine(String name, int maximumRetries) {
        this.name = name;
        this.maximumRetries = maximumRetries;
        this.tasks = new ArrayList<WorkflowTask<T>>();
        this.tasksById = new HashMap<String, WorkflowTask<T>>();
        this.rules = new ArrayList<TaskRule<T>>();
        this.metrics = new HashMap<String, Integer>();
    }

    public void addRule(TaskRule<T> rule) {
        rules.add(rule);
    }

    public void addTask(WorkflowTask<T> task) {
        tasks.add(task);
        tasksById.put(task.getId(), task);
    }

    public WorkflowReport run(TaskAction<T> action) {
        WorkflowReport report = new WorkflowReport();
        List<WorkflowTask<T>> plan = buildPlan();

        report.add("NAME " + name);
        report.add("TASKS " + tasks.size());
        report.add("PLAN " + planText(plan));
        for (WorkflowTask<T> task : plan) {
            report.add("CONFIG " + task.getId()
                    + " priority=" + task.getPriority().name()
                    + " effort=" + task.getEffort()
                    + " deps=" + task.getDependencies().size());
        }

        int pass = 1;
        boolean progress = true;
        while (progress) {
            progress = false;
            report.add("PASS " + pass);

            for (WorkflowTask<T> task : plan) {
                if (task.getState() == TaskState.REGISTERED) {
                    String missing = firstMissingDependency(task);
                    if (!missing.equals("")) {
                        task.markSkipped();
                        report.add("SKIP " + task.getId() + " reason=missing:" + missing);
                        bump("skipped");
                        progress = true;
                        continue;
                    }

                    String failed = firstFailedDependency(task);
                    if (!failed.equals("")) {
                        task.markSkipped();
                        report.add("SKIP " + task.getId() + " reason=upstream:" + failed);
                        bump("skipped");
                        progress = true;
                        continue;
                    }

                    if (allDependenciesSucceeded(task)) {
                        task.markReady();
                        progress = true;

                        String rejection = firstRejection(task);
                        if (!rejection.equals("")) {
                            task.markSkipped();
                            report.add("SKIP " + task.getId() + " reason=rule:" + rejection);
                            bump("skipped");
                            bump("rejections");
                            continue;
                        }
                    }
                }

                if (task.getState() == TaskState.READY) {
                    execute(task, action, report);
                    progress = true;
                }
            }
            pass++;
        }

        for (WorkflowTask<T> task : plan) {
            if (task.getState() == TaskState.REGISTERED || task.getState() == TaskState.READY) {
                task.markSkipped();
                report.add("SKIP " + task.getId() + " reason=unresolved-cycle");
                bump("skipped");
            }
        }

        appendSummary(plan, report);
        return report;
    }

    private void execute(WorkflowTask<T> task, TaskAction<T> action, WorkflowReport report) {
        task.startAttempt();
        bump("executions");
        report.add("RUN " + task.getId() + " attempt=" + task.getAttempts());

        ExecutionResult result = action.execute(task);
        if (result.getType() == ResultType.SUCCESS) {
            task.markSucceeded();
            report.add("OK " + task.getId() + " code=" + result.getCode()
                    + " detail=" + result.getDetail());
        } else if (result.getType() == ResultType.RETRYABLE_FAILURE
                && task.getAttempts() <= maximumRetries) {
            task.markRetry();
            bump("retries");
            report.add("RETRY " + task.getId() + " code=" + result.getCode()
                    + " detail=" + result.getDetail());
        } else {
            task.markFailed();
            report.add("FAIL " + task.getId() + " code=" + result.getCode()
                    + " detail=" + result.getDetail());
        }
    }

    private String firstMissingDependency(WorkflowTask<T> task) {
        for (String dependencyId : task.getDependencies()) {
            if (!tasksById.containsKey(dependencyId)) {
                return dependencyId;
            }
        }
        return "";
    }

    private String firstFailedDependency(WorkflowTask<T> task) {
        for (String dependencyId : task.getDependencies()) {
            WorkflowTask<T> dependency = tasksById.get(dependencyId);
            if (dependency.getState() == TaskState.FAILED
                    || dependency.getState() == TaskState.SKIPPED) {
                return dependencyId;
            }
        }
        return "";
    }

    private boolean allDependenciesSucceeded(WorkflowTask<T> task) {
        for (String dependencyId : task.getDependencies()) {
            WorkflowTask<T> dependency = tasksById.get(dependencyId);
            if (dependency == null || dependency.getState() != TaskState.SUCCEEDED) {
                return false;
            }
        }
        return true;
    }

    private String firstRejection(WorkflowTask<T> task) {
        for (TaskRule<T> rule : rules) {
            RuleResult result = rule.evaluate(task);
            if (!result.isAllowed()) {
                return rule.name() + ":" + result.getReason();
            }
        }
        return "";
    }

    private List<WorkflowTask<T>> buildPlan() {
        List<WorkflowTask<T>> plan = new ArrayList<WorkflowTask<T>>();
        for (WorkflowTask<T> task : tasks) {
            plan.add(task);
        }

        for (int i = 0; i < plan.size(); i++) {
            int best = i;
            for (int j = i + 1; j < plan.size(); j++) {
                if (comesBefore(plan.get(j), plan.get(best))) {
                    best = j;
                }
            }
            if (best != i) {
                WorkflowTask<T> previous = plan.get(i);
                plan.set(i, plan.get(best));
                plan.set(best, previous);
            }
        }
        return plan;
    }

    private boolean comesBefore(WorkflowTask<T> left, WorkflowTask<T> right) {
        if (left.getPriority().getWeight() != right.getPriority().getWeight()) {
            return left.getPriority().getWeight() > right.getPriority().getWeight();
        }
        return left.getId().compareTo(right.getId()) < 0;
    }

    private String planText(List<WorkflowTask<T>> plan) {
        StringBuilder text = new StringBuilder();
        for (int i = 0; i < plan.size(); i++) {
            if (i > 0) {
                text.append(">");
            }
            text.append(plan.get(i).getId());
        }
        return text.toString();
    }

    private void bump(String key) {
        metrics.put(key, metrics.getOrDefault(key, 0) + 1);
    }

    private void appendSummary(List<WorkflowTask<T>> plan, WorkflowReport report) {
        Map<TaskState, Integer> stateCounts = new HashMap<TaskState, Integer>();
        for (TaskState state : TaskState.values()) {
            stateCounts.put(state, 0);
        }

        int attempts = 0;
        int checksum = 17;
        for (WorkflowTask<T> task : plan) {
            TaskState state = task.getState();
            stateCounts.put(state, stateCounts.get(state) + 1);
            attempts += task.getAttempts();
            checksum = checksum * 31 + task.getId().length();
            checksum = checksum * 31 + state.ordinal();
            checksum = checksum * 31 + task.getAttempts();
        }

        report.add("SUMMARY executions=" + metrics.getOrDefault("executions", 0)
                + " retries=" + metrics.getOrDefault("retries", 0)
                + " rejections=" + metrics.getOrDefault("rejections", 0)
                + " skipped=" + metrics.getOrDefault("skipped", 0));
        for (TaskState state : TaskState.values()) {
            report.add("STATE " + state.name() + "=" + stateCounts.get(state));
        }
        report.add("ATTEMPTS " + attempts);
        for (WorkflowTask<T> task : plan) {
            report.add("HISTORY " + task.getId()
                    + " state=" + task.getState().name()
                    + " attempts=" + task.getAttempts()
                    + " path=" + task.historyText());
        }
        report.add("CHECKSUM " + checksum);
    }
}
