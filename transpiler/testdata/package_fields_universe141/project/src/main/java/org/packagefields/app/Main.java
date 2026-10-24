package org.packagefields.app;
import org.packagefields.state.Store;
public class Main {
  static int effect(int seed){Store.event("E");return seed%3+1;}
  public static void main(String[] args){
    int seed=args.length==0?17:Integer.parseInt(args[0]);
    Store.reset(seed);
    String before=Store.snapshot();
    Store.advance(effect(seed));
    String after=Store.snapshot();
    Probe probe=new Probe(seed);
    System.out.println("seed:"+seed);
    System.out.println("before:"+before);
    System.out.println("after:"+after);
    System.out.println("selectors:"+probe.read(null,7,8,9));
    System.out.println("events:"+Store.events());
    System.out.println("null:"+Store.absent());
  }
}
