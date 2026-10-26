package state;
import java.net.URI;
public class Cell {
 public URI value;
 public int version;
 public String trace="";
 private String input(String text){trace+="input;";return text;}
 public void set(String text) throws Exception {URI next=new URI(input(text));trace+="parsed;";value=next;version++;}
}
