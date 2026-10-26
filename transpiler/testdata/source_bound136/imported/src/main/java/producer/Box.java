package producer;
public class Box<T extends model.Item> implements java.lang.Comparable<T> {
 public T value; public Box(T value){this.value=value;}
 public T read(){return value;} public void put(T value){this.value=value;}
 public int compareTo(T value){return value==null?-23:this.value.key-value.key;}
 public <T> T echo(T value){return value;}
}
