import java.io.IOException;
import java.nio.file.*;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

// UNRUN public-API lifecycle oracle. Outputs are observations, not goldens.
// All changes stay under a fresh caller-supplied root; traversal never follows links.
public final class DirectoryLifecycleFallbackMatrixOracle {
    private static final String[] CASES = {
        "fresh-leaf", "fresh-nested", "existing-nested", "existing-file",
        "file-parent", "link-dir-leaf", "link-dir-parent", "link-file",
        "dangling-leaf", "dangling-parent", "link-loop-parent", "missing-intermediate-dotdot", "fallback-relative-parent", "fallback-normalized-file", "relative-dot", "null-path"
    };

    private static Path prepare(Path c, String name) throws IOException {
        switch (name) {
        case "fresh-leaf": return c.resolve("leaf");
        case "fresh-nested": return c.resolve("a/b/leaf");
        case "existing-nested":
            Files.createDirectories(c.resolve("a/b")); return c.resolve("a/b");
        case "existing-file":
            Files.write(c.resolve("leaf"), new byte[] {65, 0, 66}); return c.resolve("leaf");
        case "file-parent":
            Files.write(c.resolve("block"), new byte[] {70}); return c.resolve("block/child/leaf");
        case "link-dir-leaf":
            Files.createDirectory(c.resolve("real"));
            Files.createSymbolicLink(c.resolve("link"), Paths.get("real")); return c.resolve("link");
        case "link-dir-parent":
            Files.createDirectory(c.resolve("real"));
            Files.createSymbolicLink(c.resolve("link"), Paths.get("real")); return c.resolve("link/child");
        case "link-file":
            Files.write(c.resolve("real"), new byte[] {76});
            Files.createSymbolicLink(c.resolve("link"), Paths.get("real")); return c.resolve("link");
        case "dangling-leaf":
            Files.createSymbolicLink(c.resolve("link"), Paths.get("missing")); return c.resolve("link");
        case "dangling-parent":
            Files.createSymbolicLink(c.resolve("link"), Paths.get("missing")); return c.resolve("link/child");
        case "link-loop-parent":
            Files.createSymbolicLink(c.resolve("a"), Paths.get("b"));
            Files.createSymbolicLink(c.resolve("b"), Paths.get("a")); return c.resolve("a/child");
        case "missing-intermediate-dotdot": return c.resolve("a/../leaf");
        case "fallback-relative-parent":
            Files.createDirectory(c.resolve("inside")); return c.resolve("inside/a/../../leaf");
        case "fallback-normalized-file":
            Files.write(c.resolve("block"), new byte[] {90}); return c.resolve("a/../block/child");
        case "relative-dot":
            Files.createDirectory(c.resolve("inside"));
            Path cwd = Paths.get("").toAbsolutePath().normalize();
            return cwd.relativize(c).resolve("inside/../dot-leaf");
        case "null-path": return null;
        default: throw new AssertionError("unknown case");
        }
    }

    private static String name(Path root, Path p) {
        String value = root.relativize(p).toString();
        return value.isEmpty() ? "." : value.replace('\\', '/');
    }

    private static String snapshot(Path root) throws IOException {
        List<String> entries = new ArrayList<>();
        Files.walkFileTree(root, new SimpleFileVisitor<Path>() {
            @Override public FileVisitResult preVisitDirectory(Path p, BasicFileAttributes a) {
                entries.add(name(root, p) + ":D"); return FileVisitResult.CONTINUE;
            }
            @Override public FileVisitResult visitFile(Path p, BasicFileAttributes a) throws IOException {
                if (a.isSymbolicLink()) {
                    entries.add(name(root, p) + ":L=" + Files.readSymbolicLink(p).toString().replace('\\', '/'));
                } else {
                    StringBuilder hex = new StringBuilder();
                    for (byte b : Files.readAllBytes(p)) {
                        hex.append(Character.forDigit((b & 255) >>> 4, 16));
                        hex.append(Character.forDigit(b & 15, 16));
                    }
                    entries.add(name(root, p) + ":F=" + hex);
                }
                return FileVisitResult.CONTINUE;
            }
        });
        Collections.sort(entries);
        return String.join("|", entries);
    }

