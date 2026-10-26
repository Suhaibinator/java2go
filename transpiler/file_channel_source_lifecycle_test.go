package transpiler

import (
	"strings"
	"testing"
)

func TestFileChannelSourceLifecycleCanonicalCurrentExecution(t *testing.T) {
	out := renderGoFileFromJava(t, `class Lifecycle { static boolean state(java.nio.channels.FileChannel c){c.close();return c.isOpen();}}`)
	if !strings.Contains(out, "FileChannelCloseVirtualExecution(__java2goExecution") || !strings.Contains(out, "FileChannelIsOpenVirtual(") {
		t.Fatalf("current invocation lifecycle facade missing:\n%s", out)
	}
	if strings.Contains(out, "c.Close()") {
		t.Fatal("fresh Execution wrapper selected")
	}
}
func TestFileChannelSourceLifecycleInheritedSelection(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.nio.channels.FileChannel; abstract class Lifecycle extends FileChannel { static boolean state(Lifecycle c){c.close();return c.isOpen();} }`)
	if !strings.Contains(out, "FileChannelCloseVirtualExecution(__java2goExecution") || !strings.Contains(out, "FileChannelIsOpenVirtual(") || !strings.Contains(out, "Java2goFileChannelView()") {
		t.Fatalf("source inherited final lifecycle lost canonical stored view:\n%s", out)
	}
}
func TestFileChannelSourceLifecycleSourceHomonym(t *testing.T) {
	out := renderGoFileFromJava(t, `class FileChannel {void close(){} boolean isOpen(){return true;}} class Lifecycle {static boolean state(FileChannel c){c.close();return c.isOpen();}}`)
	if strings.Contains(out, "FileChannelCloseVirtualExecution(") || strings.Contains(out, "FileChannelIsOpenVirtual(") {
		t.Fatalf("source homonym became native:\n%s", out)
	}
}

func TestFileChannelSourceLifecycleInheritedResultShields(t *testing.T) {
	for name, java := range map[string]string{
		"canonical": `abstract class Lifecycle extends java.nio.channels.FileChannel{String state(){return "open="+isOpen();}}`,
		"overload":  `abstract class Lifecycle extends java.nio.channels.FileChannel{int isOpen(int x){return x;}String state(){return "open="+isOpen()+":"+isOpen(4);}}`,
		"captured":  `abstract class Lifecycle extends java.nio.channels.FileChannel{<FileChannel> String state(){return "open="+isOpen();}}`,
	} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, java)
			if !strings.Contains(out, "JavaStringValueOfBoolean(stdjava.FileChannelIsOpenVirtual(") {
				t.Fatalf("proved inherited result lost:\n%s", out)
			}
		})
	}
	for name, java := range map[string]string{
		"source":  `class FileChannel{int isOpen(){return 3;}}class Lifecycle extends FileChannel{String state(){return "open="+isOpen();}}`,
		"static":  `abstract class Lifecycle extends java.nio.channels.FileChannel{static int isOpen(int x){return x;}static String state(){return "open="+isOpen(3);}}`,
		"unknown": `abstract class Lifecycle extends UnknownChannel{String state(){return "open="+isOpen();}}`,
	} {
		t.Run(name, func(t *testing.T) {
			out := renderGoFileFromJava(t, java)
			if strings.Contains(out, "FileChannelIsOpenVirtual(") {
				t.Fatalf("unproved result admitted:\n%s", out)
			}
		})
	}
}
