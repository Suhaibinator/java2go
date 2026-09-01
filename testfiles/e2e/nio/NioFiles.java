import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.io.PrintStream;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;

public class NioFiles {
    public static void main(String[] args) throws IOException {
        // Path arithmetic touches no filesystem, so these literals are safe to print.
        Path sample = Paths.get("alpha", "beta", "gamma.txt");
        System.out.println("nameCount " + sample.getNameCount());
        System.out.println("fileName " + sample.getFileName().toString());
        System.out.println("parent " + sample.getParent().toString());
        System.out.println("startsWith " + sample.startsWith("alpha/beta"));
        System.out.println("startsWithPartial " + sample.startsWith("alph"));
        System.out.println("endsWith " + sample.endsWith("gamma.txt"));
        System.out.println("normalize " + Paths.get("alpha/./beta/../gamma").normalize().toString());
        System.out.println("resolve " + Paths.get("alpha").resolve("beta").toString());
        System.out.println("absolute " + Paths.get("alpha").toAbsolutePath().startsWith("/"));
        System.out.println("pathOf " + Path.of("alpha", "beta").toString());

        // Everything below is anchored on a temp file, so only derived values are
        // printed: the Java and Go runs use different directories.
        File temp = File.createTempFile("java2go_nio", ".txt");
        Path base = temp.toPath();
        System.out.println("tempSuffix " + base.getFileName().toString().endsWith(".txt"));

        Path dir = base.getParent().resolve(base.getFileName().toString() + ".d");
        Files.createDirectories(dir);
        System.out.println("dirExists " + Files.exists(dir));
        System.out.println("isDirectory " + Files.isDirectory(dir));

        Path text = dir.resolve("notes.txt");
        Files.writeString(text, "alpha\nbeta\n");
        System.out.println("readString " + Files.readString(text).length());
        System.out.println("size " + Files.size(text));
        System.out.println("isRegularFile " + Files.isRegularFile(text));

        List<String> lines = Files.readAllLines(text);
        System.out.println("lineCount " + lines.size());
        for (String line : lines) {
            System.out.println("line " + line);
        }
        System.out.println("streamCount " + Files.lines(text).count());

        List<String> written = new ArrayList<>();
        written.add("one");
        written.add("two");
        written.add("three");
        Path listed = dir.resolve("listed.txt");
        Files.write(listed, written);
        System.out.println("writtenLines " + Files.readAllLines(listed).size());
        System.out.println("writtenSize " + Files.size(listed));

        Path copied = dir.resolve("copied.txt");
        Files.copy(listed, copied);
        System.out.println("copiedSize " + Files.size(copied));
        Path moved = dir.resolve("moved.txt");
        Files.move(copied, moved);
        System.out.println("movedExists " + Files.exists(moved));
        System.out.println("copiedGone " + Files.exists(copied));

        Path created = dir.resolve("created.txt");
        Files.createFile(created);
        System.out.println("createdSize " + Files.size(created));

        // java.io writers and streams over the same temp tree.
        PrintWriter writer = new PrintWriter(text.toFile());
        writer.println("first");
        writer.close();
        FileWriter appender = new FileWriter(text.toFile(), true);
        appender.write("second\n");
        appender.close();
        System.out.println("afterAppend " + Files.readAllLines(text).size());

        BufferedWriter buffered = new BufferedWriter(new FileWriter(created.toFile()));
        buffered.write("buffered");
        buffered.newLine();
        buffered.write("writer");
        buffered.newLine();
        buffered.close();
        System.out.println("bufferedSize " + Files.size(created));

        BufferedReader reader = new BufferedReader(new FileReader(created.toFile()));
        System.out.println("readerLines " + reader.lines().count());
        reader.close();

        StringWriter memory = new StringWriter();
        PrintWriter memoryWriter = new PrintWriter(memory);
        memoryWriter.print("in-memory");
        memoryWriter.flush();
        System.out.println("stringWriter " + memory.toString());

        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        PrintStream stream = new PrintStream(bytes);
        stream.print("captured");
        stream.flush();
        System.out.println("byteStream " + bytes.toString() + " " + bytes.size());

        FileInputStream input = new FileInputStream(created.toFile());
        System.out.println("firstByte " + input.read());
        input.close();

        Files.delete(created);
        System.out.println("deleted " + Files.exists(created));
        System.out.println("deleteIfExists " + Files.deleteIfExists(created));

        Files.delete(listed);
        Files.delete(moved);
        Files.delete(text);
        Files.delete(dir);
        System.out.println("dirGone " + Files.exists(dir));
        System.out.println("tempDeleted " + temp.delete());
    }
}
