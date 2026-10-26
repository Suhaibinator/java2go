package propertyprobe;
public class SystemPropertyProfileOracle {
 public static void main(String[] args){
  String[] names={"java.version","java.specification.version","java.class.version","java.runtime.name","java.runtime.version","java.vendor","java.vendor.url","java.vm.name","java.vm.version","java.vm.vendor","java.specification.name","java.specification.vendor","java.home","java.class.path","java.library.path","os.name","os.arch","os.version","user.name","user.home","user.dir","file.separator","path.separator","line.separator","file.encoding","native.encoding","sun.jnu.encoding"};
  for(String name:names){String value=System.getProperty(name);StringBuilder units=new StringBuilder();if(value!=null){for(int i=0;i<value.length();i++){if(i>0)units.append(',');units.append((int)value.charAt(i));}}System.out.println(name+"\t"+(value==null?"NULL":units.toString()));}
 }
}
