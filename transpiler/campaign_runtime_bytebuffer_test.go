package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeByteBuffer(t *testing.T) {
	const source = `import java.nio.ByteBuffer;
public class CampaignRuntimeByteBuffer {
    public static String run() {
        byte[] source = {10, 20, 30, 40, 50};
        ByteBuffer buffer = ByteBuffer.wrap(source, 1, 3);
        boolean shared = buffer.array() == source;
        int initial = buffer.remaining();
        byte[] first = new byte[2];
        buffer.get(first);
        int position = buffer.position();
        String underflow = "missing";
        try { buffer.get(new byte[2]); }
        catch (java.nio.BufferUnderflowException expected) { underflow = "underflow"; }
        int afterFailure = buffer.position();
        source[3] = 99;
        byte[] last = new byte[1];
        buffer.get(last);
        buffer.position(1);
        byte[] copy = new byte[3];
        buffer.get(copy);
        return shared + ":" + buffer.hasArray() + ":" + initial + ":" + first[0] + ":" + first[1]
            + ":" + position + ":" + underflow + ":" + afterFailure + ":" + last[0]
            + ":" + buffer.remaining() + ":" + copy[2];
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeByteBuffer", source)
	t.Logf("JDK buffer oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestByteBufferOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
