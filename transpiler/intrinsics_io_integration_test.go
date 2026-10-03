package transpiler

import (
	"strings"
	"testing"
)

func TestIO_FileAndWriterDispatch(t *testing.T) {
	src := `
import java.io.File;
import java.io.PrintWriter;
public class IOProgram {
    public static void run(String path) throws Exception {
        PrintWriter w = new PrintWriter(path);
        w.println("x");
        w.print("y");
        w.close();
        File f = new File(path);
        boolean e = f.exists();
        String n = f.getName();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NewPrintWriter(path)")
	assertContains(t, out, "w.Println(stdjava.JavaStringLiteralUTF16([]uint16{120}))")
	assertContains(t, out, "w.Print(stdjava.JavaStringLiteralUTF16([]uint16{121}))")
	assertContains(t, out, "w.Close()")
	assertContains(t, out, "stdjava.NewJavaFileReference(path)")
	assertContains(t, out, "f.Exists()")
	assertContains(t, out, "f.GetNameReference()")
}

func TestIO_BufferedReaderUnwrapsFileReader(t *testing.T) {
	src := `
import java.io.BufferedReader;
import java.io.FileReader;
public class ReaderProgram {
    public static void run(String path) throws Exception {
        BufferedReader r = new BufferedReader(new FileReader(path));
        String line = r.readLine();
        r.close();
    }
}
`
	out := renderGoFileFromJava(t, src)
	// The nested new FileReader(path) is unwrapped to its path.
	assertContains(t, out, "stdjava.NewBufferedReaderReference(path)")
	assertContains(t, out, "r.ReadLineReference()")
	assertContains(t, out, "r.Close()")
}

func TestIO_ScannerStdinAndFile(t *testing.T) {
	src := `
import java.util.Scanner;
import java.io.File;
public class ScannerProgram {
    public static void stdin() {
        Scanner sc = new Scanner(System.in);
        int n = sc.nextInt();
        sc.close();
    }
    public static void file(String path) throws Exception {
        Scanner sc = new Scanner(new File(path));
        String tok = sc.next();
        sc.close();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NewScannerStdin()")
	// Keep the canonical File constructor so path validation and normalization
	// happen before the Scanner encodes its native path.
	assertContains(t, out, "stdjava.NewScannerFile(stdjava.NewJavaFileReference(path))")
	if strings.Count(out, "stdjava.NewJavaFileReference(path)") != 1 {
		t.Fatalf("Scanner's File argument must be constructed exactly once:\n%s", out)
	}
	assertContains(t, out, "sc.NextInt()")
	assertContains(t, out, "sc.Next()")
}

func TestIO_FileWriterAppendAndBufferedWriter(t *testing.T) {
	src := `
import java.io.BufferedWriter;
import java.io.FileWriter;
public class WriterProgram {
    public static void run(String path) throws Exception {
        FileWriter appender = new FileWriter(path, true);
        appender.write("x");
        appender.close();
        BufferedWriter buffered = new BufferedWriter(new FileWriter(path));
        buffered.write("y");
        buffered.newLine();
        buffered.close();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NewPrintWriterAppend(path, true)")
	// The nested new FileWriter(path) is unwrapped to its path.
	assertContains(t, out, "stdjava.NewBufferedWriter(path)")
	assertContains(t, out, "buffered.WriteString(stdjava.JavaStringLiteralUTF16([]uint16{121}))")
	assertContains(t, out, "buffered.NewLine()")
}

func TestIO_InMemoryWritersAndStreams(t *testing.T) {
	src := `
import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.io.PrintWriter;
import java.io.StringWriter;
public class MemoryProgram {
    public static String run() {
        StringWriter memory = new StringWriter();
        PrintWriter writer = new PrintWriter(memory);
        writer.println("line");
        writer.flush();
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        PrintStream stream = new PrintStream(bytes);
        stream.print("x");
        stream.close();
        int size = bytes.size();
        return memory.toString() + bytes.toString() + size;
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NewStringWriter()")
	assertContains(t, out, "stdjava.NewPrintWriter(memory)")
	assertContains(t, out, "writer.Println(stdjava.JavaStringLiteralUTF16([]uint16{108, 105, 110, 101}))")
	assertContains(t, out, "stdjava.NewByteArrayOutputStream()")
	assertContains(t, out, "stdjava.NewPrintStream(bytes)")
	assertContains(t, out, "stream.Print(stdjava.JavaStringLiteralUTF16([]uint16{120}))")
	assertContains(t, out, "stream.Close()")
	assertContains(t, out, "bytes.Size()")
	assertContains(t, out, "memory.StringReference()")
	assertContains(t, out, "bytes.StringReference()")
}

func TestIO_ByteStreamsAndConsoleReader(t *testing.T) {
	src := `
import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.InputStreamReader;
public class StreamProgram {
    public static void run(String path) throws Exception {
        FileOutputStream out = new FileOutputStream(new File(path), true);
        out.write(65);
        out.close();
        FileInputStream in = new FileInputStream(path);
        int first = in.read();
        in.close();
        BufferedReader console = new BufferedReader(new InputStreamReader(System.in));
        String line = console.readLine();
    }
}
`
	out := renderGoFileFromJava(t, src)
	// Keep the canonical File constructor so path validation and normalization
	// happen before the output stream encodes its native path.
	assertContains(t, out, "stdjava.NewFileOutputStreamAppend(stdjava.NewJavaFileReference(path), true)")
	if strings.Count(out, "stdjava.NewJavaFileReference(path)") != 1 {
		t.Fatalf("output stream's File argument must be constructed exactly once:\n%s", out)
	}
	assertContains(t, out, "out.WriteBytes(")
	assertContains(t, out, "stdjava.NewFileInputStream(path)")
	assertContains(t, out, "in.ReadByteValue()")
	assertContains(t, out, "stdjava.NewBufferedReaderReference(stdjava.NewInputStreamReaderStdin())")
	assertContains(t, out, "console.ReadLineReference()")
}

func TestIO_BufferedReaderLinesIsAStream(t *testing.T) {
	src := `
import java.io.BufferedReader;
import java.io.FileReader;
public class LinesProgram {
    public static long run(String path) throws Exception {
        BufferedReader reader = new BufferedReader(new FileReader(path));
        return reader.lines().count();
    }
}
`
	out := renderGoFileFromJava(t, src)
	// lines() must be typed as a Stream so the chained count() resolves.
	assertContains(t, out, "reader.Lines().Count()")
}

func TestNio_PathsAndPathMethods(t *testing.T) {
	src := `
import java.nio.file.Path;
import java.nio.file.Paths;
public class PathProgram {
    public static void run() {
        Path p = Paths.get("alpha", "beta");
        Path q = Path.of("gamma");
        String name = p.getFileName().toString();
        Path parent = p.getParent();
        Path child = parent.resolve("delta");
        int count = p.getNameCount();
        boolean starts = p.startsWith("alpha");
        boolean ends = p.endsWith("beta");
        String norm = p.normalize().toString();
        String abs = p.toAbsolutePath().toString();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.PathsGetReference(stdjava.JavaStringLiteralUTF16([]uint16{97, 108, 112, 104, 97}), stdjava.JavaStringLiteralUTF16([]uint16{98, 101, 116, 97}))")
	assertContains(t, out, "stdjava.PathsGetReference(stdjava.JavaStringLiteralUTF16([]uint16{103, 97, 109, 109, 97}))")
	assertContains(t, out, "stdjava.PathToStringReference(p.GetFileName())")
	assertContains(t, out, "p.GetParent()")
	assertContains(t, out, "stdjava.PathResolveStringReference(parent, stdjava.JavaStringLiteralUTF16([]uint16{100, 101, 108, 116, 97}))")
	assertContains(t, out, "p.GetNameCount()")
	assertContains(t, out, "p.StartsWithReference(stdjava.JavaStringLiteralUTF16([]uint16{97, 108, 112, 104, 97}))")
	assertContains(t, out, "p.EndsWithReference(stdjava.JavaStringLiteralUTF16([]uint16{98, 101, 116, 97}))")
	assertContains(t, out, "stdjava.PathToStringReference(p.Normalize())")
	assertContains(t, out, "stdjava.PathToStringReference(p.ToAbsolutePath())")
}

func TestNio_FilesStatics(t *testing.T) {
	src := `
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
public class FilesProgram {
    public static void run(Path p, Path q) throws IOException {
        Files.writeString(p, "text");
        String content = Files.readString(p);
        int length = content.length();
        List<String> lines = Files.readAllLines(p);
        int size = lines.size();
        long streamed = Files.lines(p).count();
        boolean exists = Files.exists(p);
        boolean directory = Files.isDirectory(p);
        boolean regular = Files.isRegularFile(p);
        long bytes = Files.size(p);
        Files.createDirectories(q);
        Files.createFile(q);
        Files.copy(p, q);
        Files.move(p, q);
        Files.delete(p);
        boolean removed = Files.deleteIfExists(p);
    }
}
`
	out := renderGoFileFromJava(t, src)
	for _, want := range []string{
		"stdjava.FilesWriteStringExecution(__java2goExecution, p, stdjava.JavaStringLiteralUTF16([]uint16{116, 101, 120, 116}))",
		"stdjava.FilesReadStringReference(p)",
		"stdjava.FilesReadAllLinesReference(p)",
		"stdjava.FilesLinesReference(p).Count()",
		"stdjava.FilesExists(p)",
		"stdjava.FilesIsDirectory(p)",
		"stdjava.FilesIsRegularFile(p)",
		"stdjava.FilesSize(p)",
		"stdjava.FilesCreateDirectories(q)",
		"stdjava.FilesCreateFile(q)",
		"stdjava.FilesCopy(p, q)",
		"stdjava.FilesMove(p, q)",
		"stdjava.FilesDelete(p)",
		"stdjava.FilesDeleteIfExists(p)",
	} {
		assertContains(t, out, want)
	}
	// readString/readAllLines carry their Java result types, so chained calls resolve.
	assertContains(t, out, "stdjava.RequireJavaString(content).Length()")
	assertContains(t, out, "lines.Size()")
}

func TestNio_FilesAcceptsStringsAndFilePaths(t *testing.T) {
	src := `
import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
public class InteropProgram {
    public static void run(String path, File file) throws IOException {
        boolean fromString = Files.exists(Path.of(path));
        Path viaFile = file.toPath();
        String name = viaFile.getFileName().toString();
        long size = Files.size(viaFile);
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.FilesExists(stdjava.PathsGetReference(path))")
	assertContains(t, out, "file.ToPathReference()")
	assertContains(t, out, "stdjava.PathToStringReference(viaFile.GetFileName())")
	assertContains(t, out, "stdjava.FilesSize(viaFile)")
}

func TestNio_UnmodeledOverloadsFallThrough(t *testing.T) {
	src := `
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
public class OptionsProgram {
    public static void run(Path p, Path q) throws IOException {
        Files.copy(p, q, StandardCopyOption.REPLACE_EXISTING);
        String text = Files.readString(p, StandardCharsets.UTF_8);
    }
}
`
	out := renderGoFileFromJava(t, src)
	// Charset and CopyOption overloads are not modeled; they must not silently
	// lower to the option-dropping shims.
	if strings.Contains(out, "stdjava.FilesCopy(") || strings.Contains(out, "stdjava.FilesReadString(") || strings.Contains(out, "stdjava.FilesReadStringReference(") {
		t.Fatalf("an unmodeled Files overload was lowered anyway:\n%s", out)
	}
}
