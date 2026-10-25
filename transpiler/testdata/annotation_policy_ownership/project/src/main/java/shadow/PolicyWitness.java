package shadow;
enum Policy {RUNTIME}
@interface Retention {Policy value();}
@Retention(Policy.RUNTIME) @interface Hidden {}
@interface Inherited {}
@java.lang.annotation.Retention(java.lang.annotation.RetentionPolicy.RUNTIME) @Inherited @interface Mark {}
@Mark class Parent {}
class Child extends Parent {}
@Hidden class Model {}
@java.lang.annotation.Retention(java.lang.annotation.RetentionPolicy.RUNTIME) @java.lang.annotation.Target(java.lang.annotation.ElementType.TYPE_USE) @interface UseOnly {}
@java.lang.annotation.Retention(java.lang.annotation.RetentionPolicy.RUNTIME) @java.lang.annotation.Target({java.lang.annotation.ElementType.FIELD,java.lang.annotation.ElementType.TYPE_USE}) @interface Mixed {}
@UseOnly class TypeModel {@UseOnly String field; @Mixed String both; @UseOnly String read(){return field;}}
enum OwnElement {FIELD}
@interface Target {OwnElement value();}
@java.lang.annotation.Retention(java.lang.annotation.RetentionPolicy.RUNTIME) @Target(OwnElement.FIELD) @interface AnyDeclaration {}
@AnyDeclaration class TargetModel {}
public class PolicyWitness {
 public static void main(String[] args) throws Exception {
  System.out.println("retention="+(Model.class.getAnnotation(Hidden.class)==null)+":"+Model.class.isAnnotationPresent(Hidden.class));
  System.out.println("inherited="+(Child.class.getAnnotation(Mark.class)==null)+":"+Child.class.isAnnotationPresent(Mark.class));
  System.out.println("type-use="+(TypeModel.class.getDeclaredField("field").getAnnotation(UseOnly.class)==null)+":"+(TypeModel.class.getDeclaredMethod("read").getAnnotation(UseOnly.class)==null));
  System.out.println("mixed="+(TypeModel.class.getDeclaredField("both").getAnnotation(Mixed.class)!=null));
  System.out.println("type-declaration="+(TypeModel.class.getAnnotation(UseOnly.class)!=null));
  System.out.println("target-owner="+(TargetModel.class.getAnnotation(AnyDeclaration.class)!=null));
 }
}
