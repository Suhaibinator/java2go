#!/usr/bin/env python3
"""Apply reviewed mechanical candidates only to an isolated runtime snapshot."""
from pathlib import Path
import argparse, shutil
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args();base=a.scratch.resolve()
for name in ('edits','edits_output'):
 out=base/name;out.mkdir()
 for f in ('go.mod','go.sum'):shutil.copyfile(base/'snapshot'/f,out/f)
 shutil.copytree(base/'snapshot/stdjava',out/'stdjava')
 path=out/'stdjava/stringbuilder.go';s=path.read_text()
 s=s.replace('import "fmt"','import ("fmt"; "unicode/utf16"; "strings")' if name=='edits_output' else 'import ("fmt"; "unicode/utf16")')
 s=s.replace('b.buf = append(b.buf, StringChars(v)...)','b.appendText(v)',1)
 old='''tail := append([]rune{}, b.buf[offset:]...)
	b.buf = append(b.buf[:offset], inserted...)
	b.buf = append(b.buf, tail...)'''
 new='''// Preserve standalone Go slice inputs too: they may alias builder storage
	// inside this package. Java char[] wrappers cannot share this private buffer.
	if units, ok := value.([]rune); ok { inserted = append([]rune(nil), units...) }
	oldLength := len(b.buf)
	b.buf = append(b.buf, make([]rune, len(inserted))...)
	copy(b.buf[int(offset)+len(inserted):], b.buf[int(offset):oldLength])
	copy(b.buf[offset:], inserted)'''
 assert old in s;s=s.replace(old,new)
 s+='''
// Append UTF-16 units directly into the retained builder capacity.
func (b *StringBuilder) appendText(s string) {
 for _, unit := range s {
  if unit > 0xffff { high, low := utf16.EncodeRune(unit); b.buf = append(b.buf, high, low) } else { b.buf = append(b.buf, unit) }
 }
}
'''
 if name=='edits_output':
  old='return StringFromChars(b.buf)'
  new='''var out strings.Builder
	out.Grow(len(b.buf))
	for i := 0; i < len(b.buf); i++ {
	 unit := b.buf[i]
	 if utf16.IsSurrogate(unit) {
	  if unit >= 0xdc00 || i+1 == len(b.buf) || b.buf[i+1] < 0xdc00 || b.buf[i+1] > 0xdfff {
	   panic(NewUnsupportedOperationException("String storage does not yet preserve isolated UTF-16 surrogates"))
	  }
	  unit = utf16.DecodeRune(unit, b.buf[i+1]); i++
	 }
	 out.WriteRune(unit)
	}
	return out.String()'''
  assert old in s;s=s.replace(old,new)
 path.write_text(s)
 print(out)
