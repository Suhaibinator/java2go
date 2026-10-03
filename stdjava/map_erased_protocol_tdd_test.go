package stdjava

import "testing"

// Native test probes only; the application oracle uses the actual JDK classes.
type nativeMapTDDRoles interface {
 JavaMap
 MapSizeJava2goExecution(*Execution) int32
 MapIsEmptyJava2goExecution(*Execution) bool
 MapContainsKeyJava2goExecution(*Execution, any) bool
 MapContainsValueJava2goExecution(*Execution, any) bool
 MapGetJava2goExecution(*Execution, any) any
 MapRemoveJava2goExecution(*Execution, any) any
 MapPutJava2goExecution(*Execution, any, any) any
 MapClearJava2goExecution(*Execution)
 MapKeySetJava2goExecution(*Execution) JavaIterable
 MapValuesJava2goExecution(*Execution) JavaIterable
}
func nativeMapTDDRolesOf(t *testing.T, value any) nativeMapTDDRoles {
 t.Helper(); roles,ok:=value.(nativeMapTDDRoles);if !ok {t.Fatalf("native Map lacks erased protocol: %T",value)};return roles
}
func nativeMapTDDPanic(t *testing.T, name string, call func()) any {
 t.Helper();var failure any;func(){defer func(){failure=recover()}();call()}();if failure==nil||!CaughtAs(failure,name){t.Fatalf("want %s, got %T %v",name,failure,failure)};return failure
}
func nativeMapTDDSentinel(name string) any {
 switch name{case "ClassCastException":failure:=NewClassCastException("sentinel");return &failure;case "NullPointerException":failure:=NewNullPointerException("sentinel");return &failure;default:failure:=NewIllegalArgumentException("sentinel");return &failure}
}
func nativeMapTDDNoPanic(t *testing.T, call func()) {
 t.Helper();var failure any;func(){defer func(){failure=recover()}();call()}();if failure!=nil{t.Fatalf("source-map copy refused: %T %v",failure,failure)}
}
func nativeMapTDDEntry(e *Execution, view JavaIterable) JavaMapEntry {
 return ObjectView[JavaMapEntry](IteratorNextExecution(e,IterableIteratorExecution(e,view)),JavaMapEntryTypeID)
}
func TestNativeMapErasedProtocolRolesAndNominal(t *testing.T) {
 for _,test:=range []struct{name string;value *Map[string,string]}{{"hash",NewMap[string,string]()},{"linked",NewMap[string,string]()},{"tree",NewTreeMap[string,string]()}} {
  t.Run(test.name,func(t *testing.T){
   e:=NewExecution();m:=test.value;p:=nativeMapTDDRolesOf(t,m)
   if !ObjectInstanceOf(m,MapTypeID)||!ObjectInstanceOf(m,AbstractMapTypeID)||!JavaReferenceEqual(ObjectView[JavaMap](m,AbstractMapTypeID),m){t.Fatal("native nominal membership or allocation identity")}
   for _,id:=range []TypeID{nativeCollectionTypeID,IterableTypeID,nativeListTypeID,JavaMapEntryTypeID}{if ObjectInstanceOf(m,id){t.Fatalf("Map gained unrelated %s membership",id)}}
   if !p.MapIsEmptyJava2goExecution(e)||p.MapSizeJava2goExecution(e)!=0{t.Fatal("empty map")}
   if p.MapPutJava2goExecution(e,"Aa","one")!=nil||p.MapGetJava2goExecution(e,"Aa")!="one"||!p.MapContainsKeyJava2goExecution(e,"Aa")||!p.MapContainsValueJava2goExecution(e,"one"){t.Fatal("raw protocol dispatch")}
   if p.MapRemoveJava2goExecution(e,"Aa")!="one"||p.MapSizeJava2goExecution(e)!=0{t.Fatal("raw removal")}
  })
 }
}
func TestNativeMapErasedProtocolRawAndLiveViews(t *testing.T) {
 e:=NewExecution();m:=NewMap[string,string]();p:=nativeMapTDDRolesOf(t,m);p.MapPutJava2goExecution(e,"Aa","initial")
 entries:=p.MapEntrySetJava2goExecution(e);keys:=p.MapKeySetJava2goExecution(e);values:=p.MapValuesJava2goExecution(e)
 if !JavaReferenceEqual(entries,p.MapEntrySetJava2goExecution(e))||!JavaReferenceEqual(keys,p.MapKeySetJava2goExecution(e))||!JavaReferenceEqual(values,p.MapValuesJava2goExecution(e)){t.Fatal("views copied")}
 entry:=nativeMapTDDEntry(e,entries);if MapEntrySetValueExecution(e,entry,"entry-write")!="initial"||p.MapGetJava2goExecution(e,"Aa")!="entry-write"{t.Fatal("entry write alias")}
 p.MapPutJava2goExecution(e,"Aa","replacement");if MapEntryGetValueExecution(e,entry)!="replacement"{t.Fatal("retained entry replacement")}
 box:=NewInteger(17);p.MapPutJava2goExecution(e,"Aa",box);if p.MapGetJava2goExecution(e,"Aa")!=box||MapEntryGetValueExecution(e,entry)!=box{t.Fatal("typed backend projected erased value")}
 nativeMapTDDPanic(t,"ClassCastException",func(){ObjectView[*JavaString](p.MapGetJava2goExecution(e,"Aa"),StringTypeID)})
 effects:=0;var old any;nativeMapTDDPanic(t,"ClassCastException",func(){value:=func()any{effects++;return "completed"}();old=p.MapPutJava2goExecution(e,"Aa",value);ObjectView[*JavaString](old,StringTypeID)})
 if old!=box||effects!=1||p.MapGetJava2goExecution(e,"Aa")!="completed"{t.Fatal("result cast rolled back completed write")}
 if MapEntrySetValueExecution(e,entry,box)!="completed"||p.MapGetJava2goExecution(e,"Aa")!=box{t.Fatal("raw entry setter")}
 p.MapPutJava2goExecution(e,nil,nil);if !p.MapContainsKeyJava2goExecution(e,nil)||p.MapGetJava2goExecution(e,nil)!=nil{t.Fatal("null slot")}
 p.MapClearJava2goExecution(e);for _,view:=range []JavaIterable{entries,keys,values}{if CollectionSizeExecution(e,view)!=0{t.Fatal("old live view after clear")}}
 p.MapPutJava2goExecution(e,"Aa","x");cursor:=IterableIteratorExecution(e,entries);p.MapPutJava2goExecution(e,"BB","y");nativeMapTDDPanic(t,"ConcurrentModificationException",func(){IteratorNextExecution(e,cursor)})
 keyMap:=NewMap[string,string]();kp:=nativeMapTDDRolesOf(t,keyMap);kp.MapPutJava2goExecution(e,box,"v");keyCursor:=IterableIteratorExecution(e,kp.MapKeySetJava2goExecution(e));if IteratorNextExecution(e,keyCursor)!=box||IteratorHasNextExecution(e,keyCursor){t.Fatal("raw key identity/cursor advancement")}
}

