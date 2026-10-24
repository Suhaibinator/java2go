package parity.analytics.fields;

public class FieldReceiverProbe implements Parser {
    private int calls;
    private boolean nullValue;

    public FieldReceiverProbe(boolean nullValue) {
        this.nullValue = nullValue;
    }

    public String parse() {
        this.calls++;
        if (this.nullValue) {
            return null;
        }
        return "p" + this.calls;
    }

    public static String run() {
        FieldReceiverProbe parser = new FieldReceiverProbe(false);
        FieldBackedCalls calls = new FieldBackedCalls(parser, new Worker());
        String result = calls.use() + "\n";
        result += calls.use() + "\n";
        result += "calls:" + parser.calls + "\n";
        try {
            new FieldBackedCalls(parser, null).use();
            result += "missing-worker-exception\n";
        } catch (NullPointerException expected) {
            result += "null-worker:" + parser.calls + "\n";
        }
        try {
            new FieldBackedCalls(null, new Worker()).use();
            result += "missing-parser-exception\n";
        } catch (NullPointerException expected) {
            result += "null-parser:" + parser.calls + "\n";
        }
        FieldBackedCalls nulls = new FieldBackedCalls(new FieldReceiverProbe(true), new Worker());
        result += "null-value:" + nulls.use() + "\n";
        return result;
    }
}
