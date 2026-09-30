package parity.objectmodel.app;

import parity.objectmodel.model.AddExpression;
import parity.objectmodel.model.ConstantExpression;
import parity.objectmodel.model.Expression;
import parity.objectmodel.model.InputExpression;
import parity.objectmodel.model.MultiplyExpression;
import parity.objectmodel.model.RecursiveChain;
import parity.objectmodel.semantics.ConstructionChild;
import parity.objectmodel.semantics.DoubleAccumulator;
import parity.objectmodel.semantics.EvenState;
import parity.objectmodel.semantics.HiddenBase;
import parity.objectmodel.semantics.HiddenChild;
import parity.objectmodel.semantics.OddState;
import parity.objectmodel.semantics.OuterScope;
import parity.objectmodel.semantics.RecursiveAccumulator;

public final class RecursiveObjectModelApplication {
    public static void main(String[] args) {
        System.out.println("OBJECT MODEL REPORT");

        Expression input = new InputExpression();
        Expression three = new ConstantExpression(3);
        Expression one = new ConstantExpression(1);
        AddExpression left = new AddExpression(input, three);
        AddExpression right = new AddExpression(input, one);
        MultiplyExpression root = new MultiplyExpression(left, right);
        Expression interfaceView = root;
        int concreteEvaluation = root.evaluate(5);
        int interfaceEvaluation = interfaceView.evaluate(2);
        System.out.println("expression=" + root.summary(5));
        System.out.println("interface=" + interfaceView.summary(2));

        ConstructionChild construction = new ConstructionChild();
        System.out.println("construction="
                + construction.observedDuringBaseInitialization() + ":"
                + construction.readyWeight());

        RecursiveAccumulator accumulator = new DoubleAccumulator();
        DoubleAccumulator doubleAccumulator = (DoubleAccumulator) accumulator;
        int virtualTotal = accumulator.accumulate(4);
        int superTotal = doubleAccumulator.throughSuper(4);
        System.out.println("virtual=" + virtualTotal + ":" + superTotal);

        EvenState even = new EvenState();
        OddState odd = new OddState();
        boolean evenAccepted = even.accepts(40, odd);
        boolean oddAccepted = odd.accepts(41, even);
        System.out.println("mutual=" + evenAccepted + ":" + oddAccepted);

        RecursiveChain<String> chain = new RecursiveChain<>("tail", null);
        chain = new RecursiveChain<>("middle", chain);
        chain = new RecursiveChain<>("head", chain);
        int chainSize = chain.size();
        System.out.println("chain=" + chainSize + ":" + chain.tail());

        int nested = new OuterScope(3).resolve(4);
        System.out.println("nested=" + nested);

        HiddenBase hiddenBase = new HiddenChild();
        HiddenChild hiddenChild = (HiddenChild) hiddenBase;
        int baseCode = hiddenBase.baseCode();
        int childCode = hiddenChild.childCode();
        System.out.println("hidden=" + baseCode + ":" + childCode);

        int checksum = concreteEvaluation + interfaceEvaluation
                + construction.observedDuringBaseInitialization()
                + construction.readyWeight() + virtualTotal + superTotal
                + (evenAccepted ? 1 : 0) + (oddAccepted ? 1 : 0)
                + chainSize + nested + baseCode + childCode;
        System.out.println("checksum=" + checksum);
    }
}
