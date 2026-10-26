package probe

import (
    "math"
    "testing"
    sj "github.com/NickyBoy89/java2go/stdjava"
)
var sink int32

//go:noinline
func forcedCall(a *sj.PrimitiveArray[int32], i int32, v int32) int32 {
    return sj.PrimitiveArrayAssign(a, i, v)
}

func BenchmarkRuntimeAssign(b *testing.B) {
    a := sj.NewPrimitiveArray[int32](1024, sj.PrimitiveIntTypeID)
    b.ReportAllocs(); b.ResetTimer()
    var value int32
    for i:=0; i<b.N; i++ { value = sj.PrimitiveArrayAssign(a, int32(i&1023), value+1) }
    b.StopTimer(); sink = value + a.Elements[0]
}
func BenchmarkForcedCall(b *testing.B) {
    a := sj.NewPrimitiveArray[int32](1024, sj.PrimitiveIntTypeID)
    b.ReportAllocs(); b.ResetTimer()
    var value int32
    for i:=0; i<b.N; i++ { value = forcedCall(a, int32(i&1023), value+1) }
    b.StopTimer(); sink = value + a.Elements[0]
}
func BenchmarkCanonicalFill(b *testing.B) {
    a := sj.NewPrimitiveArray[int32](28, sj.PrimitiveIntTypeID)
    b.ReportAllocs(); b.ResetTimer()
    for n:=0; n<b.N; n++ {
        for i:=int32(0); i<28; i++ { sj.PrimitiveArrayAssign(a, i, int32(n)+i) }
        sink += a.Elements[n%28]
    }
}
func TestWideIndices(t *testing.T) {
    a:=sj.NewPrimitiveArray[int32](1,sj.PrimitiveIntTypeID)
    cases:=[]struct{name string; run func()}{
        {"int64-min",func(){sj.PrimitiveArrayAssign(a,int64(math.MinInt64),int32(2))}},
        {"int64-max",func(){sj.PrimitiveArrayAssign(a,int64(math.MaxInt64),int32(2))}},
        {"uint16-max",func(){sj.PrimitiveArrayAssign(a,uint16(math.MaxUint16),int32(2))}},
        {"int8-min",func(){sj.PrimitiveArrayAssign(a,int8(math.MinInt8),int32(2))}},
    }
    for _,tc:=range cases { t.Run(tc.name,func(t *testing.T){
        defer func(){ r:=recover(); if r==nil {t.Fatal("missing exception")}; e,ok:=r.(interface{ThrowableTypeName() string}); if !ok||e.ThrowableTypeName()!="ArrayIndexOutOfBoundsException" {t.Fatalf("wrong exception: %T",r)}; if a.Elements[0]!=0 {t.Fatal("store occurred")} }()
        tc.run()
    }) }
    if sj.PrimitiveArrayAssign(a,uint16(0),int32(7))!=7 || a.Elements[0]!=7 {t.Fatal("valid unsigned index failed")}
}
