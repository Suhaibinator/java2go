package com.acme.refs;

public class Outer {
    public class Inner {
        String value;

        public Inner(String value) {
            this.value = value;
        }

        public String value() {
            return this.value;
        }
    }

    public Inner build(String in) {
        return this.new Inner(in);
    }

    public static Mapper<String, String> mapper() {
        return RefUtil::id;
    }
}
