package transpiler

import "testing"

// Supplemental owner guard; the original parser regression remains unchanged.
// A source Integer import must select its declared parseInt, while an explicit
// java.lang.Integer qualifier still selects the canonical builtin.
func TestCampaignJavaIntegerParseIntOwnerGuardJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>integer-owner-guard</artifactId><version>1</version></project>`,
		"src/main/java/owned/Integer.java": `package owned;
public class Integer {
    public static int parseInt(String ignored){return 7;}
    public static int parseInt(String ignored,int radix){return radix+7;}
}`,
		"src/main/java/ownercalls/Entry.java": `package ownercalls;
import owned.Integer;
import static owned.Integer.parseInt;
public class Entry {
    public static void main(String[] args){
        System.out.println("source="+parseInt("not numeric")+":"+parseInt("ignored",16)+":"+Integer.parseInt("source"));
        System.out.println("builtin="+java.lang.Integer.parseInt("42")+":"+java.lang.Integer.parseInt("ff",16));
    }
}`,
	}, "ownercalls.Entry", "source=7:23:7\nbuiltin=42:255\n")
}
