package transpiler

import (
	"strings"
	"testing"
)

// Replay every declaration and the original main from the void-lambda fixture.
// javac requires its two public top-level types in separate named files.
func TestComplexVoidLambdaOriginalJVMParity(t *testing.T) {
	const original = `
package complex.handler;

public interface Handler<T> { void handle(T value); }

public class Logger {
    public static void writeAll(Handler<String> handler) {
        handler.handle("first");
        handler.handle("second");
    }

    public static void main(String[] args) {
        writeAll(v -> { System.out.println(v); });
    }
}
`
	const packageHeader = "\npackage complex.handler;\n\n"
	loggerStart := strings.Index(original, "public class Logger {")
	if loggerStart < len(packageHeader) {
		t.Fatal("original Logger declaration missing")
	}
	files := map[string]string{
		"pom.xml": `<project><groupId>complex</groupId><artifactId>void-lambda-original</artifactId><version>1</version></project>`,
		"src/main/java/complex/handler/Handler.java": original[:loggerStart],
		"src/main/java/complex/handler/Logger.java":  packageHeader + original[loggerStart:],
	}
	runCampaignCompilerStrictProjectOracle(t, files, "complex.handler.Logger", "first\nsecond\n")
}
