package campaign.data.ingest;

import campaign.data.model.Event;
import campaign.data.model.Reject;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.text.Normalizer;
import org.apache.commons.codec.DecoderException;
import org.apache.commons.codec.binary.Hex;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.codec.net.URLCodec;

public final class EventLoader {
    private static final URLCodec URL = new URLCodec();

    public Loaded load() throws IOException {
        Loaded loaded = new Loaded();
        readResource("events-east.tsv", loaded);
        readResource("events-west.tsv", loaded);
        return loaded;
    }

    private void readResource(String name, Loaded loaded) throws IOException {
        InputStream resource = EventLoader.class.getResourceAsStream("/" + name);
        if (resource == null) throw new IOException("missing resource: " + name);
        try (TrackedInputStream tracked = new TrackedInputStream(resource, loaded);
             BufferedReader reader = new BufferedReader(new InputStreamReader(tracked, StandardCharsets.UTF_8))) {
            String row;
            int line = 0;
            while ((row = reader.readLine()) != null) {
                line++;
                if (row.isEmpty() || row.startsWith("#")) continue;
                parse(name, line, row, loaded);
            }
        }
    }

    private void parse(String source, int line, String row, Loaded loaded) {
        String location = source + ":" + line;
        String fingerprint = DigestUtils.sha256Hex(row).substring(0, 12);
        String[] fields = row.split("\t", -1);
        if (fields.length != 5) {
            loaded.rejects.add(new Reject(location, "COLUMNS", fingerprint));
            return;
        }
        String label;
        try {
            label = Normalizer.normalize(URL.decode(fields[2], "UTF-8"), Normalizer.Form.NFC);
        } catch (DecoderException | java.io.UnsupportedEncodingException ex) {
            loaded.rejects.add(new Reject(location, "URL", fingerprint));
            return;
        }
        try {
            Hex.decodeHex(fields[4]);
        } catch (DecoderException ex) {
            loaded.rejects.add(new Reject(location, "HEX", fingerprint));
            return;
        }
        int amount;
        try {
            amount = Integer.parseInt(fields[3]);
        } catch (NumberFormatException ex) {
            loaded.rejects.add(new Reject(location, "NUMBER", fingerprint));
            return;
        }
        String expected = DigestUtils.sha256Hex(fields[0] + "|" + fields[1] + "|" + label + "|" + amount).substring(0, 12);
        if (!expected.equals(fields[4])) {
            loaded.rejects.add(new Reject(location, "SIGNATURE", fingerprint));
            return;
        }
        if (!fields[1].equals("PUT") && !fields[1].equals("ADD") && !fields[1].equals("DEL")) {
            loaded.rejects.add(new Reject(location, "OPERATION", fingerprint));
            return;
        }
        loaded.events.add(new Event(source, line, fields[0], fields[1], label, amount));
    }
}
