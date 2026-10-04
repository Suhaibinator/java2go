package stdjava
import("testing";"encoding/json";"os";"fmt";"strings";"reflect")
type campaignTimRow struct {*ObjectInfo;key int32;id int}
func campaignTimNew(k int32,id int)*campaignTimRow{r:=&campaignTimRow{key:k,id:id};r.ObjectInfo=NewObjectInfo("campaign.TimRow",func(t TypeID)any{if t=="campaign.TimRow"||t==ObjectTypeID{return r};return nil});return r}
func campaignTimID(r *campaignTimRow)int{if r==nil{return -1};return r.id}
type campaignTimOracle struct{Kind string;N,Pattern,Stop int;Header string;IDs []int;Trace string}
func TestCampaignTimSortWholeJDKTraces(t *testing.T){
 b,err:=os.ReadFile("testdata/campaign-timsort/original-traces.json");if err!=nil{t.Fatal(err)};var cases []campaignTimOracle;if err=json.Unmarshal(b,&cases);err!=nil{t.Fatal(err)}
 RegisterJavaType("campaign.TimRow",ObjectTypeID)
 for _,test:=range cases{t.Run(fmt.Sprintf("%s-%d-%d-%d",test.Kind,test.N,test.Pattern,test.Stop),func(t *testing.T){
  values:=make([]any,test.N);for i:=range values{key:=i;null:=false;if test.Kind=="boundary"{switch test.Pattern{case 1:key=test.N-i;case 2:key=1;case 3:key=(i*37+11)%29;case 4:if (i/16)%2==0{key=test.N+i};case 5:null=i%7==2}}else{switch test.Pattern{case 0:if i<test.N/2{key=test.N+i}else{key=i-test.N/2};case 1:if i<test.N*3/4{key=test.N+i}else{key=i-test.N*3/4};case 2:key=(i*37+19)%43;null=i%11==3}}
   if !null{values[i]=campaignTimNew(int32(key),i)}}
  array:=ReferenceArrayLiteral("campaign.TimRow",values...);alias:=array;descriptor:=array.JavaArrayTypeID();execution:=NewExecution();lock:=NewObject();local:=NewThreadLocal[string]();local.Set(execution,"job");var trace strings.Builder;calls:=0;marker:=NewRuntimeException("sort marker")
  p:=sortNullPanic(func(){SortArrayWith(array,Comparator[*campaignTimRow](func(a,b *campaignTimRow)int32{var guard *MonitorGuard;if test.Kind=="abrupt"{guard=MonitorEnterExecution(execution,lock);defer MonitorExitExecution(guard)};calls++;if test.Kind=="boundary"{fmt.Fprintf(&trace,"%d/%d;",campaignTimID(a),campaignTimID(b))}else{fmt.Fprintf(&trace,"%d/%d:%s:%t;",campaignTimID(a),campaignTimID(b),local.Get(execution),ThreadHoldsLockExecution(execution,lock));if calls==test.Stop{panic(marker)}};if a==nil{if b==nil{return 0};return -1};if b==nil{return 1};if a.key<b.key{return -1};if a.key>b.key{return 1};return 0}),execution)})
  header:=fmt.Sprintf("%d:%d:%d:%t",test.N,test.Pattern,calls,array==alias);if test.Kind=="abrupt"{header=fmt.Sprintf("%d:%d:%d:%d:%t:%t:%t:%t",test.N,test.Pattern,test.Stop,calls,p!=nil,p==marker,ThreadHoldsLockExecution(execution,lock),array==alias)}else if p!=nil{t.Fatalf("unexpected comparator failure %v",p)}
  ids:=make([]int,len(array.elements));for i,v:=range array.elements{if v==nil{ids[i]=-1}else{ids[i]=v.(*campaignTimRow).id}}
  if header!=test.Header||!reflect.DeepEqual(ids,test.IDs)||trace.String()!=test.Trace||array.JavaArrayTypeID()!=descriptor{t.Fatalf("JDK trace/backing mismatch header=%s want=%s trace-length=%d/%d",header,test.Header,trace.Len(),len(test.Trace))}
 })}
}
