package parity.sideeffects.app;

import parity.sideeffects.core.EvaluationOrderSuite;

public final class SideEffectApplication {
    private SideEffectApplication() {
    }

    public static void main(String[] args) {
        System.out.println("=== SIDE EFFECT SEMANTICS ===");
        System.out.println("SHORT " + EvaluationOrderSuite.shortCircuit());
        System.out.println("TERNARY " + EvaluationOrderSuite.ternary());
        System.out.println("INSTANCE " + EvaluationOrderSuite.ordinaryInvocation());
        System.out.println("NULL_INSTANCE " + EvaluationOrderSuite.nullInvocation());
        System.out.println("STATIC_QUALIFIER " + EvaluationOrderSuite.staticInvocationThroughExpression());
        System.out.println("COMPOUND " + EvaluationOrderSuite.compoundAssignment());
        System.out.println("FINALLY " + EvaluationOrderSuite.finallyAfterReturnExpression());
        System.out.println("RECURSION " + EvaluationOrderSuite.recursiveArguments());
        System.out.println("=== END SIDE EFFECT SEMANTICS ===");
    }
}
