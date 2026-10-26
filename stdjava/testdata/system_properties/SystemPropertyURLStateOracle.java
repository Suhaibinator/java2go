package propertyprobe;
public class SystemPropertyURLStateOracle {
 public static void main(String[] args){
  String[] keys={"java.protocol.handler.pkgs","jdk.net.url.delayParsing"};String[] values={new String("mine"),new String("true")};
  for(int i=0;i<keys.length;i++){String key=keys[i],old=System.getProperty(key);try{System.clearProperty(key);System.out.println("set:"+(System.setProperty(key,values[i])==null)+":"+(System.getProperty(new String(key))==values[i]));System.out.println("clear:"+(System.clearProperty(key)==values[i])+":"+(System.getProperty(key)==null));}finally{if(old==null)System.clearProperty(key);else System.setProperty(key,old);}}
 }
}
