package transpiler

import "testing"

func TestCampaignStringCharAtBoundsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>char-bounds</artifactId><version>1</version></project>`,
		"src/main/java/bounds/Entry.java": `package bounds;
public class Entry {
    static void probe(String label, String value, int index) {
        try { System.out.println(label + ":value=" + (int) value.charAt(index)); }
        catch (StringIndexOutOfBoundsException expected) {
            System.out.println(label + ":" + expected.getClass().getName() + ":" + expected.getMessage());
        }
        catch (IndexOutOfBoundsException wrong) { System.out.println(label + ":wrong-subtype"); }
    }
    public static void main(String[] args) {
        probe("negative", "ab", -1);
        probe("equal", "ab", 2);
        probe("greater", "ab", 8);
        probe("empty", "", 0);
        probe("bmp", "é", 1);
        probe("supplementary", "😀", 2);
        probe("minimum", "ab", Integer.MIN_VALUE);
        probe("maximum", "ab", Integer.MAX_VALUE);
        probe("high", "😀", 0);
        probe("low", "😀", 1);
        probe("valid-bmp", "é", 0);
    }
}`,
	}, "bounds.Entry", "negative:java.lang.StringIndexOutOfBoundsException:Index -1 out of bounds for length 2\nequal:java.lang.StringIndexOutOfBoundsException:Index 2 out of bounds for length 2\ngreater:java.lang.StringIndexOutOfBoundsException:Index 8 out of bounds for length 2\nempty:java.lang.StringIndexOutOfBoundsException:Index 0 out of bounds for length 0\nbmp:java.lang.StringIndexOutOfBoundsException:Index 1 out of bounds for length 1\nsupplementary:java.lang.StringIndexOutOfBoundsException:Index 2 out of bounds for length 2\nminimum:java.lang.StringIndexOutOfBoundsException:Index -2147483648 out of bounds for length 2\nmaximum:java.lang.StringIndexOutOfBoundsException:Index 2147483647 out of bounds for length 2\nhigh:value=55357\nlow:value=56832\nvalid-bmp:value=233\n")
}
