package transpiler

import (
	"strings"
	"testing"
)

// The standalone JDK21 ResourceMatrix oracle covers execution, nulls, initializer
// effects, identity and exact suppression ordering. These tests pin admission.
func TestFileChannelResourceCanonicalProjectionAndClose(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.nio.channels.FileChannel;
 abstract class ResourceProbe extends FileChannel {
  static void probe(ResourceProbe value) throws Exception {
   try(FileChannel canonical=value){}
   try(ResourceProbe source=value){}
   try(var inferred=value){}
   try(FileChannel empty=null){}
   try(ResourceProbe emptySource=null){}
  }
 }`)
	if !strings.Contains(out, "var canonical *stdjava.FileChannel") || !strings.Contains(out, "__java2goChannelSource := value") {
		t.Fatalf("resource discarded actual canonical declaration/source projection:\n%s", out)
	}
	if got := strings.Count(out, "FileChannelCloseVirtualExecution(__java2goExecution,"); got != 5 {
		t.Fatalf("canonical/source/null/var resource close facades=%d want5:\n%s", got, out)
	}
	if strings.Contains(out, "canonical.Close()") || strings.Contains(out, "source.Close()") || strings.Contains(out, "inferred.Close()") {
		t.Fatalf("canonical inherited resource used public Close ABI:\n%s", out)
	}
}
func TestFileChannelResourceDeclarationShields(t *testing.T) {
	for name, java := range map[string]string{
		"homonym":                `class FileChannel implements AutoCloseable{public void close(){}}class Keep{void run(FileChannel v)throws Exception{try(FileChannel r=v){}}}`,
		"binder":                 `class Keep{<FileChannel extends AutoCloseable>void run(FileChannel v)throws Exception{try(FileChannel r=v){}}}`,
		"captured":               `class Keep<FileChannel extends AutoCloseable>{class Inner{void run(FileChannel v)throws Exception{try(FileChannel r=v){}}}}`,
		"foreign":                `import foreign.FileChannel;class Keep{void run(FileChannel v)throws Exception{try(FileChannel r=v){}}}`,
		"unknown":                `class Keep{void run(FileChannel v)throws Exception{try(FileChannel r=v){}}}`,
		"package-root":           `class java{static class nio{static class channels{static class FileChannel implements AutoCloseable{public void close(){}}}}}class Keep{void run(java.nio.channels.FileChannel v)throws Exception{try(java.nio.channels.FileChannel r=v){}}}`,
		"ordinary-reader":        `import java.io.StringReader;class Keep{void run(StringReader v)throws Exception{try(StringReader r=v){}}}`,
		"ordinary-autocloseable": `class Keep{void run(AutoCloseable v)throws Exception{try(AutoCloseable r=v){}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, java)
			if strings.Contains(out, "FileChannelCloseVirtualExecution(") || strings.Contains(out, "__java2goChannelSource") {
				t.Fatalf("unproved or other resource became canonical FileChannel:\n%s", out)
			}
		})
	}
}
func TestFileChannelResourceExplicitSourceCloseRetainsSelection(t *testing.T) {
	// Deliberately invalid Java override remains a declaration-resolution guard;
	// valid JDK source may only overload this inherited final close method.
	out := renderGoFileFromJava(t, `abstract class OwnClose extends java.nio.channels.FileChannel{public void close(){}static void run(OwnClose v){try(OwnClose r=v){}}}`)
	if strings.Contains(out, "FileChannelCloseVirtualExecution(") {
		t.Fatalf("own source close declaration replaced by canonical route:\n%s", out)
	}
}
