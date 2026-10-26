package transpiler

import "testing"

func TestCampaignGroupedHashStrictJVM(t *testing.T) {
	old := diagnostics.strict
	setStrictMode(true)
	t.Cleanup(func() { setStrictMode(old) })
	const source = `public class GroupedHashOracle {
 static int evaluations;
 static java.lang.String next(){evaluations++;return new java.lang.String(new char[]{'A', '\uD800',0,'\uDC00','\uFFFF'});}
 static class String { public int hashCode(){synchronized(this){return Thread.holdsLock(this)?73:91;}} }
 public static java.lang.String run(){
  int[] lengths={0,1,2,3,4,5,6,7,8,9,15,16,17,31,32,33,127,128,129,1023,1024,1025,8193};
  java.lang.String out="";
  for(int n:lengths){
   for(int mode=0;mode<4;mode++){
    char[] units=new char[n];int state=17;
    for(int i=0;i<n;i++){state=state*1664525+1013904223;int choice=(state>>>24)&3;
     if(mode==0)units[i]=(char)('a'+choice);
     else if(mode==1)units[i]=(char)(0xe000+choice);
     else if(mode==2)units[i]=(char)(choice==0?0:choice==1?0xd800:choice==2?0xdc00:0xffff);
     else units[i]=0;
    }
    java.lang.String value=new java.lang.String(units);
    Object erased=value;CharSequence sequence=value;
    out+=n+":"+mode+":"+value.hashCode()+":"+erased.hashCode()+":"+sequence.hashCode()+":"+new java.lang.String(value).hashCode()+";";
   }
  }
  java.util.function.ToIntFunction<java.lang.String> ref=java.lang.String::hashCode;
  out+="ref:"+ref.applyAsInt(next())+":"+evaluations+":"+new String().hashCode()+";";
  java.lang.String absent=null;try{absent.hashCode();out+="bad";}catch(NullPointerException expected){out+="NPE";}
  return out;
 }
}`
	verifyCanonicalStringStreamOracle(t, "GroupedHashOracle", source)
}
