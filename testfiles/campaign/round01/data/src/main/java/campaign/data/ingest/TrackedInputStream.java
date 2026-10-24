package campaign.data.ingest;

import java.io.FilterInputStream;
import java.io.IOException;
import java.io.InputStream;

final class TrackedInputStream extends FilterInputStream {
    private final Loaded loaded;
    private boolean closed;

    TrackedInputStream(InputStream source, Loaded loaded) {
        super(source);
        this.loaded = loaded;
    }

    @Override public int read() throws IOException {
        int value = super.read();
        if (value >= 0) loaded.bytesRead++;
        return value;
    }

    @Override public int read(byte[] bytes, int offset, int length) throws IOException {
        int count = in.read(bytes, offset, length);
        if (count > 0) loaded.bytesRead += count;
        return count;
    }

    @Override public void close() throws IOException {
        if (!closed) {
            closed = true;
            try {
                super.close();
            } finally {
                loaded.resourcesClosed++;
            }
        }
    }
}