type nativeMapTDDSource struct {
 *ObjectInfo
 want *Execution
 backend *Map[any,any]
 entries JavaIterable
 calls string
 sizeFailure,getFailure any
}
func nativeMapTDDSourceOf(e *Execution, backend *Map[any,any]) *nativeMapTDDSource {
 const id TypeID="native.map.tdd.Source";RegisterJavaType(id,AbstractMapTypeID)
 source:=&nativeMapTDDSource{want:e,backend:backend};source.ObjectInfo=NewObjectInfo(id,func(TypeID)any{return source});return source
}
func (s *nativeMapTDDSource) check(e *Execution,call string){if s.want!=nil&&s.want!=e{panic("substituted logical execution")};s.calls+=call}
func (s *nativeMapTDDSource) MapEntrySetJava2goExecution(e *Execution) JavaIterable{s.check(e,"E");if s.entries!=nil{return s.entries};return MapEntriesView(s.backend)}
func (s *nativeMapTDDSource) MapSizeJava2goExecution(e *Execution) int32{s.check(e,"S");if s.sizeFailure!=nil{panic(s.sizeFailure)};if s.entries!=nil{return CollectionSizeExecution(e,s.entries)};return s.backend.Size()}
func (s *nativeMapTDDSource) MapGetJava2goExecution(e *Execution,key any)any{s.check(e,"G");if s.getFailure!=nil{panic(s.getFailure)};return s.backend.GetObject(key,e)}
func (s *nativeMapTDDSource) MapContainsKeyJava2goExecution(e *Execution,key any)bool{s.check(e,"K");return s.backend.ContainsKey(key,e)}
func TestNativeMapErasedProtocolSourceEquality(t *testing.T) {
 e:=NewExecution();m:=NewMap[any,any]();m.PutObject("Aa","same",e);m.PutObject("BB",nil,e);sourceBackend:=NewMap[any,any]();sourceBackend.PutObject("Aa","same",e);sourceBackend.PutObject("BB",nil,e);source:=nativeMapTDDSourceOf(e,sourceBackend)
 if !m.EqualsJava2goExecution(e,source){t.Fatal("native/source equality refused")};view:=MapReferenceView(m);if !AbstractMapEqualsExecution(e,source,view){t.Fatal("source/native equality refused")}
 if m.HashCodeJava2goExecution(e)!=AbstractMapHashCodeExecution(e,source){t.Fatal("equal map hash")}
 sourceBackend.RemoveObject("BB",e);sourceBackend.PutObject("other",nil,e);if m.EqualsJava2goExecution(e,source)||AbstractMapEqualsExecution(e,source,view){t.Fatal("null mapped versus absent")}
}
func TestNativeMapErasedProtocolEqualityExceptionBoundary(t *testing.T) {
 e:=NewExecution();m:=NewMap[any,any]();m.PutObject("Aa","same",e);backend:=NewMap[any,any]();backend.PutObject("Aa","same",e);source:=nativeMapTDDSourceOf(e,backend)
 for _,failure:=range []any{nativeMapTDDSentinel("ClassCastException"),nativeMapTDDSentinel("NullPointerException")} {source.calls="";source.sizeFailure=failure;name:="ClassCastException";if CaughtAs(failure,"NullPointerException"){name="NullPointerException"};if nativeMapTDDPanic(t,name,func(){m.EqualsJava2goExecution(e,source)})!=failure{t.Fatal("size failure replaced")};if source.calls!="S"{t.Fatalf("size boundary calls=%s",source.calls)}}
 source.sizeFailure=nil;for _,failure:=range []any{NewClassCastException("get"),NewNullPointerException("get")} {source.calls="";source.getFailure=failure;if m.EqualsJava2goExecution(e,source)||source.calls!="SG"{t.Fatalf("get exception not caught after size: %s",source.calls)}}
 source.getFailure=nativeMapTDDSentinel("IllegalArgumentException");if nativeMapTDDPanic(t,"IllegalArgumentException",func(){m.EqualsJava2goExecution(e,source)})!=source.getFailure{t.Fatal("get failure replaced")}
 var absent *Map[any,any];nativeMapTDDPanic(t,"NullPointerException",func(){absent.EqualsJava2goExecution(e,m)})
 source.getFailure=nil;source.want=nil;if !m.Equals(source){t.Fatal("legacy source equality boundary")}
}

