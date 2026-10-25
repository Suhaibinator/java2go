package propertyprobe;
public class SystemPropertyStateOracle {
 static final String key="java2go.property.contract.state";
 static int calls;
 static String countedValue(){calls++;return "evaluated";}
 static void invalid(int row){
  switch(row){
   case 0:System.getProperty((String)null);break;
   case 1:System.getProperty("");break;
   case 2:System.getProperty((String)null,countedValue());break;
   case 3:System.getProperty("",countedValue());break;
   case 4:System.setProperty((String)null,countedValue());break;
   case 5:System.setProperty("",countedValue());break;
   case 6:System.setProperty(key,(String)null);break;
   case 7:System.setProperty((String)null,(String)null);break;
   case 8:System.setProperty("",(String)null);break;
   case 9:System.clearProperty((String)null);break;
   case 10:System.clearProperty("");break;
   default:throw new AssertionError("bad row");
  }
 }
 static void restore(String name,String saved){if(saved==null)System.clearProperty(name);else System.setProperty(name,saved);}
 public static void main(String[] args){
  String old=System.getProperty(key), version=System.getProperty("java.version");
  String whitespace=" ", oldWhitespace=System.getProperty(whitespace);
  String surrogate="java2go.property.\uD800", replacement="java2go.property.\uFFFD";
  String oldSurrogate=System.getProperty(surrogate), oldReplacement=System.getProperty(replacement);
  try{
   System.clearProperty(key);
   String fallback=new String("fallback"), first=new String("first"), second=new String("second");
   System.out.println("missing:"+(System.getProperty(key)==null)+":"+(System.getProperty(key,fallback)==fallback)+":"+(System.getProperty(key,null)==null));
   System.out.println("first:"+(System.setProperty(key,first)==null)+":"+(System.getProperty(new String(key))==first)+":"+(System.getProperty(key,fallback)==first));
   System.out.println("replace:"+(System.setProperty(new String(key),second)==first)+":"+(System.getProperty(key)==second));
   System.out.println("clear:"+(System.clearProperty(new String(key))==second)+":"+(System.clearProperty(key)==null));
   String empty=new String("");System.setProperty(key,empty);System.out.println("empty:"+(System.getProperty(key)==empty)+":"+(System.clearProperty(key)==empty));
   System.setProperty(whitespace,first);System.out.println("whitespace:"+(System.getProperty(whitespace)==first));
   System.setProperty(surrogate,first);System.setProperty(replacement,second);
   System.out.println("UTF16:"+(System.getProperty(new String(surrogate))==first)+":"+(System.getProperty(replacement)==second));
   System.out.println("profile:"+(version!=null&&!version.isEmpty())+":"+System.getProperty("java.specification.version")+":"+System.getProperty("java.class.version"));
   System.clearProperty("java.version");System.out.println("clear-default:"+(System.getProperty("java.version")==null)+":"+(System.getProperty("java.version",fallback)==fallback));
   System.setProperty("java.version",second);System.out.println("override-default:"+(System.getProperty("java.version")==second));
   for(int row=0;row<11;row++){
    calls=0;
    try{invalid(row);System.out.println("invalid:"+row+":ACCEPTED:"+calls);}
    catch(RuntimeException failure){System.out.println("invalid:"+row+":"+failure.getClass().getName()+":"+failure.getMessage()+":"+calls);}
   }
   System.out.println("failed-set-unchanged:"+(System.getProperty(key)==null));
  }finally{restore(key,old);restore(whitespace,oldWhitespace);restore(surrogate,oldSurrogate);restore(replacement,oldReplacement);restore("java.version",version);}
 }
}
