package transpiler

import (
	"strings"
	"testing"
)

func TestNIOHeapCompactChainedLowering(t *testing.T) {
	got := renderGoFileFromJava(t, `import java.nio.ByteBuffer; public class CompactChain { public static int run(){ByteBuffer b=ByteBuffer.allocate(8);return b.position(3).compact().flip().remaining();}}`)
	if !strings.Contains(got, ".Compact()") || !strings.Contains(got, ".Flip()") || !strings.Contains(got, ".Remaining()") {
		t.Fatalf("heap compact chain lost nominal result type: %s", got)
	}
}
