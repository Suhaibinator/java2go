package probe;
public class Canonical<Comparable> implements java.lang.Comparable<Main.Item> {
 int key;public Canonical(int k){key=k;}
 public int compareTo(Main.Item value){return value==null?-37:key-value.key;}
}
