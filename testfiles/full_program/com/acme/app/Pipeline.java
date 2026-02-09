package com.acme.app;

import com.acme.common.Mapper;
import com.acme.domain.ParseTask;
import com.acme.domain.Task;

public class Pipeline {
    public static int execute(Task task, Mapper<String, String> mapper) {
        String out = mapper.map(task.name());
        if (task instanceof ParseTask) {
            return out.length();
        }
        return 0;
    }

    public static int guardedValue() {
        int total;
        total = 0;
        try {
            int denom;
            denom = 0;
            total += 10 / denom;
        } catch (Exception e) {
            total += 50;
        } finally {
            total += 3;
        }
        return total;
    }

    public static int guardedFinallyOverride() {
        try {
            return 10;
        } finally {
            return 20;
        }
    }

    public static int guardedCatchFinallyOverride() {
        try {
            int denom;
            denom = 0;
            return 10 / denom;
        } catch (Exception e) {
            return 7;
        } finally {
            return 9;
        }
    }

    public static int guardedFinallyPanicOverride() {
        try {
            return 1;
        } catch (Exception e) {
            return 2;
        } finally {
            int denom;
            denom = 0;
            return 3 / denom;
        }
    }

    public static String guardedResourceOrder() {
        CloseTrace trace = new CloseTrace();
        try (ResourceToken first = new ResourceToken(trace, "A");
             ResourceToken second = new ResourceToken(trace, "B")) {
            trace.append("X");
        }
        return trace.get();
    }
}
