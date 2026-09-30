package com.acme.refs;

public interface Mapper<T, R> {
    R map(T value);
}