    private static int removeTree(Path root) throws IOException {
        if (!Files.exists(root, LinkOption.NOFOLLOW_LINKS)) return 0;
        int[] removed = {0};
        Files.walkFileTree(root, new SimpleFileVisitor<Path>() {
            @Override public FileVisitResult visitFile(Path p, BasicFileAttributes a) throws IOException {
                Files.delete(p); removed[0]++; return FileVisitResult.CONTINUE;
            }
            @Override public FileVisitResult postVisitDirectory(Path p, IOException failure) throws IOException {
                if (failure != null) throw failure;
                Files.delete(p); removed[0]++; return FileVisitResult.CONTINUE;
            }
        });
        return removed[0];
    }

    // Only the declared fresh absolute root prefix is normalized. Every UTF16
    // message unit, including helpful NPE text, is otherwise preserved.
    private static String messageUnits(Path root, String message) {
        if (message == null) return "<null>";
        String normalized = message.replace(root.toString(), "<ROOT>");
        StringBuilder hex = new StringBuilder();
        for (int i = 0; i < normalized.length(); i++) {
            char unit = normalized.charAt(i);
            for (int shift = 12; shift >= 0; shift -= 4)
                hex.append(Character.forDigit((unit >>> shift) & 15, 16));
        }
        return hex.toString();
    }

    private static void runCase(Path base, String label) throws Exception {
        Path c = base.resolve(label);
        System.out.println("CASE\t" + label);
        try {
            Files.createDirectory(c);
            Path p = prepare(c, label); // setup failure escapes: never a semantic skip
            if (p != null && !p.toAbsolutePath().normalize().startsWith(c)) {
                throw new AssertionError("case escaped owned root");
            }
            System.out.println("BEFORE\t" + snapshot(c));
            try {
                Path result = Files.createDirectories(p);
                System.out.println("RETURN\tidentity=" + (result == p)
                    + "\tpathEquals=" + result.equals(p)
                    + "\tdirectory=" + Files.isDirectory(result));
            } catch (IOException | RuntimeException failure) {
                System.out.println("THROW\t" + failure.getClass().getName());
                System.out.println("MESSAGE_UTF16\t" + messageUnits(base, failure.getMessage()));
                Throwable cause = failure.getCause();
                int depth = 0;
                while (cause != null) {
                    if (depth >= 32) throw new AssertionError("cause chain exceeded bound");
                    System.out.println("CAUSE" + depth + "\t" + cause.getClass().getName()
                        + "\t" + messageUnits(base, cause.getMessage()));
                    cause = cause.getCause(); depth++;
                }
                System.out.println("CAUSE" + depth + "\t<null>");
            }
            System.out.println("AFTER\t" + snapshot(c));
        } finally {
            int removed = removeTree(c);
            System.out.println("CLEANUP\t" + removed + "\tpresent="
                + Files.exists(c, LinkOption.NOFOLLOW_LINKS));
        }
    }

    public static void main(String[] args) throws Exception {
        if (args.length != 1) throw new IllegalArgumentException("one fresh root required");
        Path root = Paths.get(args[0]).toAbsolutePath().normalize();
        if (Files.exists(root, LinkOption.NOFOLLOW_LINKS)) {
            throw new IllegalArgumentException("root already exists");
        }
        Files.createDirectory(root); // caller precreates only the parent
        try {
            for (String label : CASES) runCase(root, label);
        } finally {
            removeTree(root);
            System.out.println("FINAL\tpresent=" + Files.exists(root, LinkOption.NOFOLLOW_LINKS));
        }
    }
}
