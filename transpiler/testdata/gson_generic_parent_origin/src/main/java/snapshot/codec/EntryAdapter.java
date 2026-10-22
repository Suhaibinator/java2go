package snapshot.codec;

import com.google.gson.TypeAdapter;
import com.google.gson.stream.JsonReader;
import com.google.gson.stream.JsonToken;
import com.google.gson.stream.JsonWriter;
import java.io.IOException;
import snapshot.model.Entry;

public final class EntryAdapter extends TypeAdapter<Entry> {
    private final String label;
    public EntryAdapter(String label) { this.label = label; }
    @Override
    public Entry read(JsonReader input) throws IOException {
        if (input.peek() == JsonToken.NULL) { input.nextNull(); return null; }
        String orderId = "";
        int delta = 0;
        input.beginObject();
        while (input.hasNext()) {
            String name = input.nextName();
            if (name.equals("order_id") || name.equals("id")) orderId = input.nextString();
            else if (name.equals("delta")) delta = input.nextInt();
            else input.skipValue();
        }
        input.endObject();
        return new Entry(orderId, delta, label);
    }
    @Override
    public void write(JsonWriter output, Entry value) throws IOException {
        if (value == null) { output.nullValue(); return; }
        output.beginObject();
        output.name("adapter").value(label);
        output.name("order_id").value(value.orderId);
        output.name("delta").value(value.delta);
        output.endObject();
    }
}
