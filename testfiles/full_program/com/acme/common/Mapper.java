package com.acme.common;

public interface Mapper<T, R> {
    R map(T value);
}
