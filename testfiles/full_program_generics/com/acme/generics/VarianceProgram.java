package com.acme.generics;

import java.util.List;

public class VarianceProgram {
    public static int consume(List<? extends Number> source, List<? super Integer> sink) {
        if (source instanceof List) {
            return 1;
        }
        return 0;
    }
}
