// General scalar normalization. See UPSTREAM.md and LICENSE for data provenance.
package normalization15

import "sort"

// maxNonStarters is required only by unused copied table metadata. The engine
// never limits combining sequence length or inserts a Grapheme Joiner.
const maxNonStarters = 30

type Form uint8
const (
    NFC Form = iota
    NFD
    NFKC
    NFKD
)
func (f Form) compatibility() bool {return f==NFKC||f==NFKD}
func (f Form) composing() bool {return f==NFC||f==NFKC}
func (f Form) valid() {if f>NFKD {panic("invalid internal normalization form")}}
type character struct {value rune; ccc uint8}

// Normalize accepts Unicode scalar values. Java UTF16 surrogate preservation
// belongs to the outer canonical boundary and never becomes UTF8 replacement.
func Normalize(input []rune, form Form) []rune {
    form.valid()
    decomposed:=make([]character,0,len(input))
    var decompose func(rune)
    decompose=func(r rune) {
        if r<0||r>0x10ffff||(r>=0xd800&&r<=0xdfff) {panic("non-scalar normalization input")}
        if r>=0xac00&&r<0xd7a4 {
            syllable:=r-0xac00
            decompose(0x1100+syllable/588)
            decompose(0x1161+(syllable%588)/28)
            if trailing:=syllable%28;trailing!=0 {decompose(0x11a7+trailing)}
            return
        }
        p:=properties(r,form.compatibility())
        if data:=p.Decomposition();len(data)!=0 {
            for _,part:=range string(data) {decompose(part)}
            return
        }
        decomposed=append(decomposed,character{r,p.CCC()})
    }
    for _,r:=range input {decompose(r)}
    // Sort every nonstarter run stably, including the run before the first
    // starter. A literal CGJ has CCC0 and therefore retains its barrier role.
    start:=0
    order:=func(end int) {
        run:=decomposed[start:end]
        sort.SliceStable(run,func(i,k int)bool{return run[i].ccc<run[k].ccc})
    }
    for i,p:=range decomposed {if p.ccc==0 {order(i);start=i+1}}
    order(len(decomposed))
    if !form.composing() {
        out:=make([]rune,len(decomposed));for i,p:=range decomposed {out[i]=p.value};return out
    }
    recompMapOnce.Do(buildRecompMap)
    composed:=make([]character,0,len(decomposed));starter:=-1;var previousCCC uint8
    for _,p:=range decomposed {
        if starter>=0&&(previousCCC==0||previousCCC<p.ccc) {
            if merged:=composePair(composed[starter].value,p.value);merged!=0 {
                composed[starter].value=merged
                // A consumed mark does not block a later composition.
                continue
            }
        }
        composed=append(composed,p)
        if p.ccc==0 {starter=len(composed)-1}
        previousCCC=p.ccc
    }
    out:=make([]rune,len(composed));for i,p:=range composed {out[i]=p.value};return out
}
func composePair(a,b rune) rune {
    if a>=0x1100&&a<0x1113&&b>=0x1161&&b<0x1176 {return 0xac00+(a-0x1100)*588+(b-0x1161)*28}
    if a>=0xac00&&a<0xd7a4&&(a-0xac00)%28==0&&b>0x11a7&&b<0x11c3 {return a+b-0x11a7}
    return combine(a,b)
}

// QuickCheckYes is the proposed reference-preserving fastpath predicate. A
// composing-form Maybe is deliberately not promoted to Yes by content equality.
// Identity is distinct from content equality for composing-form Maybe inputs.
func QuickCheckYes(input []rune,form Form) bool {
    form.valid();var lastCCC uint8
    for _,r:=range input {
        p:=properties(r,form.compatibility());ccc:=p.CCC()
        if ccc!=0&&lastCCC>ccc {return false}
        if form.composing() {if !p.isYesC() {return false}} else if !p.isYesD() {return false}
        lastCCC=ccc
    }
    return true
}
