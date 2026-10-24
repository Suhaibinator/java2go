package transpiler

import "testing"

// Receiver evaluation precedes arguments, but the receiver null check follows
// every argument conversion. A throwing argument wins over a null receiver.
func TestCampaignStringIntrinsicInvocationOrderJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-order</artifactId><version>1</version></project>`,
		"src/main/java/order/Entry.java": `package order;
public class Entry {
    static int trace;
    static String text = "ABZ";
    static String receiver(boolean absent) {
        trace = trace * 10 + 1;
        return absent ? null : text;
    }
    static String throwingReceiver() {
        trace = trace * 10 + 1;
        throw new IllegalArgumentException("receiver-first");
    }
    static int index(boolean fail) {
        trace = trace * 10 + 2;
        if (fail) { throw new IllegalArgumentException("index-first"); }
        return 2;
    }
    static int begin() { trace = trace * 10 + 2; return 0; }
    static int end() { trace = trace * 10 + 3; return 2; }
    static String argument(boolean fail) {
        trace = trace * 10 + 2;
        if (fail) { throw new IllegalArgumentException("argument-first"); }
        return "ABZ";
    }
    static int replaceReceiver() {
        trace = trace * 10 + 2;
        text = "X";
        return 2;
    }
    static Integer boxedIndex() { trace = trace * 10 + 2; return null; }
    public static void main(String[] args) {
        trace = 0;
        try { receiver(true).charAt(index(false)); }
        catch (NullPointerException expected) { System.out.println("null=" + trace); }
        trace = 0;
        try { receiver(true).charAt(index(true)); }
        catch (IllegalArgumentException expected) { System.out.println("throw=" + trace + ":" + expected.getMessage()); }
        catch (NullPointerException wrong) { System.out.println("wrong-null=" + trace); }
        trace = 0;
        int valid = receiver(false).charAt(index(false));
        System.out.println("valid=" + valid + ":" + trace);
        trace = 0;
        try { receiver(true).substring(begin(), end()); }
        catch (NullPointerException expected) { System.out.println("range=" + trace); }
        trace = 0;
        try { receiver(true).equals(argument(false)); }
        catch (NullPointerException expected) { System.out.println("equals=" + trace); }
        trace = 0;
        try { receiver(true).concat(argument(true)); }
        catch (IllegalArgumentException expected) { System.out.println("concat=" + trace + ":" + expected.getMessage()); }
        catch (NullPointerException wrong) { System.out.println("wrong-concat=" + trace); }
        trace = 0;
        try { throwingReceiver().charAt(index(false)); }
        catch (IllegalArgumentException expected) { System.out.println("receiver=" + trace); }
        trace = 0;
        int captured = receiver(false).charAt(replaceReceiver());
        System.out.println("captured=" + captured + ":" + trace + ":" + text);
        trace = 0;
        try { receiver(true).charAt(boxedIndex()); }
        catch (NullPointerException expected) { System.out.println("unbox=" + trace); }
        trace = 0;
        try { receiver(true).length(); }
        catch (NullPointerException expected) { System.out.println("zero=" + trace); }
    }
}`,
	}, "order.Entry", "null=12\nthrow=12:index-first\nvalid=90:12\nrange=123\nequals=12\nconcat=12:argument-first\nreceiver=1\ncaptured=90:12:X\nunbox=12\nzero=1\n")
}
