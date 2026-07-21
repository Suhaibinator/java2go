package parity.constructordelegation;

class ConstructorTrace {
    int initializations;
    String trace = initialize();

    ConstructorTrace() {
        this("target");
        trace += ":delegating";
    }

    ConstructorTrace(String marker) {
        trace += ":" + marker;
    }

    String initialize() {
        initializations++;
        return "field";
    }
}

public final class ConstructorDelegationApplication {
    public static void main(String[] args) {
        ConstructorTrace value = new ConstructorTrace();
        System.out.println(value.trace + ":" + value.initializations);
    }
}
