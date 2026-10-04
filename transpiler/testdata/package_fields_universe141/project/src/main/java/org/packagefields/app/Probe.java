package org.packagefields.app;
final class Probe {
  int nil,len,any,iota;
  Probe(int seed){this.nil=seed;this.len=2;this.any=3;this.iota=4;}
  int read(Object nil,int len,int any,int iota){
    int nilJava2goLocal=40,nilJava2goLocal1=41;
    int[] xs={len,any,iota};
    Object absent=null;
    try {if(absent!=null)throw new IllegalStateException("unreachable");}
    finally {nilJava2goLocal+=1;}
    return this.nil+this.len+this.any+this.iota+xs.length+(nil==null?1:0)+nilJava2goLocal+nilJava2goLocal1;
  }
}
