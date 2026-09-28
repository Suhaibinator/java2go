package campaign.data2.ingest;

import campaign.data2.domain.Command;
import campaign.data2.domain.LocatedCommand;
import com.google.gson.Gson;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParseException;
import com.google.gson.JsonParser;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;

public final class JsonImporter {
    private final Gson gson;

    public JsonImporter(Gson gson) { this.gson = gson; }

    public Loaded load() throws IOException {
        Loaded loaded = new Loaded();
        read("orders-east.jsonl", loaded);
        read("orders-west.jsonl", loaded);
        return loaded;
    }

    private void read(String name, Loaded loaded) throws IOException {
        InputStream input = JsonImporter.class.getResourceAsStream("/" + name);
        if (input == null) throw new IOException("missing resource: " + name);
        try (InputStream source = input;
             BufferedReader reader = new BufferedReader(new InputStreamReader(source, StandardCharsets.UTF_8))) {
            loaded.resourceCount++;
            String row;
            int line = 0;
            while ((row = reader.readLine()) != null) {
                line++;
                loaded.lineCount++;
                if (row.isBlank() || row.startsWith("#")) continue;
                String location = name + ":" + line;
                String fingerprint = Integer.toHexString(row.hashCode());
                try {
                    JsonElement tree = JsonParser.parseString(row);
                    if (!tree.isJsonObject()) {
                        loaded.issues.add(new Issue(location, "SHAPE", fingerprint));
                        continue;
                    }
                    JsonObject object = tree.getAsJsonObject();
                    if (!object.has("kind") || object.get("kind").isJsonNull()) {
                        loaded.issues.add(new Issue(location, "KIND", fingerprint));
                        continue;
                    }
                    Command command = gson.fromJson(tree, Command.class);
                    loaded.commands.add(new LocatedCommand(name, line, command));
                } catch (JsonParseException | IllegalStateException ex) {
                    loaded.issues.add(new Issue(location, "JSON", fingerprint));
                }
            }
        }
    }
}
