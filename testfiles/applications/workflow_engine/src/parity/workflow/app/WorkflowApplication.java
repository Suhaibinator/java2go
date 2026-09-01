package parity.workflow.app;

import parity.workflow.engine.TaskAction;
import parity.workflow.engine.WorkflowEngine;
import parity.workflow.engine.WorkflowReport;
import parity.workflow.model.ExecutionResult;
import parity.workflow.model.Priority;
import parity.workflow.model.WorkflowTask;
import parity.workflow.rules.EffortLimitRule;
import parity.workflow.rules.PayloadPrefixRule;

public class WorkflowApplication {
    public static void main(String[] args) {
        WorkflowEngine<String> engine = new WorkflowEngine<String>("nightly-sync", 1);
        engine.addRule(new EffortLimitRule<String>(5));
        engine.addRule(new PayloadPrefixRule("BLOCK:"));

        WorkflowTask<String> publish = new WorkflowTask<String>(
                "publish", Priority.CRITICAL, 2, "FAIL:remote-rejected");
        publish.dependsOn("normalize");

        WorkflowTask<String> normalize = new WorkflowTask<String>(
                "normalize", Priority.HIGH, 3, "RETRY:normalized");
        normalize.dependsOn("fetch");

        WorkflowTask<String> oversize = new WorkflowTask<String>(
                "oversize", Priority.HIGH, 9, "OK:bulk");
        oversize.dependsOn("fetch");

        WorkflowTask<String> quarantine = new WorkflowTask<String>(
                "quarantine", Priority.HIGH, 1, "BLOCK:unsafe");
        quarantine.dependsOn("fetch");

        WorkflowTask<String> cycleA = new WorkflowTask<String>(
                "cycle-a", Priority.NORMAL, 1, "OK:cycle-a");
        cycleA.dependsOn("cycle-b");

        WorkflowTask<String> cycleB = new WorkflowTask<String>(
                "cycle-b", Priority.NORMAL, 1, "OK:cycle-b");
        cycleB.dependsOn("cycle-a");

        WorkflowTask<String> enrich = new WorkflowTask<String>(
                "enrich", Priority.NORMAL, 4, "OK:enriched");
        enrich.dependsOn("normalize");

        WorkflowTask<String> orphan = new WorkflowTask<String>(
                "orphan", Priority.NORMAL, 1, "OK:orphan");
        orphan.dependsOn("absent");

        WorkflowTask<String> archive = new WorkflowTask<String>(
                "archive", Priority.LOW, 2, "OK:archived");
        archive.dependsOn("publish");

        WorkflowTask<String> cleanup = new WorkflowTask<String>(
                "cleanup", Priority.LOW, 1, "OK:cleaned");
        cleanup.dependsOn("fetch");
        cleanup.dependsOn("normalize");

        WorkflowTask<String> fetch = new WorkflowTask<String>(
                "fetch", Priority.LOW, 2, "OK:fetched");

        engine.addTask(archive);
        engine.addTask(fetch);
        engine.addTask(normalize);
        engine.addTask(publish);
        engine.addTask(enrich);
        engine.addTask(cleanup);
        engine.addTask(oversize);
        engine.addTask(quarantine);
        engine.addTask(orphan);
        engine.addTask(cycleB);
        engine.addTask(cycleA);

        TaskAction<String> action = task -> {
            String payload = task.getPayload();
            if (payload.startsWith("RETRY:") && task.getAttempts() == 1) {
                return ExecutionResult.retryable(503, "cache-warming");
            }
            if (payload.startsWith("FAIL:")) {
                return ExecutionResult.failure(422, payload.substring(5));
            }
            if (payload.startsWith("RETRY:")) {
                return ExecutionResult.success(200, payload.substring(6));
            }
            return ExecutionResult.success(200, payload.substring(3));
        };

        System.out.println("=== WORKFLOW PARITY ===");
        WorkflowReport report = engine.run(action);
        report.print();
        System.out.println("REPORT_LINES " + report.size());
        System.out.println("=== END WORKFLOW ===");
    }
}
