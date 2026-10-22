package transpiler
import (
 "testing"
 "github.com/NickyBoy89/java2go/parsing"
 "github.com/NickyBoy89/java2go/symbol"
)
func TestCI164SourceClassOriginBindingsUnrelatedEnclosingOwner(t *testing.T) {
 const source = `
public class SpecializedInnerOverrideProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            U inner;
            Inner(U inner) { this.inner = inner; }
            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }
    static final class ExactOuter extends Outer<Item> {
        ExactOuter(Item value) { super(value); }
        class ExactInner extends Inner<Item> {
            ExactInner(Item value) { super(value); }
            @Override int mutate(Item nextOuter, Item nextInner) {
                return nextOuter.number() * 10 + nextInner.number();
            }
        }
    }
    public static String run() {
        ExactOuter outer = new ExactOuter(new Item(1));
        ExactOuter.ExactInner inner = outer.new ExactInner(new Item(2));
        return String.valueOf(inner.mutate(new Item(3), new Item(4)));
    }
}
`
 previous:=symbol.GlobalScope
 symbol.GlobalScope=&symbol.GlobalSymbols{Packages:map[string]*symbol.PackageScope{}}
 t.Cleanup(func(){symbol.GlobalScope=previous})
 file:=parsing.SourceFile{Name:"SpecializedInnerOverrideProbe.java",Source:[]byte(source)}
 if err:=file.ParseAST();err!=nil {t.Fatal(err)}
 f:=file.ParseSymbols();symbol.AddSymbolsToPackage(f)
 var outer,inner *symbol.ClassScope
 var visit func(*symbol.ClassScope)
 visit=func(s *symbol.ClassScope){
  if s==nil||s.Class==nil{return}
  t.Logf("scope=%s go=%s super=%s inner=%t enclosing=%p params=%d own=%d",s.Class.OriginalName,s.Class.Name,s.Superclass,s.IsInner,s.Enclosing,len(s.TypeParameters),len(s.OwnTypeParameters()))
  if s.Class.OriginalName=="Outer" {outer=s}
  if s.Class.OriginalName=="ExactInner" {inner=s}
  for _, child:=range s.Subclasses {visit(child)}
 }
 for _,s:=range f.TopLevelClasses {visit(s)}
 if outer==nil||inner==nil {t.Fatal("original declarations missing")}
 ctx:=Ctx{currentFile:f,currentClass:inner}
 if classScopeDescendsFrom(inner,outer,ctx) {t.Fatal("lexical enclosing owner unexpectedly became Java superclass")}
 t.Log("PROVED: actual ExactInner is not a Java subtype of original Outer; querying unchanged sourceClassOriginBindings")
 if got:=sourceClassOriginBindings(inner,classParameterTypeOrigins(inner),outer,ctx);len(got)!=0 {t.Fatalf("non-ancestor supplied bindings: %v",got)}
}
