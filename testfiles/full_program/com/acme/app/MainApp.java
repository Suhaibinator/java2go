package com.acme.app;

import com.acme.common.Logger;
import com.acme.common.Mapper;
import com.acme.common.Mode;
import com.acme.domain.ParseTask;
import com.acme.domain.Task;

public class MainApp {
    public static void main(String[] args) {
        Task task = new ParseTask("alpha");
        Mapper<String, String> identity = v -> v;
        int count = Pipeline.execute(task, identity);

        Mode mode = Mode.valueOf("FAST");
        for (Mode each : Mode.values()) {
            Logger.log(each.name());
        }

        Logger.log(task.name());
        Logger.log("" + count + mode.ordinal());
        Logger.log("" + Pipeline.guardedValue());
        Logger.log("" + Pipeline.guardedFinallyOverride());
        Logger.log("" + Pipeline.guardedCatchFinallyOverride());
        try {
            Logger.log("" + Pipeline.guardedFinallyPanicOverride());
        } catch (Exception e) {
            Logger.log("PANIC_FINALLY");
        }
        Logger.log(Pipeline.guardedResourceOrder());
    }
}
