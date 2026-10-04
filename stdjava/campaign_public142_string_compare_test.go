package stdjava

import (
	"slices"
	"testing"
)

var campaign142ComparisonResult int32

func campaign142StringPairs() []struct{ left, right []uint16; want int32 } {
	return []struct{ left, right []uint16; want int32 }{
		{nil, nil, 0}, {nil, []uint16{'a'}, -1}, {[]uint16{'a'}, nil, 1},
		{[]uint16{'a'}, []uint16{'a'}, 0}, {[]uint16{'a'}, []uint16{'z'}, -25},
		{[]uint16{'z'}, []uint16{'a'}, 25}, {[]uint16{'a'}, []uint16{'a','b'}, -1},
		{[]uint16{'a','b'}, []uint16{'a'}, 1}, {[]uint16{0}, []uint16{'a'}, -97},
		{[]uint16{'a',0}, []uint16{'a',1}, -1}, {[]uint16{0xffff}, []uint16{0}, 65535},
		{[]uint16{0}, []uint16{0xffff}, -65535}, {[]uint16{0xd800}, []uint16{0xdc00}, -1024},
		{[]uint16{0xdc00}, []uint16{0xd800}, 1024}, {[]uint16{0xd83d,0xde00}, []uint16{0xd83d,0xde01}, -1},
		{[]uint16{0xd83d,0xde00}, []uint16{0xe000}, -1987},
		{[]uint16{'Q',0xdfff}, []uint16{'Q',0xd800}, 2047},
		{[]uint16{0x00e9}, []uint16{0x0065,0x0301}, 132},
		{[]uint16{'a',0xd800,0}, []uint16{'a',0xd800,0,0}, -1},
		{[]uint16{'a',0xdfff,'x'}, []uint16{'a',0xdfff,'z'}, -2},
	}
}

func TestCampaignPublic142StringCompareExactValues(t *testing.T) {
	execution := NewExecution()
	for i, pair := range campaign142StringPairs() {
		a,b := NewJavaStringUTF16(pair.left), NewJavaStringUTF16(pair.right)
		got := javaCompareValuesExecution(execution,a,b)
		old,ok := compareViaCompareToExecution(execution,a,b)
		if !ok || got!=pair.want || old!=pair.want { t.Fatalf("pair%d current=%d reflection=%d/%v want=%d",i,got,old,ok,pair.want) }
		if a==b || a.cachedHash.Load()!=0 || b.cachedHash.Load()!=0 || !slices.Equal(a.units,pair.left) || !slices.Equal(b.units,pair.right) { t.Fatal("comparison changed identity, hash or units") }
		if NaturalOrder[*JavaString](execution)(a,b)!=pair.want || ReverseOrder[*JavaString](execution)(b,a)!=pair.want { t.Fatal("natural/reverse exact result changed") }
	}
}

func TestCampaignPublic142StringCompareNoResultAllocations(t *testing.T) {
	a,b := NewJavaStringUTF16([]uint16{'a',0xd800,0}), NewJavaStringUTF16([]uint16{'z',0xdc00,0})
	count := testing.AllocsPerRun(1000,func(){ campaign142ComparisonResult=javaCompareValuesExecution(nil,a,b) })
	t.Logf("canonical nonnull String comparison allocations/call=%g",count)
	if count!=0 { t.Fatalf("expected no allocation for existing UTF16 comparison result, got %g",count) }
}

type campaign142Comparable struct { execution *Execution; calls int; value int32 }
func (value *campaign142Comparable) CompareTo(other *campaign142Comparable) int32 { panic("ordinary callback bypassed execution") }
func (value *campaign142Comparable) CompareToJava2goExecution1(execution *Execution, other *campaign142Comparable) int32 {
	value.execution=execution;value.calls++
	if other==nil { panic(NewNullPointerException("source-null")) }
	if other.value==99 { panic(NewIllegalStateException("source-stop")) }
	return value.value-other.value
}
func campaign142Failure(action func()) (failure any) {
	defer func(){ failure=recover() }();action();return nil
}
func TestCampaignPublic142StringCompareFallbacks(t *testing.T) {
	a:=NewJavaStringUTF16([]uint16{'a'})
	for _,right:=range []any{nil,(*JavaString)(nil),(*Integer)(nil)} {
		failure:=campaign142Failure(func(){ javaCompareValuesExecution(nil,a,right) })
		original:=campaign142Failure(func(){ compareViaCompareToExecution(nil,a,right) })
		if !CaughtAs(failure,"NullPointerException") || !CaughtAs(original,"NullPointerException") || !JavaThrowableMessageDefault(failure).Equals(JavaThrowableMessageDefault(original)) { t.Fatal("null right changed original failure") }
	}
	for _,right:=range []any{"a",NewInteger(7),struct{}{},[]uint16{'a'}} {
		failure:=campaign142Failure(func(){ javaCompareValuesExecution(nil,a,right) })
		if !CaughtAs(failure,"ClassCastException") || !slices.Equal(JavaThrowableMessageDefault(failure).UTF16Copy(),NewJavaStringUTF16([]uint16{'v','a','l','u','e',' ','i','s',' ','n','o','t',' ','n','a','t','u','r','a','l','l','y',' ','c','o','m','p','a','r','a','b','l','e'}).units) { t.Fatal("wrong type fallback changed") }
	}
	if !CaughtAs(campaign142Failure(func(){ javaCompareValuesExecution(nil,(*JavaString)(nil),a) }),"NullPointerException") { t.Fatal("null left accepted") }
	execution:=NewExecution();left,right:=&campaign142Comparable{value:1},&campaign142Comparable{value:7}
	if javaCompareValuesExecution(execution,left,right)!=-6 || left.execution!=execution || left.calls!=1 { t.Fatal("source callback/Execution changed") }
	if !CaughtAs(campaign142Failure(func(){ javaCompareValuesExecution(execution,left,&campaign142Comparable{value:99}) }),"IllegalStateException") || left.calls!=2 || left.execution!=execution { t.Fatal("abrupt source callback changed") }
	if javaCompareValuesExecution(execution,NewInteger(1),NewInteger(7))>=0 { t.Fatal("boxed ordering changed") }
}

func BenchmarkCampaignPublic142CanonicalStringCompare(b *testing.B) {
	a,c := NewJavaStringUTF16([]uint16{'a',0xd800,0}), NewJavaStringUTF16([]uint16{'z',0xdc00,0})
	b.ReportAllocs();b.ResetTimer()
	for i:=0;i<b.N;i++ { campaign142ComparisonResult=javaCompareValuesExecution(nil,a,c) }
}
