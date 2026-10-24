package campaign.gsonvariation.io;
import java.io.IOException;
import java.io.Reader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
public final class TrackedReader extends Reader {
    private final Reader delegate;
    private final Journal journal;
    private final String label;
    private final boolean closeFault;
    private boolean closed;
    public TrackedReader(Path path, String label, Journal journal, boolean closeFault) throws IOException {
        this.delegate = Files.newBufferedReader(path, StandardCharsets.UTF_8);
        this.label = label;
        this.journal = journal;
        this.closeFault = closeFault;
        journal.add("open:" + label);
    }
    @Override public int read(char[] target, int offset, int length) throws IOException {
        return delegate.read(target, offset, length);
    }
    @Override public void close() throws IOException {
        if (closed) throw new IOException("duplicate close:" + label);
        closed = true;
        delegate.close();
        journal.add("close:" + label);
        if (closeFault) {
            journal.add("close-fault:" + label);
            throw new IOException("close fault:" + label);
        }
    }
}
