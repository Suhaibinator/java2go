package probe.app;

import probe.nometa.Lookalike;

public final class Main {
    private Main() {}

    private static String counters(Lookalike value) {
        return value.messageCalls + ":" + value.nameCalls + ":"
                + value.errorCalls + ":" + value.hashCalls;
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        int aliasReads = 1 + seed % 3;
        Lookalike value = new Lookalike();
        Object alias = value;

        // A source-qualified string literal avoids obtaining Class metadata.
        String expected = "probe.nometa.Lookalike@" + Integer.toHexString(value.hashCode());
        System.out.println("seed=" + seed + ",expected=" + expected
                + ",initial=" + counters(value));

        String fromValueOf = String.valueOf(value);
        System.out.println("valueOf=" + fromValueOf + ",matches="
                + fromValueOf.equals(expected) + ",calls=" + counters(value));

        String direct = value.toString();
        System.out.println("direct=" + direct + ",matches="
                + direct.equals(expected) + ",calls=" + counters(value));

        for (int i = 0; i < aliasReads; i++) {
            String fromAlias = String.valueOf(alias);
            System.out.println("aliasValueOf#" + (i + 1) + '=' + fromAlias
                    + ",matches=" + fromAlias.equals(expected)
                    + ",calls=" + counters(value));
        }

        String aliasDirect = alias.toString();
        System.out.println("aliasDirect=" + aliasDirect + ",matches="
                + aliasDirect.equals(expected) + ",calls=" + counters(value));
    }
}
