package org.packagefields.state;
final class OtherFields {
  static int nil=5,any=6,iota=7,nil0=401,any0=402,iota0=403;
  static void reset(int seed){nil=seed+1;any=seed+2;iota=seed+3;}
  static void advance(int delta){nil+=delta;any-=delta;iota+=2*delta;}
  static String snapshot(){return nil+":"+any+":"+iota+":"+nil0+":"+any0+":"+iota0;}
}
