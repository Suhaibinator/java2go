package com.acme.domain;

import com.acme.common.Mapper;

public class ParseTask extends Task {
    public ParseTask(String id) {
        super(id);
    }

    public int run(String input) {
        Mapper<String, String> trim = v -> v;
        String normalized = trim.map(input);
        if (normalized instanceof String) {
            return normalized.length();
        }
        return 0;
    }
}
