package campaign.gsonvariation.workflow;
import campaign.gsonvariation.domain.Batch;
import campaign.gsonvariation.domain.Entry;
import campaign.gsonvariation.domain.DomainTypes;
import campaign.gsonvariation.io.Journal;
import campaign.gsonvariation.io.TrackedReader;
import campaign.gsonvariation.json.DateFactory;
import campaign.gsonvariation.state.Ledger;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import com.google.gson.JsonSyntaxException;
import com.google.gson.TypeAdapter;
import com.google.gson.reflect.TypeToken;
import java.lang.reflect.ParameterizedType;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.time.LocalDate;
import java.util.List;
import java.util.Map;
public final class Pipeline {
    private final int seed;
    public Pipeline(int seed) { this.seed = seed; }
    public String run() throws Exception {
        Journal journal = new Journal();
        DateFactory<String> factory = new DateFactory<>("owner-東京-" + seed, journal);
        Gson gson = new GsonBuilder().registerTypeAdapterFactory(factory).disableHtmlEscaping().serializeNulls().create();
        TypeToken<Batch<LocalDate>> batchType = DomainTypes.batch(LocalDate.class);
        TypeToken<Batch<LocalDate>> literalType = new TypeToken<Batch<LocalDate>>() {};
        if (!batchType.getType().equals(literalType.getType())) throw new IllegalStateException("domain type mismatch");
        TypeAdapter<Batch<LocalDate>> adapter = gson.getAdapter(batchType);
        boolean cacheIdentity = adapter == gson.getAdapter(literalType);
        if (!cacheIdentity) throw new IllegalStateException("Gson cache identity mismatch");
        TypeToken<Map<String, List<Entry<LocalDate>>>> stateType = new TypeToken<Map<String, List<Entry<LocalDate>>>>() {};
        Ledger ledger = new Ledger();
        String first = seed == 41 ? "batch-west.json" : "batch-east.json";
        String second = seed == 41 ? "batch-east.json" : "batch-west.json";
        try (TrackedReader in = new TrackedReader(Paths.get(first), "first", journal, false)) {
            ledger.apply(adapter.fromJson(in), seed);
        }
        try (TrackedReader in = new TrackedReader(Paths.get(second), "second", journal, false)) {
            ledger.apply(adapter.fromJson(in), seed);
        }
        Files.createDirectories(Paths.get("out"));
        Path pending = Paths.get("out/pending.tmp");
        Files.writeString(pending, "pending:" + seed, StandardCharsets.UTF_8);
        String beforeFailure = gson.toJson(ledger.view(), stateType.getType());
        boolean invalidAccepted = false;
        JsonObject error = new JsonObject();
        try (TrackedReader in = new TrackedReader(Paths.get("batch-invalid.json"), "invalid", journal, seed == 41)) {
            adapter.fromJson(in);
            invalidAccepted = true;
        } catch (RuntimeException failure) {
            if (!(failure instanceof JsonSyntaxException)) throw failure;
            error.addProperty("class", failure.getClass().getName());
            error.addProperty("message", failure.getMessage());
            JsonArray suppressed = new JsonArray();
            for (Throwable secondary : failure.getSuppressed()) suppressed.add(secondary.getClass().getName());
            error.add("suppressed", suppressed);
            if (failure.getSuppressed().length != (seed == 41 ? 1 : 0)) throw new IllegalStateException("suppression contract");
            journal.add("caught:" + failure.getClass().getName());
        } finally {
            journal.add("finally:invalid");
            Files.deleteIfExists(pending);
        }
        if (invalidAccepted) throw new IllegalStateException("invalid input accepted");
        boolean stateUnchanged = beforeFailure.equals(gson.toJson(ledger.view(), stateType.getType()));
        if (!stateUnchanged || Files.exists(pending)) throw new IllegalStateException("failure cleanup changed committed state");
        String state = gson.toJson(ledger.view(), stateType.getType());
        Files.writeString(Paths.get("out/state.json"), state, StandardCharsets.UTF_8);
        ParameterizedType parameterized = (ParameterizedType) batchType.getType();
        JsonObject metadata = new JsonObject();
        metadata.addProperty("batch", batchType.getType().getTypeName());
        metadata.addProperty("state", stateType.getType().getTypeName());
        metadata.addProperty("raw", parameterized.getRawType().getTypeName());
        metadata.addProperty("argument", parameterized.getActualTypeArguments()[0].getTypeName());
        metadata.addProperty("equalAnonymousType", batchType.equals(literalType));
        metadata.addProperty("cacheIdentity", cacheIdentity);
        Files.writeString(Paths.get("out/metadata.json"), gson.toJson(metadata), StandardCharsets.UTF_8);
        JsonArray events = new JsonArray();
        for (String event : journal.snapshot()) events.add(event);
        Files.writeString(Paths.get("out/events.json"), gson.toJson(events), StandardCharsets.UTF_8);
        JsonObject summary = new JsonObject();
        summary.addProperty("seed", seed);
        summary.addProperty("regions", ledger.view().size());
        summary.addProperty("rows", ledger.rows());
        summary.addProperty("aliases", ledger.aliasesPreserved());
        summary.addProperty("stateUnchangedAfterFailure", stateUnchanged);
        summary.addProperty("pendingAbsent", !Files.exists(pending));
        summary.add("error", error);
        summary.add("factory", factory.statistics());
        return gson.toJson(summary);
    }
}
