package foreign;
public class URI {public String value;public URI(String value){this.value=value;}public String toASCIIString(){return "source:"+value;}public String toString(){return "foreign:"+value;}}
