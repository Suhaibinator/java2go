package parity.workflow.engine;

import java.util.ArrayList;
import java.util.List;

public class WorkflowReport {
    private final List<String> lines;

    public WorkflowReport() {
        this.lines = new ArrayList<String>();
    }

    public void add(String line) {
        lines.add(line);
    }

    public int size() {
        return lines.size();
    }

    public void print() {
        for (String line : lines) {
            System.out.println(line);
        }
    }
}
