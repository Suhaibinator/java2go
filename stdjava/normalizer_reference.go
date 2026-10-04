// Canonical java.text.Normalizer boundary preserving Java UTF16 and execution.
package stdjava

import (
    "unicode/utf16"
    "github.com/NickyBoy89/java2go/stdjava/internal/normalization15"
)

func JavaNormalizerNormalize(execution *Execution,value any,form *NormalizerForm) *JavaString {
    ReferenceRequireNonNull(value)
    snapshot:=JavaStringValueOfExecution(execution,value)
    ReferenceRequireNonNull(form) // Java invokes source.toString before testing form.
    RequireJavaString(snapshot)
    units,quickYes:=normalizeJavaUTF16(snapshot.units,normalization15.Form(form.form))
    if quickYes {return snapshot}
    return NewJavaStringUTF16(units)
}
func JavaNormalizerIsNormalized(execution *Execution,value any,form *NormalizerForm) bool {
    ReferenceRequireNonNull(value)
    snapshot:=JavaStringValueOfExecution(execution,value)
    ReferenceRequireNonNull(form)
    RequireJavaString(snapshot)
    units,_:=normalizeJavaUTF16(snapshot.units,normalization15.Form(form.form))
    if len(units)!=len(snapshot.units) {return false}
    for i,u:=range units {if u!=snapshot.units[i] {return false}}
    return true
}
func normalizeJavaUTF16(input []uint16,form normalization15.Form) ([]uint16,bool) {
    out:=make([]uint16,0,len(input));span:=make([]rune,0,len(input));quickYes:=true
    flush:=func(){
        if len(span)==0 {return}
        quickYes=normalization15.QuickCheckYes(span,form)&&quickYes
        out=append(out,utf16.Encode(normalization15.Normalize(span,form))...)
        span=span[:0]
    }
    for i:=0;i<len(input);i++ {
        u:=input[i]
        if u>=0xd800&&u<=0xdbff&&i+1<len(input)&&input[i+1]>=0xdc00&&input[i+1]<=0xdfff {
            span=append(span,utf16.DecodeRune(rune(u),rune(input[i+1])));i++;continue
        }
        if u>=0xd800&&u<=0xdfff {flush();out=append(out,u);continue}
        span=append(span,rune(u))
    }
    flush();return out,quickYes
}
