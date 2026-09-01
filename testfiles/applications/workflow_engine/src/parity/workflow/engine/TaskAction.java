package parity.workflow.engine;

import parity.workflow.model.ExecutionResult;
import parity.workflow.model.WorkflowTask;

public interface TaskAction<T> {
    ExecutionResult execute(WorkflowTask<T> task);
}
