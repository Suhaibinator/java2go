package stdjava

import (
 "math/big"
 "testing"
)

func TestCampaignTimSortCheckedArithmetic(t *testing.T) {
 hostMax:=int(^uint(0)>>1)
 values:=[]int{1,2,3,15,16,17,255,256,257,1<<29-1,1<<29,1<<30-1,1<<30,1<<31-1,hostMax-1,hostMax}
 for _,minimum:=range values {
  for _,length:=range []int{minimum,hostMax} {
   next:=new(big.Int).Lsh(big.NewInt(1),uint(big.NewInt(int64(minimum)).BitLen()))
   expected:=minimum
   if next.Cmp(big.NewInt(1<<31-1))<=0 {expected=int(next.Int64());if expected>length/2{expected=length/2}}
   if actual:=javaTimTempCapacity(minimum,length);actual!=expected{t.Fatalf("capacity(%d,%d)=%d want %d",minimum,length,actual,expected)}
  }
 }
 for _,offset:=range []int{0,1,3,7,1<<29-1,1<<30-1,1<<30,1<<31-1}{
  next:=new(big.Int).Add(new(big.Int).Lsh(big.NewInt(int64(offset)),1),big.NewInt(1));expected:=int(next.Int64());if next.Cmp(big.NewInt(1<<31-1))>0{expected=1<<31-1}
  if actual:=javaTimNextOffset(offset,1<<31-1);actual!=expected{t.Fatalf("offset %d=%d want %d",offset,actual,expected)}
 }
}

func TestCampaignTimSortGallopBoundaries(t *testing.T) {
 c:=func(a,b int)int32{if a<b{return -1};if a>b{return 1};return 0}
 for _,a:=range [][]int{{0},{0,0},{0,0,1,1,1,4,7,7,9},{-8,-4,-3,0,1,2,3,9,12,18,22,30,31,31,31,31,32}}{
  for key:=-10;key<=34;key++{left,right:=0,0;for _,v:=range a{if v<key{left++};if v<=key{right++}};for hint:=range a{
   if n:=javaTimGallopLeft(key,a,0,len(a),hint,c);n!=left{t.Fatalf("left key=%d hint=%d got=%d want=%d",key,hint,n,left)}
   if n:=javaTimGallopRight(key,a,0,len(a),hint,c);n!=right{t.Fatalf("right key=%d hint=%d got=%d want=%d",key,hint,n,right)}
  }}
 }
}

func TestCampaignTimSortCanonicalViolation(t *testing.T) {
 p:=sortNullPanic(javaTimContractViolation)
 failure,ok:=p.(IllegalArgumentException);if !ok{t.Fatalf("unexpected throwable %T",p)}
 literal:=JavaStringLiteralUTF16([]uint16{'C','o','m','p','a','r','i','s','o','n',' ','m','e','t','h','o','d',' ','v','i','o','l','a','t','e','s',' ','i','t','s',' ','g','e','n','e','r','a','l',' ','c','o','n','t','r','a','c','t','!'})
 if failure.state.javaMessage!=literal||!failure.state.canonicalMessage||failure.Message()!="Comparison method violates its general contract!"{t.Fatal("canonical literal message reference lost")}
}
