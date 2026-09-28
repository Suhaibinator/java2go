package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeReflectTypeProtocols(t *testing.T) {
	const source = `import java.lang.reflect.*;
import java.util.function.Supplier;
public class CampaignRuntimeReflectTypeProtocols {
 static class Parameter implements ParameterizedType {
  private final Class<?>[] arguments=new Class<?>[]{String.class};
  public Class<?>[] getActualTypeArguments(){return arguments;}
  public Class<?> getRawType(){synchronized(this){return java.util.List.class;}}
  public Type getOwnerType(){return null;}
  public String toString(){return "custom-parameter";}
 }
 static class Derived extends Parameter {
  public Class<?> getRawType(){return java.util.Set.class;}
 }
 static class ArrayType implements GenericArrayType {
  private final ParameterizedType component;
  ArrayType(ParameterizedType component){this.component=component;}
  public ParameterizedType getGenericComponentType(){return component;}
  public String toString(){return component.getTypeName()+"[]";}
 }
 static class Wild implements WildcardType {
  public Type[] getUpperBounds(){return new Type[]{Number.class};}
  public Type[] getLowerBounds(){return new Type[0];}
  public String toString(){return "? extends java.lang.Number";}
 }
 static class NullValues extends Parameter {
  public Class<?>[] getActualTypeArguments(){return null;}
  public Class<?> getRawType(){return null;}
 }
 static class Failure extends Parameter {
  public Class<?> getRawType(){throw new IllegalStateException("accessor-failed");}
 }
 public static String run(){
  ParameterizedType parameter=new Parameter();
  Type[] args=parameter.getActualTypeArguments();
  boolean actualArray=args.getClass()==Class[].class;
  boolean sameArray=args==parameter.getActualTypeArguments();
  Type raw;
  synchronized(parameter){raw=parameter.getRawType();}
  GenericArrayType array=new ArrayType(parameter);
  WildcardType wildcard=new Wild();
  ParameterizedType derived=new Derived();
  Supplier<Type> reference=derived::getRawType;
  boolean failed=false;
  try{ParameterizedType bad=new Failure();bad.getRawType();}catch(IllegalStateException expected){failed=true;}
  Type[] kinds=new Type[]{parameter,array,wildcard};
  ParameterizedType nullValues=new NullValues();
  return raw.getTypeName()+":"+args[0].getTypeName()+":"+actualArray+":"+sameArray+":"+(parameter.getOwnerType()==null)+":"+array.getGenericComponentType().getTypeName()+":"+array.getTypeName()+":"+wildcard.getUpperBounds()[0].getTypeName()+":"+wildcard.getLowerBounds().length+":"+reference.get().getTypeName()+":"+failed+":"+(kinds[0] instanceof ParameterizedType)+":"+(kinds[1] instanceof GenericArrayType)+":"+(kinds[2] instanceof WildcardType)+":"+(nullValues.getActualTypeArguments()==null)+":"+(nullValues.getRawType()==null);
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeReflectTypeProtocols", source)
	t.Logf("JVM protocol oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestProtocolsOracle(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
