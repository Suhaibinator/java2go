package transpiler

import "testing"

// Preserve the complete FloatingText source from the shape regression and
// exercise its existing print method with a separate deterministic JDK driver.
func TestEnumFloatingOriginalPrintJDK21Parity(t *testing.T) {
	const original = `
public class FloatingText {
    public static void print(double d, float f) {
        System.out.println(d);
        System.out.println("double=" + d);
        System.out.println(f);
        System.out.println(String.valueOf(d));
        System.out.println(Double.toString(d));
    }
}
`
	const driver = `public class OriginalFloatingOutputDriver {
    public static void main(String[] args) {
        FloatingText.print(98.0, 37.25f);
        FloatingText.print(-0.0, -0.0f);
        FloatingText.print(Double.NaN, Float.NaN);
        FloatingText.print(Double.POSITIVE_INFINITY, Float.NEGATIVE_INFINITY);
        FloatingText.print(Double.NEGATIVE_INFINITY, Float.POSITIVE_INFINITY);
        FloatingText.print(Double.MIN_VALUE, Float.MIN_VALUE);
        FloatingText.print(0.0001, 0.0001f);
        FloatingText.print(10000000.0, 10000000.0f);
        FloatingText.print(0.0, 0.0f);
    }
}
`
	const want = "98.0\ndouble=98.0\n37.25\n98.0\n98.0\n-0.0\ndouble=-0.0\n-0.0\n-0.0\n-0.0\nNaN\ndouble=NaN\nNaN\nNaN\nNaN\nInfinity\ndouble=Infinity\n-Infinity\nInfinity\nInfinity\n-Infinity\ndouble=-Infinity\nInfinity\n-Infinity\n-Infinity\n4.9E-324\ndouble=4.9E-324\n1.4E-45\n4.9E-324\n4.9E-324\n1.0E-4\ndouble=1.0E-4\n1.0E-4\n1.0E-4\n1.0E-4\n1.0E7\ndouble=1.0E7\n1.0E7\n1.0E7\n1.0E7\n0.0\ndouble=0.0\n0.0\n0.0\n0.0\n"
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                         `<project><modelVersion>4.0.0</modelVersion><groupId>enumformat</groupId><artifactId>original-floating</artifactId><version>1</version></project>`,
		"src/main/java/FloatingText.java": original,
		"src/main/java/OriginalFloatingOutputDriver.java": driver,
	}, "OriginalFloatingOutputDriver", want)
}
