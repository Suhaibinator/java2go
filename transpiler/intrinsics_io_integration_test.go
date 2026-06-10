package transpiler

import "testing"

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
	assertContains(t, out, "w.Println(\"x\")")
	assertContains(t, out, "w.Print(\"y\")")
	assertContains(t, out, "w.Close()")
	assertContains(t, out, "stdjava.NewJavaFile(path)")
	assertContains(t, out, "f.Exists()")
	assertContains(t, out, "f.GetName()")
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
	assertContains(t, out, "stdjava.NewBufferedReader(path)")
	assertContains(t, out, "r.ReadLine()")
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
	assertContains(t, out, "stdjava.NewScannerFile(path)")
	assertContains(t, out, "sc.NextInt()")
	assertContains(t, out, "sc.Next()")
}
