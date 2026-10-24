package transpiler

import "testing"

func TestCIFileCanonicalStringAndResultBoundaries(t *testing.T) {
	const source = `import java.io.File;
import java.io.IOException;
public class FileStringBoundary {
 public static String run() throws IOException {
  String path = new String(new char[]{'a', '/', 'b', (char)0xd800});
  File named = new File(path);
  String kept = named.getPath();
  String name = named.getName();
  File temp = File.createTempFile("java2goBoundary", null);
  boolean exists = temp.exists(), regular = temp.isFile(), directory = temp.isDirectory();
  long length = temp.length();
  boolean recreated = temp.createNewFile();
  boolean removed = temp.delete(), missing = temp.exists();
  return (kept == path) + ":" + (int)kept.charAt(3) + ":" + name.length() + ":" + (int)name.charAt(1)
   + ":" + temp.getName().endsWith(".tmp") + ":" + exists + ":" + regular + ":" + directory + ":" + length + ":" + recreated + ":" + removed + ":" + missing;
 }
}`
	verifyCanonicalStringStreamOracle(t, "FileStringBoundary", source)
}

func TestCINioReadStringCanonicalBoundary(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.io.File;
import java.io.IOException;
public class NioReadStringBoundary {
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goReadBoundary", ".txt");
  Path path = temp.toPath();
  Files.writeString(path, "A😀\0Z");
  String first = Files.readString(path), second = Files.readString(path);
  Files.delete(path);
  return first.length() + ":" + (int)first.charAt(1) + ":" + (int)first.charAt(2) + ":" + (int)first.charAt(3)
   + ":" + first.equals(second) + ":" + (first == second);
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioReadStringBoundary", source)
}

func TestCIFileCanonicalOwnerPreservesSourceShadow(t *testing.T) {
	const source = `public class FileOwnerBoundary {
 static class File {
  String value; File(String text) { value = text; }
  String getName() { return value; }
  boolean exists() { return false; }
 }
 public static String run() {
  File source = new File("source");
  java.io.File platform = new java.io.File("platform.txt");
  return source.getName() + ":" + source.exists() + ":" + platform.getName();
 }
}`
	verifyCanonicalStringStreamOracle(t, "FileOwnerBoundary", source)
}

func TestCINioReadStringMalformedUTF8(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.io.File;
import java.io.IOException;
public class NioMalformedBoundary {
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goMalformedBoundary", ".txt");
  Path path = temp.toPath();
  byte[][] inputs = new byte[][] {
   {(byte)0xc0, (byte)0x80}, {(byte)0xed, (byte)0xa0, (byte)0x80},
   {(byte)0xe0, (byte)0x80, (byte)0x80}, {(byte)0xf0, (byte)0x90, (byte)0x80},
   {(byte)0xe2, (byte)0x82, 65}, {(byte)0xf4, (byte)0x90, (byte)0x80, (byte)0x80},
   {(byte)0xf0, (byte)0x90, 65}, {(byte)0xf0, (byte)0x90, (byte)0x80, 65},
   {(byte)0x80}, {(byte)0xed, (byte)0xa0}, {(byte)0xe2}, {(byte)0xf0}
  };
  String result = "";
  for (byte[] input : inputs) {
   Files.write(path, input);
   try { Files.readString(path); result += "accepted;"; }
   catch (IOException failure) { result += failure.getClass().getName() + ":" + failure.getMessage() + ";"; }
  }
  Files.delete(path);
  return result;
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioMalformedBoundary", source)
}

func TestCINioWriteStringSourceExecutionAndFailures(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.io.File;
import java.io.IOException;
public class NioWriteStringSource {
 static class Text implements CharSequence {
  String payload = "A😀\0Z"; int calls; boolean same, held;
  Thread caller = Thread.currentThread(); Object lock = new Object(); RuntimeException marker;
  public int length() { return payload.length(); }
  public char charAt(int index) { return payload.charAt(index); }
  public CharSequence subSequence(int start, int end) { return payload.substring(start, end); }
  public String toString() { calls++; same = Thread.currentThread() == caller; held = Thread.holdsLock(lock); if (marker != null) throw marker; return payload; }
 }
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goWriteSource", ".txt"); Path path = temp.toPath();
  Text text = new Text(); String written; boolean nullResult = false, sameFailure = false;
  synchronized (text.lock) {
   Files.writeString(path, text); written = Files.readString(path);
   text.payload = null;
   try { Files.writeString(path, text); } catch (NullPointerException expected) { nullResult = true; }
   text.marker = new IllegalStateException("marker");
   try { Files.writeString(path, text); } catch (RuntimeException failure) { sameFailure = failure == text.marker; }
  }
  String retained = Files.readString(path); Files.delete(path);
  return written.length() + ":" + (int)written.charAt(1) + ":" + (int)written.charAt(2) + ":" + (int)written.charAt(3)
   + ":" + written.equals(retained) + ":" + text.calls + ":" + text.same + ":" + text.held + ":" + nullResult + ":" + sameFailure;
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioWriteStringSource", source)
}

func TestCINioReadEmptyStringAndNullBoundary(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.io.File;
import java.io.IOException;
public class NioReadEmptyBoundary {
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goEmptyRead", ".txt"); Path path = temp.toPath();
  String first = Files.readString(path), second = Files.readString(path); Files.delete(path);
  boolean failed = false;
  try { Files.readString((Path)null); } catch (NullPointerException expected) { failed = true; }
  return first.length() + ":" + (first == second) + ":" + (first == "") + ":" + failed;
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioReadEmptyBoundary", source)
}

func TestCINioWriteStringStrictEncoding(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.charset.StandardCharsets;
import java.nio.charset.Charset;
import java.io.File;
import java.io.IOException;
public class NioWriteStrictBoundary {
 static String check(Path path, String text, Charset charset) throws IOException {
  Files.writeString(path, "before");
  String result;
  try { Files.writeString(path, text, charset); result = "accepted"; }
  catch (IOException failure) { result = failure.getClass().getName() + ":" + failure.getMessage(); }
  return result + ":" + Files.readString(path).equals("before") + ";";
 }
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goStrictWrite", ".txt"); Path path = temp.toPath();
  String lone = new String(new char[]{'A', (char)0xd800}), pair = "😀";
  String result = check(path, lone, StandardCharsets.UTF_8)
   + check(path, lone, StandardCharsets.ISO_8859_1) + check(path, pair, StandardCharsets.ISO_8859_1)
   + check(path, lone, StandardCharsets.US_ASCII) + check(path, pair, StandardCharsets.US_ASCII)
   + check(path, lone, StandardCharsets.UTF_16BE);
  Files.delete(path); return result;
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioWriteStrictBoundary", source)
}

func TestCIFilePrimitiveResultBoxing(t *testing.T) {
	const source = `import java.io.File;
import java.io.IOException;
public class FilePrimitiveBoundary {
 static String boxed(Object value) { return value.getClass().getName(); }
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goBoxedFile", ".txt");
  File dir = new File(temp.getPath() + ".dir");
  String result = boxed(temp.exists()) + ":" + boxed(temp.isDirectory()) + ":" + boxed(temp.isFile()) + ":" + boxed(temp.length())
   + ":" + boxed(temp.createNewFile()) + ":" + boxed(dir.mkdir()) + ":" + boxed(dir.mkdirs()) + ":" + boxed(temp.delete());
  dir.delete(); return result;
 }
}`
	verifyCanonicalStringStreamOracle(t, "FilePrimitiveBoundary", source)
}

func TestCIFileNormalizationAndInvalidPathBoundary(t *testing.T) {
	const source = `import java.io.File;
import java.nio.file.InvalidPathException;
public class FileNormalizationBoundary {
 public static String run() {
  String original = "a//b///"; File file = new File(original); boolean nullFailed = false, nulFailed = false, surrogateFailed = false;
  try { new File((String)null); } catch (NullPointerException expected) { nullFailed = true; }
  File nul = new File("a\0b"), surrogate = new File(new String(new char[]{'a', (char)0xd800}));
  try { nul.toPath(); } catch (InvalidPathException expected) { nulFailed = true; }
  try { surrogate.toPath(); } catch (InvalidPathException expected) { surrogateFailed = true; }
  return file.getPath() + ":" + file.getName() + ":" + (file.getPath() == original) + ":" + file.getAbsolutePath().endsWith("/a/b")
   + ":" + new File("").getName().length() + ":" + new File("/").getName().length() + ":" + nullFailed + ":" + nul.exists() + ":" + nulFailed + ":" + surrogateFailed;
 }
}`
	verifyCanonicalStringStreamOracle(t, "FileNormalizationBoundary", source)
}

func TestCINioPathPrimitiveAndStringArgumentBoundaries(t *testing.T) {
	const source = `import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.Files;
public class NioPathPrimitiveBoundary {
 public static String run() {
  Path path = Paths.get("alpha", "beta", "gamma.txt");
  Object count = path.getNameCount(), starts = path.startsWith("alpha/beta"), ends = path.endsWith("gamma.txt"), exists = Files.exists(path);
  return path.getNameCount() + ":" + path.startsWith("alpha/beta") + ":" + path.startsWith("alph") + ":" + path.endsWith("gamma.txt")
   + ":" + Files.exists(path) + ":" + count.getClass().getName() + ":" + starts.getClass().getName() + ":" + ends.getClass().getName() + ":" + exists.getClass().getName();
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioPathPrimitiveBoundary", source)
}

func TestCIPrintWriterCanonicalUTF16Boundary(t *testing.T) {
	const source = `import java.io.File;
import java.io.PrintWriter;
import java.nio.file.Files;
import java.nio.file.Path;
import java.io.IOException;
public class PrintWriterBoundary {
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goPrintBoundary", ".txt"); Path path = temp.toPath();
  PrintWriter writer = new PrintWriter(temp);
  writer.print(new String(new char[]{'A', (char)0xd83d, (char)0xde00, 0, (char)0xd800}));
  writer.println((String)null); writer.close();
  String read = Files.readString(path); Files.delete(path);
  return read.length() + ":" + (int)read.charAt(0) + ":" + (int)read.charAt(1) + ":" + (int)read.charAt(2) + ":" + (int)read.charAt(3) + ":" + (int)read.charAt(4) + ":" + read.endsWith("null\n");
 }
}`
	verifyCanonicalStringStreamOracle(t, "PrintWriterBoundary", source)
}

func TestCIFileNameAndPathReferenceIdentity(t *testing.T) {
	const source = `import java.io.File;
import java.io.IOException;
public class FileNameIdentity {
 public static String run() throws IOException {
  String plain = new String(new char[]{'a'}); File file = new File(plain);
  File absolute = new File("/a"), root = new File("/");
  File temp = File.createTempFile("java2goPathIdentity", ".txt");
  boolean stable = temp.getPath() == temp.getPath(); temp.delete();
  return (file.getPath() == plain) + ":" + (file.getName() == plain) + ":" + (absolute.getAbsolutePath() == absolute.getPath())
   + ":" + (root.getName() == "") + ":" + stable;
 }
}`
	verifyCanonicalStringStreamOracle(t, "FileNameIdentity", source)
}

func TestCINioPathCanonicalOwnerAndInvalidStrings(t *testing.T) {
	const source = `import java.nio.file.InvalidPathException;
public class NioPathOwnerBoundary {
 static class Path {
  String startsWith(String text) { return "source-start:" + text; }
  String endsWith(String text) { return "source-end:" + text; }
 }
 static class Files { static String exists(Path path) { return "source-exists"; } }
 public static String run() {
  Path source = new Path(); java.nio.file.Path platform = java.nio.file.Paths.get("alpha", "beta");
  boolean nil = false, nul = false, surrogate = false;
  try { platform.startsWith((String)null); } catch (NullPointerException expected) { nil = true; }
  try { platform.endsWith("a\0b"); } catch (InvalidPathException expected) { nul = true; }
  try { platform.startsWith(new String(new char[]{(char)0xd800})); } catch (InvalidPathException expected) { surrogate = true; }
  return source.startsWith("a") + ":" + source.endsWith("b") + ":" + Files.exists(source) + ":" + platform.startsWith(java.nio.file.Paths.get("alpha"))
   + ":" + java.nio.file.Files.exists(platform) + ":" + nil + ":" + nul + ":" + surrogate;
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioPathOwnerBoundary", source)
}

func TestCINioCanonicalLineCollectionsAndWrite(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.charset.StandardCharsets;
import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
public class NioLineBoundary {
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goLineBoundary", ".txt"); Path path = temp.toPath();
  Files.write(path, "a\r\nb\rc\n\n😀\0Z\n".getBytes(StandardCharsets.UTF_8));
  List<String> lines = Files.readAllLines(path); String lengths = "";
  for (String line : lines) { lengths += line.length() + ","; }
  long streamed = Files.lines(path).count();
  List<String> output = new ArrayList<String>(); output.add("A😀"); output.add("\0Z"); output.add(null);
  boolean nullElement = false;
  try { Files.write(path, output); } catch (NullPointerException expected) { nullElement = true; }
  String text = Files.readString(path); long bytes = Files.size(path); Files.delete(path);
  return lines.size() + ":" + lengths + ":" + streamed + ":" + nullElement + ":" + text.length() + ":" + bytes + ":" + text.endsWith("null\n");
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioLineBoundary", source)
}

func TestCINioReadAllLinesMalformedUTF8(t *testing.T) {
	const source = `import java.nio.file.Files;
import java.nio.file.Path;
import java.io.File;
import java.io.IOException;
public class NioReadLinesMalformed {
 static String check(Path path, byte[] bytes) throws IOException {
  Files.write(path, bytes);
  try { Files.readAllLines(path); return "accepted;"; }
  catch (IOException failure) { return failure.getClass().getName() + ":" + failure.getMessage() + ";"; }
 }
 public static String run() throws IOException {
  File temp = File.createTempFile("java2goReadLinesMalformed", ".txt"); Path path = temp.toPath();
  String result = check(path, new byte[]{(byte)0xe0, (byte)0x80, (byte)0x80})
   + check(path, new byte[]{(byte)0xed, (byte)0xa0, (byte)0x80})
   + check(path, new byte[]{(byte)0xf0, (byte)0x90, (byte)0x80})
   + check(path, new byte[]{(byte)0xe2, (byte)0x82, 65});
  Files.delete(path); return result;
 }
}`
	verifyCanonicalStringStreamOracle(t, "NioReadLinesMalformed", source)
}

func TestCIStringWriterCanonicalUTF16Snapshots(t *testing.T) {
	const source = `import java.io.StringWriter;
import java.io.PrintWriter;
public class StringWriterBoundary {
 public static String run() {
  StringWriter memory = new StringWriter();
  memory.write(new String(new char[]{'A', 0, (char)0xd800}));
  PrintWriter outer = new PrintWriter(new PrintWriter(memory));
  outer.print(new String(new char[]{(char)0xdc00})); outer.println((String)null); outer.flush();
  String first = memory.toString(), second = memory.toString();
  memory.write("Z"); String third = memory.toString(); outer.close();
  return first.length() + ":" + (int)first.charAt(1) + ":" + (int)first.charAt(2) + ":" + (int)first.charAt(3)
   + ":" + first.endsWith("null\n") + ":" + first.equals(second) + ":" + (first == second) + ":" + third.length() + ":" + first.length();
 }
}`
	verifyCanonicalStringStreamOracle(t, "StringWriterBoundary", source)
}

func TestCIByteArrayOutputStreamCanonicalDecode(t *testing.T) {
	const source = `import java.io.ByteArrayOutputStream;
public class ByteStreamStringBoundary {
 public static String run() throws Exception {
  ByteArrayOutputStream bytes = new ByteArrayOutputStream();
  bytes.write(new byte[]{65, 0, (byte)0xed, (byte)0xa0, (byte)0x80, (byte)0xf0, (byte)0x9f, (byte)0x98, (byte)0x80});
  String first = bytes.toString(), second = bytes.toString();
  Object size = bytes.size(); bytes.reset();
  return first.length() + ":" + (int)first.charAt(1) + ":" + (int)first.charAt(2) + ":" + (int)first.charAt(3) + ":" + (int)first.charAt(4)
   + ":" + first.equals(second) + ":" + (first == second) + ":" + bytes.toString().length() + ":" + size.getClass().getName();
 }
}`
	verifyCanonicalStringStreamOracle(t, "ByteStreamStringBoundary", source)
}
