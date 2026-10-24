package transpiler

import "testing"

// Null is a valid overload argument at compile time for the Locale overload.
// Both a null Locale and a null receiver must throw at invocation time.
func TestCampaignStringNullLocaleInvocationJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>null-locale</artifactId><version>1</version></project>`,
		"src/main/java/order/Entry.java": `package order;
public class Entry {
    static String upper(String value) {
        try { return value.toUpperCase(null); }
        catch (NullPointerException expected) { return "npe"; }
    }
    static String lower(String value) {
        try { return value.toLowerCase(null); }
        catch (NullPointerException expected) { return "npe"; }
    }
    public static void main(String[] args) {
        System.out.println("upper=" + upper("abc") + ":" + upper(null));
        System.out.println("lower=" + lower("ABC") + ":" + lower(null));
    }
}`,
	}, "order.Entry", "upper=npe:npe\nlower=npe:npe\n")
}
