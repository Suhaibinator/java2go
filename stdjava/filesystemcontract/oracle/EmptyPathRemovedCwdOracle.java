import java.io.IOException;
import java.nio.file.*;

// Isolated child only: the caller supplies an already-created empty cwd.
// No parent cwd, permissions, locale, or global process state is changed.
public final class EmptyPathRemovedCwdOracle {
    private static String units(Path own, String text) {
        if (text == null) return "<null>";
        text = text.replace(own.toString(), "<ROOT>");
        StringBuilder output = new StringBuilder();
        for (int i=0; i<text.length(); i++)
            for (int shift=12; shift>=0; shift-=4)
                output.append(Character.forDigit((text.charAt(i) >>> shift) & 15,16));
        return output.toString();
    }
    public static void main(String[] args) throws Exception {
        if (args.length != 1) throw new IllegalArgumentException("one owned cwd required");
        Path own = Paths.get(args[0]);
        if (!own.isAbsolute() || !own.equals(Paths.get("").toAbsolutePath()))
            throw new AssertionError("caller did not isolate owned cwd");
        System.out.println("BEFORE\tcwd-exists="+Files.exists(own,LinkOption.NOFOLLOW_LINKS));
        Files.delete(own); // setup failure escapes, never a semantic skip
        System.out.println("REMOVED\tcwd-exists="+Files.exists(own,LinkOption.NOFOLLOW_LINKS));
        Path input = Paths.get("");
        try {
            Path result = Files.createDirectories(input);
            System.out.println("RETURN\tidentity="+(result==input)+"\tlexical-empty="+result.toString().isEmpty()+"\tdirectory="+Files.isDirectory(result));
        } catch (IOException | RuntimeException failure) {
            System.out.println("THROW\t"+failure.getClass().getName());
            System.out.println("MESSAGE_UTF16\t"+units(own,failure.getMessage()));
            Throwable cause=failure.getCause();int depth=0;
            while(cause!=null){if(depth>=32)throw new AssertionError("cause bound");System.out.println("CAUSE"+depth+"\t"+cause.getClass().getName()+"\t"+units(own,cause.getMessage()));cause=cause.getCause();depth++;}
            System.out.println("CAUSE"+depth+"\t<null>");
        }
        System.out.println("AFTER\tcwd-exists="+Files.exists(own,LinkOption.NOFOLLOW_LINKS)+"\tlocal-directory="+Files.isDirectory(Paths.get(".")));
    }
}
