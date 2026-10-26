package stdjava
import("os";"strconv";"strings";"testing")
// The distinct additive oracle is frozen actual JDK21 stdout; original150 remains unchanged.
func TestCanonicalStringSplitPropertyCaseOriginalJDK21(t *testing.T){
 for _,seed:=range []string{"property-case"}{
  t.Run(seed,func(t *testing.T){
   raw,err:=os.ReadFile("testdata/canonical_split_property_v1/"+seed+".stdout");if err!=nil{t.Fatal("independent oracle missing:",err)}
   count:=0
   for _,line:=range strings.Split(strings.TrimSpace(string(raw)),"\n"){
    f:=strings.Split(line,"\t");if f[0]=="END"{continue};if len(f)<8{t.Fatalf("bad independent oracle row %q",line)};count++
    t.Run(f[0],func(t *testing.T){
     input,regex:=splitProbeUnits(f[1]),splitProbeUnits(f[2]);limit64,err:=strconv.ParseInt(f[3],10,32);if err!=nil{t.Fatal(err)}
     var got *ReferenceArray;var caught any
     call:=func()*ReferenceArray{if f[4]=="true"{return JavaStringSplitArray(input,regex)};return JavaStringSplitArray(input,regex,int32(limit64))}
     func(){defer func(){caught=recover()}();got=call()}()
     if f[5]=="EX"{
      thrown,ok:=caught.(Throwable);if !ok{t.Fatalf("JDK %s; Go panic %#v",f[6],caught)}
      wanted:=strings.TrimPrefix(f[6],"java.lang.");wanted=strings.TrimPrefix(wanted,"java.util.regex.")
      if thrown.ThrowableTypeName()!=wanted{t.Fatalf("JDK %s; Go %s",wanted,thrown.ThrowableTypeName())}
      // Pattern syntax detail/accessors are a separately stated runtime gap.
      if wanted=="PatternSyntaxException"{x:=strings.SplitN(f[7],"/",4);if len(x)!=4{t.Fatal("syntax oracle")};wantMessage:=splitProbeUnits(x[3]);if !wantMessage.Equals(JavaThrowableMessageExecution(nil,caught)){t.Fatalf("syntax UTF16 message differs: %v",caught)};syntax,ok:=caught.(interface{GetIndex() int32;GetDescription() *JavaString;GetPattern() *JavaString});if !ok{t.Fatal("syntax accessors unavailable")};idx,err:=strconv.ParseInt(x[0],10,32);if err!=nil{t.Fatal(err)};if syntax.GetIndex()!=int32(idx)||!splitProbeUnits(x[1]).Equals(syntax.GetDescription())||!splitProbeUnits(x[2]).Equals(syntax.GetPattern()){t.Fatal("syntax accessor mismatch")};if !CaughtAs(caught,"IllegalArgumentException")||!CaughtAs(caught,"RuntimeException"){t.Fatal("syntax exception ancestry")}}
      return
     }
     if caught!=nil{t.Fatalf("JDK split succeeded; Go panic %v",caught)}
     if got==nil||got.componentType!=StringTypeID{t.Fatal("must return canonical String[]")}
     n,err:=strconv.Atoi(f[6]);if err!=nil{t.Fatal(err)};if len(got.elements)!=n{t.Fatalf("length %d want %d",len(got.elements),n)}
     values:=[]string{};for _,v:=range got.elements{s,ok:=v.(*JavaString);if !ok||s==nil{t.Fatalf("noncanonical element %#v",v)};values=append(values,splitProbeEncode(s))};encoded:="-";if len(values)>0{encoded=strings.Join(values,"/")};if encoded!=f[7]{t.Fatalf("units %s want %s",encoded,f[7])}
     identity:=len(got.elements)==1&&got.elements[0]==input;if strconv.FormatBool(identity)!=f[8]{t.Fatalf("receiver identity %v want %s",identity,f[8])}
     again:=call();if got==again{t.Fatal("String[] allocation reused")};if len(got.elements)>0{saved:=again.elements[0];got.elements[0]=nil;if again.elements[0]!=saved{t.Fatal("array storage shared")}}
    })
   }
   if count!=60{t.Fatal("oracle closure incomplete",count)}
  })
 }
}
