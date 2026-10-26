package transpiler

import "testing"

func TestRawInnerCanonicalConstructorUninferredSlots(t *testing.T) {
	t.Run("no-own-argument", func(t *testing.T) {
		assertGeneratedLocalConstructorResult(t, `
public class RawEmptyConstructorProbe {
 static class Outer<T> {
  T outer;
  Outer(T value) {outer=value;}
  class Inner<U> {
   U inner;
   Inner() {}
   String render(){return outer+":"+inner;}
  }
 }
 @SuppressWarnings({"rawtypes","unchecked"})
 public static String run(){
  Outer outer=new Outer("initial");
  Outer.Inner inner=outer.new Inner();
  outer.outer=42;inner.inner=7;
  return inner.render();
 }
}
`, "42:7")
	})
	t.Run("typed-alias", func(t *testing.T) {
		assertGeneratedLocalConstructorResult(t, `
public class RawTypedAliasProbe {
 static class Outer<T> {
  T outer;
  Outer(T value) {outer=value;}
  class Inner<U> {
   U inner;
   Inner(U value) {inner=value;}
   String render(){return outer+":"+inner;}
  }
 }
 @SuppressWarnings({"rawtypes","unchecked"})
 public static String run(){
  Outer<String> typed=new Outer<String>("initial");
  Outer raw=typed;
  Outer.Inner inner=raw.new Inner("initial-inner");
  raw.outer=42;inner.inner=7;
  return inner.render()+":"+(raw==typed);
 }
}
`, "42:7:true")
	})
}
