package stdjava
import("slices";"testing")
func TestJavaClassCastConstructorReference99(t *testing.T){
 execution:=NewExecution()
 for _,message:=range []*JavaString{nil,JavaStringLiteralUTF16(nil),NewJavaStringUTF16([]uint16{'p',0xd800,0,0xdc00})}{
 value:=NewJavaClassCastExceptionMessage(message)
 if JavaThrowableMessageExecution(execution,value)!=message||GetCause(value)!=nil{t.Fatal("constructor lost message reference/null or cause state")}
 cause:=NewJavaRuntimeExceptionMessage(nil)
 if !JavaReferenceEqual(ThrowableInitCauseExecution(execution,value,cause),value)||!JavaReferenceEqual(GetCause(value),cause){t.Fatal("constructor lost initCause receiver/cause identity")}
 expectBoxedException(t,"IllegalStateException",func(){_ = ThrowableInitCauseExecution(execution,value,nil)})
 text:=JavaThrowableToStringExecution(execution,value);want:=JavaStringFromHostUTF8("java.lang.ClassCastException").UTF16Copy();if message!=nil{want=append(want,':',' ');want=append(want,message.UTF16Copy()...)}
 if !slices.Equal(text.UTF16Copy(),want){t.Fatal("toString changed null, empty, or UTF16 message")}
 }
 if NewClassCastException("native").Message()!="native"{t.Fatal("native ABI changed")}
}
