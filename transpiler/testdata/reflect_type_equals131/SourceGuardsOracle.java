package eqguards;
import foreign.Type;
public class SourceGuardsOracle {
 static class Source implements Type {public String equals(String other){return "foreign:"+other;}}
 static <Type extends foreign.Type> String selected(Type value,String other){return value.equals(other);}
 static class TypeVariable {public String equals(String other){return "source:"+other;}public int hashCode(){return 47;}}
 public static void main(String[] args){Type source=new Source();System.out.println("foreign="+source.equals("direct")+":"+selected(new Source(),"bound"));TypeVariable variable=new TypeVariable();System.out.println("source="+variable.equals("direct")+":"+variable.hashCode());java.lang.reflect.Type canonical=String.class;System.out.println("canonical="+canonical.equals(String.class)+":"+!canonical.equals(null));}
}
