package campaign.concurrent2.io;

import campaign.concurrent2.service.Ledger;
import java.io.ByteArrayInputStream;
import java.io.IOException;

public final class TrackedInput extends ByteArrayInputStream {
    private final Ledger ledger;
    private boolean didClose;

    public TrackedInput(byte[] bytes, Ledger ledger) {
        super(bytes);
        this.ledger = ledger;
        ledger.opened();
    }

    @Override
    public void close() throws IOException {
        if (!didClose) {
            didClose = true;
            ledger.closed();
        }
        super.close();
    }
}
