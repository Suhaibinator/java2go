#!/usr/bin/env python3
"""Create isolated runtime copies and differential Java/Go oracle programs."""
from pathlib import Path
import argparse, json, shutil
p=argparse.ArgumentParser();p.add_argument('scratch',type=Path);a=p.parse_args()
root=Path(__file__).resolve().parents[3];own=Path(__file__).resolve().parent;base=a.scratch.resolve()
if base.exists() and any(base.iterdir()):raise SystemExit('scratch must be empty')
base.mkdir(parents=True,exist_ok=True);frozen=base/'snapshot';frozen.mkdir()
for name in ('go.mod','go.sum'):shutil.copyfile(root/name,frozen/name)
shutil.copytree(root/'stdjava',frozen/'stdjava')
sources=['job-17|payload','x'*4096+'|data','','abc|abc','é|東京|é','a😀b😀a','𐀀\ue000\U0010ffff😀','🧪|naïve/雪|🧪','a\x00b\uffff','aaaaa','😀😀','|']
needles=['','a','aa','|','é','東京','😀','🧪','𐀀','\ue000','\uffff','\x00','😀a','a😀','not-found']
points=[-2147483648,-1,0,32,97,124,127,128,0xe9,0xd7ff,0xd800,0xd83d,0xde00,0xdbff,0xdfff,0xe000,0xfffd,0xffff,0x10000,0x1f600,0x1f9ea,0x10ffff,0x110000,2147483647]
starts=[-2147483648,-100,-1,0,1,2,3,4,5,6,7,8,9,10,100,2147483647]
(base/'inputs.json').write_text(json.dumps({'strings':sources,'needles':needles,'codepoints':points,'starts':starts},ensure_ascii=True,indent=2)+'\n')
java='''public class Oracle {
 static String[] sources={SOURCES}; static String[] needles={NEEDLES};
 static int[] points={POINTS}; static int[] starts={STARTS};
 public static void main(String[] args) {
  int id=0;
  for(String s:sources){
   for(String n:needles){System.out.println((id++)+":"+s.indexOf(n)+":"+s.lastIndexOf(n));for(int f:starts)System.out.println((id++)+":"+s.indexOf(n,f)+":"+s.lastIndexOf(n,f));}
   for(int n:points){System.out.println((id++)+":"+s.indexOf(n)+":"+s.lastIndexOf(n));for(int f:starts)System.out.println((id++)+":"+s.indexOf(n,f)+":"+s.lastIndexOf(n,f));}
  }
 }
}
'''
for key,val in [('SOURCES',','.join(json.dumps(x,ensure_ascii=True) for x in sources)),('NEEDLES',','.join(json.dumps(x,ensure_ascii=True) for x in needles)),('POINTS',','.join(map(str,points))),('STARTS',','.join(map(str,starts)))]:java=java.replace(key,val)
java=java.replace(' public static void main(String[] args) {', ' static String failure(Runnable f) {try {f.run();return "none";}catch(RuntimeException e){return e.getClass().getSimpleName();}}\n public static void main(String[] args) {')
java=java.replace('\n }\n}\n', '\n  System.out.println("null-receiver:"+failure(()->((String)null).indexOf("x")));\n  System.out.println("null-string-needle-negative:"+failure(()->"abc".lastIndexOf((String)null,-1)));\n  System.out.println("null-start:"+failure(()->"abc".indexOf("x",(Integer)null)));\n  System.out.println("null-receiver-invalid-point:"+failure(()->((String)null).indexOf(-1)));\n }\n}\n')
(base/'Oracle.java').write_text(java)
go='''package main
import("fmt";sj "github.com/NickyBoy89/java2go/stdjava")
func main(){sources:=[]string{SOURCES};needles:=[]string{NEEDLES};points:=[]int32{POINTS};starts:=[]int32{STARTS};id:=0
for _,s:=range sources {for _,n:=range needles {fmt.Printf("%d:%d:%d\\n",id,sj.StringIndexOf(s,n),sj.StringLastIndexOf(s,n));id++;for _,f:=range starts{fmt.Printf("%d:%d:%d\\n",id,sj.StringIndexOf(s,n,f),sj.StringLastIndexOf(s,n,f));id++}}
for _,n:=range points {fmt.Printf("%d:%d:%d\\n",id,sj.StringIndexOf(s,n),sj.StringLastIndexOf(s,n));id++;for _,f:=range starts{fmt.Printf("%d:%d:%d\\n",id,sj.StringIndexOf(s,n,f),sj.StringLastIndexOf(s,n,f));id++}}}
}
'''
for key,val in [('SOURCES',','.join(json.dumps(x,ensure_ascii=False) for x in sources)),('NEEDLES',','.join(json.dumps(x,ensure_ascii=False) for x in needles)),('POINTS',','.join(map(str,points))),('STARTS',','.join(map(str,starts)))]:go=go.replace(key,val)
for variant in ('baseline','candidate'):
 out=base/variant;shutil.copytree(frozen,out);(out/'oracle').mkdir();(out/'oracle/main.go').write_text(go)
 if variant=='candidate':
  search=out/'stdjava/string_search.go';s=search.read_text().replace('func StringIndexOf(', 'func stringIndexOfMaterialized(').replace('func StringLastIndexOf(', 'func stringLastIndexOfMaterialized(');search.write_text(s)
  shutil.copyfile(own/'candidate.go.txt',out/'stdjava/string_search_candidate.go')
 (out/'probe').mkdir();shutil.copyfile(own/'allocations.go.txt',out/'probe/main.go')
print(base)
