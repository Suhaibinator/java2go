package org.packagefields.state;
public final class Store {
  static int nil=1,any=2,iota=3,len=4;
  static int nil0=101,nil1=102,nil2=103,any0=201,iota0=301;
  static String log="";
  public static void reset(int seed){nil=seed;any=seed+10;iota=seed+20;len=4;log="R";OtherFields.reset(seed);}
  public static void event(String text){log+=text;}
  public static void advance(int delta){log+="A";nil+=delta;any+=nil;iota^=delta;OtherFields.advance(delta);}
  public static String snapshot(){log+="S";return nil+":"+any+":"+iota+":"+nil0+":"+nil1+":"+nil2+":"+any0+":"+iota0+":"+OtherFields.snapshot();}
  public static String events(){return log;}
  public static boolean absent(){Object object=null;return object==null;}
}
