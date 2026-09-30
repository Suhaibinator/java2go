package campaign.data2.domain;

import com.google.gson.annotations.SerializedName;
import java.util.List;
import java.util.Map;

public final class Command {
    @SerializedName("kind")
    public String kind;
    @SerializedName(value = "order_id", alternate = {"id"})
    public String orderId;
    @SerializedName(value = "customer", alternate = {"client"})
    public String customer;
    @SerializedName("lines")
    public List<LineItem> lines;
    @SerializedName("sku")
    public String sku;
    @SerializedName("delta")
    public int delta;
    @SerializedName("note")
    public String note;
    @SerializedName("metadata")
    public Map<String, String> metadata;
}
