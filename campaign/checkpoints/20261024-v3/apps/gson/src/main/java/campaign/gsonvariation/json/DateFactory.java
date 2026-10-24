package campaign.gsonvariation.json;
import campaign.gsonvariation.io.Journal;
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonSyntaxException;
import com.google.gson.TypeAdapter;
import com.google.gson.TypeAdapterFactory;
import com.google.gson.reflect.TypeToken;
import com.google.gson.stream.JsonReader;
import com.google.gson.stream.JsonToken;
import com.google.gson.stream.JsonWriter;
import java.io.IOException;
import java.time.LocalDate;
public final class DateFactory<Owner> implements TypeAdapterFactory {
    private final Owner owner;
    private final Journal journal;
    private int creates;
    private int declines;
    private int reads;
    private int writes;
    public DateFactory(Owner owner, Journal journal) { this.owner = owner; this.journal = journal; }
    @Override @SuppressWarnings("unchecked")
    public <T> TypeAdapter<T> create(Gson gson, TypeToken<T> type) {
        creates++;
        if (type.getRawType() != LocalDate.class) { declines++; return null; }
        TypeAdapter<LocalDate> adapter = new TypeAdapter<LocalDate>() {
            @Override public LocalDate read(JsonReader in) throws IOException {
                reads++;
                journal.add("date-read:" + reads);
                Integer year = null;
                Integer month = null;
                Integer day = null;
                in.beginObject();
                while (in.peek() != JsonToken.END_OBJECT) {
                    String field = in.nextName();
                    if (field.equals("year")) year = in.nextInt();
                    else if (field.equals("month")) month = in.nextInt();
                    else if (field.equals("day")) day = in.nextInt();
                    else in.skipValue();
                }
                in.endObject();
                LocalDate parsed = LocalDate.of(
                    required(year, "year", in).intValue(),
                    required(month, "month", in).intValue(),
                    required(day, "day", in).intValue());
                return required(parsed, "date", in);
            }
            @Override public void write(JsonWriter out, LocalDate value) throws IOException {
                writes++;
                journal.add("date-write:" + writes);
                out.beginObject();
                out.name("year").value(value.getYear());
                out.name("month").value(value.getMonthValue());
                out.name("day").value(value.getDayOfMonth());
                out.endObject();
            }
        }.nullSafe();
        return (TypeAdapter<T>) adapter;
    }
    private static <U> U required(U value, String label, JsonReader reader) {
        if (value == null) throw new JsonSyntaxException("Missing " + label + "; at " + reader.getPreviousPath());
        return value;
    }
    public JsonObject statistics() {
        JsonObject value = new JsonObject();
        value.addProperty("owner", owner.toString());
        value.addProperty("creates", creates);
        value.addProperty("declines", declines);
        value.addProperty("reads", reads);
        value.addProperty("writes", writes);
        return value;
    }
}
