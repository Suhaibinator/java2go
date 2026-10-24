package transpiler

import (
 "testing"
 "github.com/NickyBoy89/java2go/symbol"
)

func nativeExternalOrigin3930Owner(t *testing.T, source, owner string) Ctx {
 t.Helper()
 h:=setupParseHelper(t,source)
 scope:=h.File.Symbols.FindClassScope(owner)
 if scope==nil {t.Fatal("actual parsed owner missing")}
 ctx:=classScopeCtx(scope,h.Ctx)
 if ctx.currentFile!=h.File.Symbols || findFileScopeForClassScope(scope,ctx)!=h.File.Symbols {t.Fatal("actual declaration file not retained")}
 return ctx
}

func nativeExternalOrigin3930Parameter(t *testing.T, ctx Ctx, name string) inferredJavaTypeOrigin {
 t.Helper()
 defs:=ctx.currentClass.FindMethod().ByOriginalName(name)
 if len(defs)!=1 || len(defs[0].Parameters)!=1 {t.Fatal("actual parameter declaration missing")}
 declaring:=ctx.Clone();declaring.localScope=defs[0]
 return declaredJavaTypeOrigin(definitionDeclaredJavaType(defs[0].Parameters[0]),declaring)
}

func TestNativeExternalOrigin3930ParsedCatalog(t *testing.T) {
 ctx:=nativeExternalOrigin3930Owner(t,`import java.io.StringReader;class NativeOwner{void builder(StringBuilder value){}void buffer(StringBuffer value){}void reader(StringReader value){}void qualified(java.io.StringReader value){}}`,"NativeOwner")
 for method,owner:=range map[string]string{"builder":"java.lang.StringBuilder","buffer":"java.lang.StringBuffer","reader":"java.io.StringReader","qualified":"java.io.StringReader"} {
  t.Run(method,func(t *testing.T){
   got:=nativeExternalOrigin3930Parameter(t,ctx,method)
   if got.unresolved || got.externalName!=owner || got.nominalScope!=nil || got.parameter!=nil || got.arrayRank!=0 || got.declaringOwner!=ctx.currentClass {t.Fatalf("actual declared native origin refused: method=%s origin=%+v",method,got)}
  })
 }
}

func TestNativeExternalOrigin3930ParsedRelation(t *testing.T) {
 ctx:=nativeExternalOrigin3930Owner(t,`class NativeOwner{void builder(StringBuilder value){}void buffer(StringBuffer value){}void target(CharSequence value){}void array(StringBuilder[] value){}void root(Object value){}}`,"NativeOwner")
 formal:=nativeExternalOrigin3930Parameter(t,ctx,"target")
 for _,name:=range []string{"builder","buffer"} {
  got:=nativeExternalOrigin3930Parameter(t,ctx,name)
  cost,exact,applies,handled:=declaredOriginConversion(got,formal,ctx)
  if !handled || !applies || exact || cost!=1 {t.Fatalf("actual native CharSequence endpoint refused: %s cost=%d exact=%v applies=%v handled=%v",name,cost,exact,applies,handled)}
 }
 array:=nativeExternalOrigin3930Parameter(t,ctx,"array")
 if _,_,applies,handled:=declaredOriginConversion(array,formal,ctx);!handled || applies {t.Fatal("native array gained scalar CharSequence")}
 builder:=nativeExternalOrigin3930Parameter(t,ctx,"builder");buffer:=nativeExternalOrigin3930Parameter(t,ctx,"buffer")
 if _,_,applies,handled:=declaredOriginConversion(buffer,builder,ctx);!handled || applies {t.Fatal("shared physical builder carrier erased distinct logical classes")}
 if _,_,applies,handled:=declaredOriginConversion(array,nativeExternalOrigin3930Parameter(t,ctx,"root"),ctx);!handled || !applies {t.Fatal("existing Object array widening refused")}
}

