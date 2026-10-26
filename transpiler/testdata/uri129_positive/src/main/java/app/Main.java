package app;
import p.Box;
public class Main {
  static <URI> String read(Box box){return box.value.toString()+":"+box.get().toString()+":"+box.get().getClass().getName();}
  public static void main(String[] args) throws Exception {
    Box box=new Box("urn:e\u0301");
    System.out.println(read(box));
    System.out.println(new java.net.URL("text").toString());
    foreign.URI URI=new foreign.URI();
    System.out.println(URI.normalize());
    Object object=box.value;
    System.out.println(object.getClass().getName());
    synchronized(box.value){box.value.notify();box.value.notifyAll();}
    System.out.println(box.value.toASCIIString());
  }
}