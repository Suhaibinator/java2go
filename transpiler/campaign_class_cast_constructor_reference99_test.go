package transpiler
import("strings";"testing")
func TestCampaignClassCastConstructorCanonical99(t *testing.T){
 source:=`public class ClassCastConstructor99 {static ClassCastException empty(){return new ClassCastException();}static ClassCastException literal(){return new ClassCastException("probe");}static ClassCastException absent(){return new ClassCastException(null);}static ClassCastException typed(){return new ClassCastException((String)null);}static <T extends String> ClassCastException bounded(T text){return new ClassCastException(text);}}`
 generated:=renderGoFileFromJava(t,source)
 if strings.Contains(generated,"stdjava.NewClassCastException(")||strings.Count(generated,"stdjava.NewJavaClassCastExceptionMessage(")!=5{t.Fatalf("ClassCastException must select canonical String constructor ABI:\n%s",generated)}
 for _,source:=range []string{`class ClassCastException {ClassCastException(String value){}} public class LocalShadow99 {static ClassCastException make(String value){return new ClassCastException(value);}}`,`package custom; class ClassCastException {ClassCastException(String value){}} public class QualifiedShadow99 {static custom.ClassCastException make(String value){return new custom.ClassCastException(value);}}`}{generated:=renderGoFileFromJava(t,source);if strings.Contains(generated,"stdjava.NewJavaClassCastExceptionMessage(")||strings.Contains(generated,"stdjava.NewClassCastException("){t.Fatalf("source constructor borrowed JDK owner: %s",generated)}}
}
