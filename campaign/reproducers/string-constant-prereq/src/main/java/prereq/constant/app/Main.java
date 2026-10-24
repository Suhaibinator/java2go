package prereq.constant.app;

import prereq.constant.api.Constants;
import prereq.constant.api.Trace;
import prereq.constant.flow.ConstantFlow;

public final class Main {
    private Main() {}

    private static Constants qualifiedConstant(int seed) {
        Trace.add("Q" + seed);
        return null;
    }

    private static prereq.constant.shadow.String qualifiedShadow(int seed) {
        Trace.add("H" + seed);
        return null;
    }

    public static void main(java.lang.String[] args) {
        int seed = Integer.parseInt(args[0]);
        final java.lang.String fieldAlias = ConstantFlow.FIELD_ALIAS;
        final int primitiveAlias = Constants.OVERFLOW;
        final char letterAlias = Constants.LETTER;
        final boolean flagAlias = Constants.FLAG;
        final float floatAlias = Constants.FRACTION;
        final java.lang.String localFolded = fieldAlias + ":" + primitiveAlias + ":"
                + letterAlias + ":" + flagAlias + ":" + floatAlias;
        boolean internedFields = Constants.TEXT == "lexical"
                && prereq.constant.shadow.String.TEXT == "shadow"
                && ConstantFlow.FIELD_ALIAS == "lexical/shadow"
                && Constants.FOLDED == "lexical:-2147483648:1:1:B:true:1.75"
                && localFolded == "lexical/shadow:-2147483648:B:true:1.75";
        System.out.println("seed=" + seed + ",folded=" + internedFields
                + ",before=" + Trace.snapshot());

        java.lang.String viaConstantQualifier = qualifiedConstant(seed).TEXT;
        java.lang.String viaShadowQualifier = qualifiedShadow(seed).TEXT;
        System.out.println("qualifiers=" + viaConstantQualifier + "," + viaShadowQualifier
                + ",trace=" + Trace.snapshot());

        final java.lang.String lateAlias;
        lateAlias = ConstantFlow.FIELD_ALIAS;
        java.lang.String runtimeFromLateAlias = lateAlias + "/" + seed;
        java.lang.String runtimeFromLateAliasAgain = lateAlias + "/" + seed;
        System.out.println("lateAlias=" + runtimeFromLateAlias.equals(runtimeFromLateAliasAgain)
                + ",same=" + (runtimeFromLateAlias == runtimeFromLateAliasAgain));

        java.lang.String first = Constants.touch();
        java.lang.String second = prereq.constant.shadow.String.touch();
        java.lang.String third = ConstantFlow.touch();
        System.out.println("touch=" + (first == Constants.TEXT) + ","
                + (second == prereq.constant.shadow.String.TEXT) + ","
                + (third == ConstantFlow.FIELD_CHAIN) + ",trace=" + Trace.snapshot());

        java.lang.String dynamicOne = Constants.RUNTIME + seed;
        java.lang.String dynamicTwo = Constants.RUNTIME + seed;
        System.out.println("arithmetic=" + Constants.OVERFLOW + "," + Constants.INT_SHIFT
                + "," + Constants.LONG_SHIFT + "," + (int) Constants.LETTER + ","
                + Constants.FLAG + "," + Constants.FRACTION);
        System.out.println("dynamic=" + dynamicOne.equals(dynamicTwo) + ",same=" +
                (dynamicOne == dynamicTwo) + ",shadow=" + ConstantFlow.shadowPayload(seed)
                + ",finalTrace=" + Trace.snapshot());
    }
}
