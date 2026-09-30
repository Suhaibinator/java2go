import java.io.BufferedReader;
import java.io.File;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.io.PrintWriter;

public class FileRoundTrip {
    public static void main(String[] args) throws IOException {
        File f = File.createTempFile("java2go_e2e", ".txt");

        // write a few deterministic lines
        PrintWriter writer = new PrintWriter(new FileWriter(f));
        writer.println("line one");
        writer.println("line two");
        writer.print("no newline tail");
        writer.close();

        // metadata checks (deterministic: existence and extension, NOT the random path)
        System.out.println("exists " + f.exists());
        System.out.println("endsWith " + f.getName().endsWith(".txt"));

        // read it back
        BufferedReader reader = new BufferedReader(new FileReader(f));
        String line;
        int count = 0;
        while ((line = reader.readLine()) != null) {
            count++;
            System.out.println(count + ": " + line);
        }
        reader.close();
        System.out.println("lines " + count);

        // clean up; print deterministic confirmation, not the path
        boolean deleted = f.delete();
        System.out.println("deleted " + deleted);
        System.out.println("exists " + f.exists());
    }
}
