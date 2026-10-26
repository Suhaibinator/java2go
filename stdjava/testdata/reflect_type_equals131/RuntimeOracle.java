package eqruntime;
import java.lang.reflect.*;
import java.lang.annotation.*;
import sun.reflect.generics.reflectiveObjects.*;
public class RuntimeOracle {
 static class Model<T>{java.util.List<? extends String> upper;}
 static class Peer<T>{}
 static String events="";
 static class Effect implements Type {
  public boolean equals(Object other){events+=other==null?"N":"E";return other==null;}
  public int hashCode(){return 23;}
 }
 static class Parameter implements ParameterizedType {
  Type owner;Type raw;Type[] args;boolean fail;
  Parameter(Type owner,Type raw,Type[] args,boolean fail){this.owner=owner;this.raw=raw;this.args=args;this.fail=fail;}
  public Type getOwnerType(){events+="O";return owner;}
  public Type getRawType(){events+="R";if(fail)throw new IllegalStateException("raw");return raw;}
  public Type[] getActualTypeArguments(){events+="A";return args;}
 }
 static class Array implements GenericArrayType {
  Type component;Array(Type component){this.component=component;}
  public Type getGenericComponentType(){events+="G";return component;}
 }
 static class Wild implements WildcardType {
  Type[] lower;Type[] upper;Wild(Type[] lower,Type[] upper){this.lower=lower;this.upper=upper;}
  public Type[] getLowerBounds(){events+="L";return lower;}
  public Type[] getUpperBounds(){events+="U";return upper;}
 }
 static class Foreign implements TypeVariable<GenericDeclaration> {
  GenericDeclaration declaration;Foreign(GenericDeclaration declaration){this.declaration=declaration;}
  public String getName(){events+="N";return "T";}
  public GenericDeclaration getGenericDeclaration(){events+="D";return declaration;}
  public Type[] getBounds(){return new Type[]{Object.class};}
  public AnnotatedType[] getAnnotatedBounds(){return new AnnotatedType[0];}
  public <A extends Annotation>A getAnnotation(Class<A> type){return null;}
  public Annotation[] getAnnotations(){return new Annotation[0];}
  public Annotation[] getDeclaredAnnotations(){return new Annotation[0];}
 }
 public static void main(String[] args)throws Exception{
  Effect effect=new Effect();GenericArrayType left=GenericArrayTypeImpl.make(effect);
  events="";boolean same=left.equals(new Array(effect));String sameEvents=events;
  events="";boolean nil=left.equals(new Array(null));String nullEvents=events;
  System.out.println("component="+same+":"+sameEvents+":"+nil+":"+nullEvents);
  ParameterizedType parameter=ParameterizedTypeImpl.make(java.util.List.class,new Type[]{String.class},Model.class);
  events="";try{parameter.equals(new Parameter(null,java.util.List.class,new Type[]{String.class},true));System.out.println("parameter=wrong");}catch(IllegalStateException expected){System.out.println("parameter="+expected.getMessage()+":"+events);}
  WildcardType wild=(WildcardType)((ParameterizedType)Model.class.getDeclaredField("upper").getGenericType()).getActualTypeArguments()[0];
  events="";boolean equal=wild.equals(new Wild(new Type[0],new Type[]{String.class}));System.out.println("wild="+equal+":"+events);
  ParameterizedType plain=ParameterizedTypeImpl.make(java.util.List.class,new Type[]{String.class},null);
  events="";boolean arrays=plain.equals(new Parameter(null,java.util.List.class,null,false));System.out.println("null-array="+arrays+":"+events);
  TypeVariable<?> variable=Model.class.getTypeParameters()[0];events="";boolean foreign=variable.equals(new Foreign(Model.class));
  System.out.println("variable="+foreign+":"+events+":"+variable.equals(Model.class.getTypeParameters()[0])+":"+!variable.equals(Peer.class.getTypeParameters()[0])+":"+(variable.hashCode()==Model.class.getTypeParameters()[0].hashCode()));
  ParameterizedType identity=ParameterizedTypeImpl.make(java.util.List.class,new Type[]{effect},effect);events="";boolean self=identity.equals(identity);System.out.println("identity="+self+":"+events);
 }
}
