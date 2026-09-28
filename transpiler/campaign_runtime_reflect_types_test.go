package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeReflectTypeNames(t *testing.T) {
	const source = `import java.lang.reflect.Type;
public class CampaignRuntimeReflectTypeNames {
 static class DefaultType implements Type {
  public String toString(){return "custom-default";}
 }
 static class InheritedType extends DefaultType {}
 static class PlainType implements Type { public int hashCode(){return 42;} }
 static class NamedType implements Type {
  public String toString(){return "different-toString";}
  public String getTypeName(){return "custom-name";}
 }
 static String describe(Type type){return type.getTypeName();}
 public static String run(){
  Type stringType=String.class;
  Type arrayType=String[][].class;
  Type primitiveType=int.class;
  Type primitiveArray=int[][].class;
  Type nestedType=DefaultType.class;
  Type defaultType=new DefaultType();
  Type namedType=new NamedType();
  Object classValue=String.class;
  Object custom=defaultType;
  Type inherited=new InheritedType();
  Type[] array=new Type[]{String.class,defaultType,inherited};
  Object arrayObject=array;
  Type cast=(Type)classValue;
  return describe(stringType)+":"+describe(arrayType)+":"+describe(primitiveType)+":"+describe(primitiveArray)+":"+describe(nestedType)+":"+describe(defaultType)+":"+describe(namedType)+":"+(classValue instanceof Type)+":"+(classValue instanceof Class)+":"+(custom instanceof Type)+":"+stringType.getClass().getName()+":"+(inherited instanceof Type)+":"+describe(array[0])+":"+describe(array[2])+":"+(arrayObject instanceof Type[])+":"+describe(cast)+":"+Class.class.getName()+":"+describe(new PlainType());
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeReflectTypeNames", source)
	t.Logf("JVM Type names oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestTypeNamesOracle(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
