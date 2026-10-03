package transpiler

import "testing"

// Exercise the complete original enum helper fixture with a separate JDK driver.
func TestEnumHelpersOriginalStateJDK21Parity(t *testing.T) {
	const original = `
package enums.helpers;
public enum State {
    ON,
    OFF;
    public String label() { return name() + ":" + ordinal(); }
}
`
	const driver = `package enums.helpers;
public class OriginalJavaHelperOracle {
    public static void main(String[] args) {
        State[] values = State.values();
        System.out.println("values=" + values.length + ":" + values[0].name() + ":" + values[1].name());
        System.out.println("mapping=" + (values[0] == State.ON) + ":" + (values[1] == State.OFF) + ":" + (State.valueOf("ON") == State.ON) + ":" + (State.valueOf("OFF") == State.OFF));
        System.out.println("names=" + State.ON.name() + ":" + State.OFF.name() + ":" + (State.ON.name() == "ON") + ":" + (State.OFF.name() == "OFF"));
        System.out.println("labels=" + State.ON.label() + ":" + State.OFF.label());
        System.out.println("ordinals=" + State.ON.ordinal() + ":" + State.OFF.ordinal());
        System.out.println("compare=" + State.ON.compareTo(State.OFF) + ":" + State.OFF.compareTo(State.ON) + ":" + State.ON.compareTo(State.ON));
        State absent = null;
        System.out.println("text=" + State.ON.toString() + ":" + State.OFF.toString() + ":" + String.valueOf(absent));
        try {
            State.valueOf((String) null);
            System.out.println("null-valueOf=unexpected");
        } catch (NullPointerException expected) {
            System.out.println("null-valueOf=" + expected.getMessage());
        }
        try {
            State.valueOf("UNKNOWN");
            System.out.println("unknown-valueOf=unexpected");
        } catch (IllegalArgumentException expected) {
            System.out.println("unknown-valueOf=" + expected.getMessage());
        }
        try {
            State.ON.compareTo(absent);
            System.out.println("null-compare=unexpected");
        } catch (NullPointerException expected) {
            System.out.println("null-compare=NullPointerException");
        }
        try {
            absent.name();
            System.out.println("null-name=unexpected");
        } catch (NullPointerException expected) {
            System.out.println("null-name=NullPointerException");
        }
    }
}
`
	const want = "values=2:ON:OFF\nmapping=true:true:true:true\nnames=ON:OFF:true:true\nlabels=ON:0:OFF:1\nordinals=0:1\ncompare=-1:1:0\ntext=ON:OFF:null\nnull-valueOf=Name is null\nunknown-valueOf=No enum constant enums.helpers.State.UNKNOWN\nnull-compare=NullPointerException\nnull-name=NullPointerException\n"
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                                `<project><modelVersion>4.0.0</modelVersion><groupId>enumhelpers</groupId><artifactId>original-state</artifactId><version>1</version></project>`,
		"src/main/java/enums/helpers/State.java": original,
		"src/main/java/enums/helpers/OriginalJavaHelperOracle.java": driver,
	}, "enums.helpers.OriginalJavaHelperOracle", want)
}