func TestNativeExternalOrigin3930ParsedShields(t *testing.T) {
 t.Run("typeRoot",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`class Guard{static class java{static class lang{static class StringBuilder{}}static class io{static class StringReader{}}}void builder(java.lang.StringBuilder value){}void reader(java.io.StringReader value){}}`,"Guard")
  for _,method:=range []string{"builder","reader"} {got:=nativeExternalOrigin3930Parameter(t,ctx,method);if got.externalName!="" || got.nominalScope==nil {t.Fatalf("source TYPE prefix became native: %+v",got)}}
 })
 t.Run("valueRoot",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`class Guard{int java;void builder(java.lang.StringBuilder value){}void reader(java.io.StringReader value){}}`,"Guard")
  for method,owner:=range map[string]string{"builder":"java.lang.StringBuilder","reader":"java.io.StringReader"} {got:=nativeExternalOrigin3930Parameter(t,ctx,method);if got.externalName!=owner || got.unresolved {t.Fatalf("VALUE root changed TYPE declaration: %+v",got)}}
 })
 t.Run("foreignImport",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`import foreign.StringBuilder;class Guard{void builder(StringBuilder value){}}`,"Guard")
  got:=nativeExternalOrigin3930Parameter(t,ctx,"builder");if got.externalName!="" || !got.unresolved {t.Fatalf("unresolved foreign import became native: %+v",got)}
 })
 t.Run("sourceExpected",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`class CharSequence{}class Guard{void builder(StringBuilder value){}void target(CharSequence value){}}`,"Guard")
  a:=nativeExternalOrigin3930Parameter(t,ctx,"builder");f:=nativeExternalOrigin3930Parameter(t,ctx,"target")
  if f.nominalScope==nil {t.Fatal("source expected declaration missing")};if _,_,applies,handled:=declaredOriginConversion(a,f,ctx);!handled || applies {t.Fatal("native actual gained source CharSequence")}
 })
 t.Run("capturedClass",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`class Guard<StringBuilder>{void builder(StringBuilder value){}}`,"Guard")
  got:=nativeExternalOrigin3930Parameter(t,ctx,"builder");actual:=ctx.currentClass.TypeParameters[0].Declaration
  if got.parameter!=actual || got.externalName!="" {t.Fatalf("actual class binder lost identity: %+v",got)}
 })
 t.Run("capturedMethods",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`class Guard{<StringBuffer>void buffer(StringBuffer value){}<StringReader>void reader(StringReader value){}}`,"Guard")
  for _,method:=range []string{"buffer","reader"} {defs:=ctx.currentClass.FindMethod().ByOriginalName(method);got:=nativeExternalOrigin3930Parameter(t,ctx,method);if got.parameter!=defs[0].TypeParameters[0].Declaration || got.externalName!="" {t.Fatalf("actual method binder lost identity: %+v",got)}}
 })
 t.Run("missingAndStaleFile",func(t *testing.T){
  ctx:=nativeExternalOrigin3930Owner(t,`class Guard{}`,"Guard")
  noFile:=ctx.Clone();noFile.currentFile=nil
  noOwner:=ctx.Clone();noOwner.currentClass=nil
  stale:=ctx.Clone();foreignFile:=*ctx.currentFile;stale.currentFile=&foreignFile
  for _,bad:=range []Ctx{noFile,noOwner,stale} {if got:=declaredExternalNominalOwner("java.lang.StringBuilder",nil,bad);got!="" {t.Fatalf("new canonical proof lacked actual owning file: %s",got)}}
 })
}

func TestNativeExternalOrigin3930SourceNamesAndEmittedAliases(t *testing.T) {
 for _,alias:=range []string{"StringBuilder","StringBuffer","StringReader","java"} {
  t.Run(alias,func(t *testing.T){
   ctx:=nativeExternalOrigin3930Owner(t,`import java.io.StringReader;class Guard<Unrelated>{void builder(StringBuilder value){}void buffer(StringBuffer value){}void reader(StringReader value){}void qualified(java.io.StringReader value){}void bound(Unrelated value){}}`,"Guard")
   declaration:=ctx.currentClass.TypeParameters[0].Declaration
   if declaration==nil || declaration.SourceName!="Unrelated" {t.Fatal("parser source declaration identity missing")}
   before:=declaration.GoName;declaration.GoName=alias;t.Cleanup(func(){declaration.GoName=before})
   for repeat:=0;repeat<3;repeat++ {
    for method,owner:=range map[string]string{"builder":"java.lang.StringBuilder","buffer":"java.lang.StringBuffer","reader":"java.io.StringReader","qualified":"java.io.StringReader"} {
     got:=nativeExternalOrigin3930Parameter(t,ctx,method)
     if got.externalName!=owner || got.unresolved || got.parameter!=nil {t.Fatalf("emitted alias stole actual source spelling: alias=%s method=%s origin=%+v",alias,method,got)}
    }
    bound:=nativeExternalOrigin3930Parameter(t,ctx,"bound")
    if bound.parameter!=declaration || bound.externalName!="" || declaration.SourceName!="Unrelated" {t.Fatal("source binder identity changed during repeated native proof")}
   }
  })
 }
}

var _ *symbol.TypeParamDeclaration
