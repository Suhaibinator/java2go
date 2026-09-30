package example.app;
import example.domain.Invoice;
import java.nio.file.Files;
import java.nio.file.Paths;
public class InvoiceApp {
    public static void main(String[] commandLine) throws Exception {
        System.out.println(Invoice.render(commandLine[0], 6 + OtherApp.main(0)));
        System.out.println(Files.readString(Paths.get("resources/banner.txt")).trim());
    }
}