type nativeMapTDDPoisonEntry struct{*ObjectInfo;want *Execution;calls *string;failure any}
func (p *nativeMapTDDPoisonEntry) GetKeyJava2goExecution(e *Execution)any{if e!=p.want{panic("key execution")};*p.calls+="K";return "BB"}
func (p *nativeMapTDDPoisonEntry) GetValueJava2goExecution(e *Execution)any{if e!=p.want{panic("value execution")};*p.calls+="V";panic(p.failure)}
func (p *nativeMapTDDPoisonEntry) SetValueJava2goExecution(*Execution,any)any{panic("unexpected setter")}
func TestNativeMapErasedProtocolSourceCopyAndPrefix(t *testing.T) {
 e:=NewExecution();backend:=NewMap[any,any]();backend.PutObject("Aa","before",e);source:=nativeMapTDDSourceOf(e,backend)
 var copied *Map[any,any];nativeMapTDDNoPanic(t,func(){copied=NewMapWithArgument[any,any](source,e)});backend.PutObject("Aa","after",e);if copied.GetObject("Aa",e)!="before"{t.Fatal("source records aliased")}
 copied.Clear();copied.PutAll(source,e);if copied.GetObject("Aa",e)!="after"||source.calls!="SESE"{t.Fatalf("source PutAll callbacks or write: %s",source.calls)}
 calls:="";poison:=&nativeMapTDDPoisonEntry{want:e,calls:&calls,failure:nativeMapTDDSentinel("IllegalArgumentException")};poison.ObjectInfo=NewObjectInfo(JavaMapEntryTypeID,func(TypeID)any{return poison});list:=NewList[any]();list.Add(backend.entries[0]);list.Add(poison);source.entries=list;source.calls="";copied.Clear()
 if nativeMapTDDPanic(t,"IllegalArgumentException",func(){copied.PutAll(source,e)})!=poison.failure{t.Fatal("entry failure replaced")};if copied.Size()!=1||copied.GetObject("Aa",e)!="after"||copied.ContainsKey("BB",e)||calls!="KV"||source.calls!="SE"{t.Fatalf("prefix/entry evaluation calls=%s/%s",source.calls,calls)}
}
type nativeMapTDDKey struct{want *Execution;id int32;hashes,equals *int;fail bool;failure any}
func(k *nativeMapTDDKey) HashCodeJava2goExecution(e *Execution)int32{if e!=k.want{panic("hash execution")};*k.hashes++;if k.failure!=nil{panic(k.failure)};if k.fail{panic(NewIllegalArgumentException("hash"))};return 1}
func(k *nativeMapTDDKey) EqualsJava2goExecution(e *Execution,other any)bool{if e!=k.want{panic("equals execution")};*k.equals++;right,ok:=other.(*nativeMapTDDKey);return ok&&right.id==k.id}
func TestNativeMapErasedProtocolCallerExecutionAndWriteFailure(t *testing.T) {
 e:=NewExecution();m:=NewMap[any,any]();p:=nativeMapTDDRolesOf(t,m);hashes,equals,valueHashes,valueEquals:=0,0,0,0;key:=&nativeMapTDDKey{want:e,id:17,hashes:&hashes,equals:&equals};value:=&nativeMapTDDKey{want:e,id:7,hashes:&valueHashes,equals:&valueEquals};p.MapPutJava2goExecution(e,key,value);equalKey:=&nativeMapTDDKey{want:e,id:17,hashes:&hashes,equals:&equals};if p.MapGetJava2goExecution(e,equalKey)!=value||hashes!=2||equals!=1{t.Fatalf("caller hash/equals effects %d/%d",hashes,equals)}
 equalValue:=&nativeMapTDDKey{want:e,id:7,hashes:&valueHashes,equals:&valueEquals};if !p.MapContainsValueJava2goExecution(e,equalValue)||valueEquals!=1||valueHashes!=0{t.Fatal("containsValue lost caller equality execution")};if p.MapRemoveJava2goExecution(e,equalKey)!=value||hashes!=3||equals!=2||p.MapSizeJava2goExecution(e)!=0{t.Fatal("remove lost caller hash/equality execution")};p.MapPutJava2goExecution(e,key,value)
 effects:=0;bad:=&nativeMapTDDKey{want:e,id:18,hashes:&hashes,equals:&equals,fail:true};nativeMapTDDPanic(t,"IllegalArgumentException",func(){p.MapPutJava2goExecution(e,bad,func()any{effects++;return "write"}())});if effects!=1||p.MapSizeJava2goExecution(e)!=1{t.Fatal("hash failure effects or partial write")}
}
type nativeMapTDDComparable struct{want *Execution;id int32;calls *int;fail bool;failure any}
func(k *nativeMapTDDComparable) CompareToJava2goExecution(e *Execution,other *nativeMapTDDComparable)int32{if e!=k.want{panic("compare execution")};*k.calls++;if k.failure!=nil{panic(k.failure)};if k.fail{panic(NewIllegalArgumentException("compare"))};return k.id-other.id}
func TestNativeMapErasedProtocolSourceWriteFailurePrefix(t *testing.T) {
 t.Run("hash",func(t *testing.T){e:=NewExecution();hashes,equals:=0,0;first:=&nativeMapTDDKey{want:e,id:1,hashes:&hashes,equals:&equals};second:=&nativeMapTDDKey{want:e,id:2,hashes:&hashes,equals:&equals};backend:=NewMap[any,any]();backend.PutObject(first,"prefix",e);backend.PutObject(second,"second",e);source:=nativeMapTDDSourceOf(e,backend);target:=NewMap[any,any]();sentinel:=nativeMapTDDSentinel("IllegalArgumentException");second.failure=sentinel;if nativeMapTDDPanic(t,"IllegalArgumentException",func(){target.PutAll(source,e)})!=sentinel{t.Fatal("hash failure replaced")};second.failure=nil;if target.Size()!=1||target.GetObject(first,e)!="prefix"||target.ContainsKey(second,e)||source.calls!="SE"{t.Fatal("source hash failure prevalidated or rolled back prefix")}})
 t.Run("comparison",func(t *testing.T){e:=NewExecution();calls:=0;first:=&nativeMapTDDComparable{want:e,id:1,calls:&calls};second:=&nativeMapTDDComparable{want:e,id:2,calls:&calls};backend:=NewMap[any,any]();backend.PutObject(first,"prefix",e);backend.PutObject(second,"second",e);source:=nativeMapTDDSourceOf(e,backend);target:=NewTreeMap[any,any]();sentinel:=nativeMapTDDSentinel("IllegalArgumentException");second.failure=sentinel;if nativeMapTDDPanic(t,"IllegalArgumentException",func(){target.PutAll(source,e)})!=sentinel{t.Fatal("comparison failure replaced")};second.failure=nil;if target.Size()!=1||target.GetObject(first,e)!="prefix"||target.ContainsKey(second,e)||source.calls!="SE"{t.Fatal("source comparison failure prevalidated or rolled back prefix")}})
}
type nativeMapTDDLookalike struct{calls int}
func(p *nativeMapTDDLookalike) MapEntrySetJava2goExecution(*Execution)JavaIterable{p.calls++;return nil}
func TestNativeMapErasedProtocolNominalRefusalAndNull(t *testing.T) {
 e:=NewExecution();fake:=&nativeMapTDDLookalike{};nativeMapTDDPanic(t,"ClassCastException",func(){MapEntrySetExecution(e,MapReferenceView(fake))});if fake.calls!=0{t.Fatal("lookalike called before nominal refusal")};nativeMapTDDPanic(t,"ClassCastException",func(){MapReferenceView(NewList[any]())})
 var absent *Map[any,any];p:=nativeMapTDDRolesOf(t,absent);effects:=0;nativeMapTDDPanic(t,"NullPointerException",func(){p.MapPutJava2goExecution(e,"Aa",func()any{effects++;return "write"}())});if effects!=1{t.Fatal("argument not evaluated before null receiver")}
 for _,call:=range []func(){func(){p.MapEntrySetJava2goExecution(e)},func(){p.MapSizeJava2goExecution(e)},func(){p.MapIsEmptyJava2goExecution(e)},func(){p.MapContainsKeyJava2goExecution(e,"Aa")},func(){p.MapContainsValueJava2goExecution(e,"v")},func(){p.MapGetJava2goExecution(e,"Aa")},func(){p.MapRemoveJava2goExecution(e,"Aa")},func(){p.MapClearJava2goExecution(e)},func(){p.MapKeySetJava2goExecution(e)},func(){p.MapValuesJava2goExecution(e)}}{nativeMapTDDPanic(t,"NullPointerException",call)}
 tree:=nativeMapTDDRolesOf(t,NewTreeMap[string,string]());nativeMapTDDPanic(t,"NullPointerException",func(){tree.MapPutJava2goExecution(e,nil,"v")});if tree.MapSizeJava2goExecution(e)!=0{t.Fatal("null natural key inserted")}
}
func TestNativeMapErasedProtocolExistingNativeCompatibility(t *testing.T) {
 e:=NewExecution();left:=NewMap[any,any]();left.PutObject("Aa","one",e);right:=NewConcurrentHashMap[any,any]();right.Put("Aa","one",e);if !left.EqualsJava2goExecution(e,right)||!right.EqualsJava2goExecution(e,left){t.Fatal("native concurrent compatibility changed")}
 calls:=0;first:=&nativeMapTDDComparable{want:e,id:1,calls:&calls};second:=&nativeMapTDDComparable{want:e,id:2,calls:&calls};source:=NewTreeMap[any,string]();source.Put(first,"one",e);source.Put(second,"two",e);calls=0;first.fail=true;second.fail=true;target:=NewTreeMap[any,any]();target.PutAll(source,e);if calls!=0||target.Size()!=2{t.Fatal("native sorted fast copy invoked comparison")};first.fail=false;second.fail=false;source.Put(first,"later",e);if target.GetObject(first,e)!="one"{t.Fatal("native sorted copy records aliased")}
}

