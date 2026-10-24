import java.math.BigDecimal;
public final class DecimalCompareProbe {
 private static BigDecimal[] left;
 private static BigDecimal[] right;
 private static int observed;
 private static BigDecimal observe(BigDecimal value){observed++;return value;}
 public static void setup(int seed,int count,int mode){
  left=new BigDecimal[count];right=new BigDecimal[count];
  int state=seed;
  String prefix="";
  if(mode==1)for(int k=0;k<64;k++)prefix+="314159265";
  for(int i=0;i<count;i++){
   state=state*1664525+1013904223;int a=state&2147483647;
   state=state*1664525+1013904223;int b=state&2147483647;
   String sign=(i&1)==0?"":"-";
   String l=sign+prefix+Integer.toString(a);
   String r=sign+prefix+Integer.toString((i&3)==0?a:b);
   if((i&15)==0){l="0";r="0";}
   left[i]=new BigDecimal(l+"E-4");
   right[i]=new BigDecimal(r+(mode==2&&(i&1)==0?"E-5":"E-4"));
  }
 }
 public static long run(int repeats){
  long checksum=0;
  for(int round=0;round<repeats;round++)
   for(int i=0;i<left.length;i++)checksum+=(long)(i+1)*left[i].compareTo(right[i]);
  return checksum;
 }
 public static String audit(){
  String[] values={"2.00","-2.00","0.00","123456789012345678901234567890.00","1E-2147483647","2E-2147483647","1E+2147483647","2E+2147483647"};
  int checks=0;
  for(int i=0;i<values.length;i++){
   BigDecimal a=new BigDecimal(values[i]);BigDecimal twin=new BigDecimal(values[i]);
   Object identity=a;String before=a.toString();int scale=a.scale();int hash=a.hashCode();
   for(int j=0;j<values.length;j++){
    BigDecimal b=new BigDecimal(values[j]);String otherBefore=b.toString();int otherHash=b.hashCode();int otherScale=b.scale();
    int forward=a.compareTo(b);int reverse=b.compareTo(a);
    if(forward!=-reverse)throw new AssertionError("antisymmetry");
    if(!otherBefore.equals(b.toString())||otherHash!=b.hashCode()||otherScale!=b.scale())throw new AssertionError("right mutation");
    checks=checks*31+forward;
   }
   if(a!=identity||a==twin||!a.equals(twin)||a.compareTo(twin)!=0||!before.equals(a.toString())||hash!=a.hashCode()||scale!=a.scale())throw new AssertionError("identity or left mutation");
  }
  BigDecimal a=new BigDecimal("2.0"),b=new BigDecimal("2.00");
  if(a.compareTo(b)!=0||a.equals(b))throw new AssertionError("scale semantics");
  int caught=0;observed=0;
  try{observe(null).compareTo(observe(a));}catch(NullPointerException expected){caught++;}
  try{observe(a).compareTo(observe(null));}catch(NullPointerException expected){caught++;}
  try{observe(null).compareTo(observe(null));}catch(NullPointerException expected){caught++;}
  if(caught!=3||observed!=6)throw new AssertionError("null or evaluation order");
  return checks+":"+caught+":"+observed;
 }
}
