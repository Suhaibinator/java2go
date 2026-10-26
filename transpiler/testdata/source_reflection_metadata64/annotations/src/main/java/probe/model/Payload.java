package probe.model;
import com.google.gson.annotations.SerializedName;
public final class Payload {
 @SerializedName("canonical") public String defaultName;
 @SerializedName(value="order_id",alternate={"id","identifier"}) private String identifier;
 public Payload(String value){defaultName=value;identifier=value+"-id";}
 public String identifier(){return identifier;}
}