// Additive honest-empty-source controls: the original native controls remain byte-exact above.
type nativeMapTDDEmptySource struct {
 *ObjectInfo
 want *Execution
 mode, calls string
 failure any
}
func (s *nativeMapTDDEmptySource) observe(e *Execution, phase string) {
 if e != s.want { panic("empty source substituted logical execution") }
 s.calls += phase
 if s.mode == phase { panic(s.failure) }
}
func (s *nativeMapTDDEmptySource) MapSizeJava2goExecution(e *Execution) int32 { s.observe(e,"S"); return 0 }
func (s *nativeMapTDDEmptySource) MapEntrySetJava2goExecution(e *Execution) JavaIterable { s.observe(e,"E"); return &nativeMapTDDEmptyIterable{s} }
type nativeMapTDDEmptyIterable struct { source *nativeMapTDDEmptySource }
func (i *nativeMapTDDEmptyIterable) IteratorJava2goExecution(e *Execution) JavaIterator { i.source.observe(e,"I"); return &nativeMapTDDEmptyIterator{i.source} }
type nativeMapTDDEmptyIterator struct { source *nativeMapTDDEmptySource }
func (i *nativeMapTDDEmptyIterator) HasNextJava2goExecution(e *Execution) bool { i.source.observe(e,"H"); return false }
func (i *nativeMapTDDEmptyIterator) NextJava2goExecution(e *Execution) any { i.source.observe(e,"N"); panic("unexpected next on honest empty source") }
func TestNativeMapErasedProtocolEmptySourceObservations(t *testing.T) {
 for _, backend := range []string{"hash","linked","tree"} {
  for _, populated := range []bool{false,true} {
   for _, mode := range []string{"","E","I","H","S"} {
    state := "empty"; if populated { state = "populated" }; label := mode; if label == "" { label = "none" }
    t.Run(backend+"-"+state+"-"+label,func(t *testing.T){
     e := NewExecution(); target := NewMap[any,any](); if backend == "tree" { target = NewTreeMap[any,any]() }; if populated { target.PutObject("keep","unchanged",e) }
     failure := NewIllegalArgumentException("empty source sentinel"); sentinel := &failure
     source := &nativeMapTDDEmptySource{want:e,mode:mode,failure:sentinel}; const id TypeID = "native.map.tdd.EmptySource"; RegisterJavaType(id,AbstractMapTypeID); source.ObjectInfo = NewObjectInfo(id,func(TypeID)any{return source})
     var caught any; func(){defer func(){caught = recover()}();target.PutAll(source,e)}()
     wantTrace := "S"; wantFailure := mode == "S"
     if backend == "tree" && mode != "S" { wantTrace = "SEIH"; if mode != "" { wantFailure = true; switch mode {case "E":wantTrace="SE";case "I":wantTrace="SEI"} } }
     if source.calls != wantTrace { t.Fatalf("empty source PutAll observations: got %q want %q",source.calls,wantTrace) }
     if wantFailure { if caught != sentinel { t.Fatalf("empty source sentinel identity: got %T %v",caught,caught) } } else if caught != nil { t.Fatalf("unexpected empty source failure: %T %v",caught,caught) }
     wantSize := int32(0); if populated { wantSize = 1 }; if target.Size() != wantSize || (populated && target.GetObject("keep",e) != "unchanged") { t.Fatal("empty source changed target or discarded existing entry") }
    })
   }
  }
 }
}
