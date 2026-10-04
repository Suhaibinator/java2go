package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignDigestIOJVMParity(t *testing.T) {
	const source = `import java.io.*;
import java.nio.*;
import java.nio.channels.*;
import java.nio.file.*;
public class CampaignDigestIO {
 private static int consume(InputStream input) throws IOException {
  byte[] bytes = new byte[5];
  int count = input.read(bytes, 1, 2);
  return count * 100 + bytes[1] * 10 + bytes[2];
 }
 public static String run() throws Exception {
  File file = File.createTempFile("campaign-io", ".bin");
  Path path = file.toPath();
  Files.write(path, new byte[] {1, 2, 3, 4, 5});
  int direct;
  try (InputStream stream = Files.newInputStream(path, StandardOpenOption.READ)) { direct = consume(stream); }
  int buffered;
  try (BufferedInputStream stream = new BufferedInputStream(Files.newInputStream(path), 2)) { buffered = consume(stream); }
  int memory;
  try (InputStream stream = new ByteArrayInputStream(new byte[] {6, 7})) { memory = consume(stream); }
  int channelCount;
  int first;
  int last;
  long position;
  int eof;
  try (RandomAccessFile random = new RandomAccessFile(file, "r")) {
   random.seek(1);
   FileChannel channel = random.getChannel();
   ByteBuffer buffer = ByteBuffer.allocate(3);
   channelCount = channel.read(buffer);
   buffer.flip();
   byte[] bytes = new byte[3];
   buffer.get(bytes);
   first = bytes[0];
   last = bytes[2];
   buffer.clear();
   channel.read(buffer);
   buffer.clear();
   eof = channel.read(buffer);
   position = random.getFilePointer();
  }
  Files.delete(path);
  return direct + ":" + buffered + ":" + memory + ":" + channelCount + ":" + first + ":" + last + ":" + position + ":" + eof;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignDigestIO", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestDigestIO(t *testing.T) { if got:=Run(); got!=%q { t.Fatalf("JVM %%q != generated Go %%q",%q,got) } }
 `, want, want))
}
