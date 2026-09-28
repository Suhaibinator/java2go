package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignExceptionCauseConstructorsJVMParity(t *testing.T) {
	const source = `public class ExceptionCauseConstructors {
  public static class CodecFailure extends Exception {
   public CodecFailure(String message, Throwable cause) { super(message, cause); }
   public CodecFailure(Throwable cause) { super(cause); }
  }
  public static String run() {
   Throwable cause = new IllegalArgumentException("bad input");
   CodecFailure described = new CodecFailure("decode", cause);
   CodecFailure inferred = new CodecFailure(cause);
   Exception direct = new Exception("direct", cause);
   return described.getMessage() + ":" + (described.getCause() == cause)
    + ":" + inferred.getMessage() + ":" + (inferred.getCause() == cause)
    + ":" + direct.getMessage() + ":" + (direct.getCause() == cause);
  }
 }`
	want := campaignRuntimeJavaOracle(t, "ExceptionCauseConstructors", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestExceptionCause(t *testing.T) { if got:=Run();got!=%q {t.Fatalf("got %%q want %%q",got,%q)} }
 `, want, want))
}

func TestCampaignExceptionTypesAndFinalCatchJVMParity(t *testing.T) {
	const source = `import java.io.UnsupportedEncodingException;
 public class ExceptionSignatures {
  public static class Failure extends Exception {
   public Failure() {super();}
   public Failure(String message) {super(message);}
   public Failure(Throwable cause) {super(cause);}
   public Failure(String message, Throwable cause) {super(message,cause);}
  }
  static IllegalStateException wrap(UnsupportedEncodingException cause) {
   return new IllegalStateException("encoding",cause);
  }
  static String caught() {
   try { throw new ClassCastException("bad cast"); }
   catch (@SuppressWarnings("unused") final ClassCastException e) {
    Failure wrapped = new Failure(e.getMessage(), e);
    return wrapped.getMessage() + ":" + (wrapped.getCause() == e);
   }
  }
  public static String run() {
   UnsupportedEncodingException cause = new UnsupportedEncodingException("bad encoding");
   IllegalStateException wrapped=wrap(cause);
   RuntimeException runtime = new RuntimeException(cause);
   IllegalArgumentException illegal = new IllegalArgumentException("argument",cause);
   return caught()+":"+wrapped.getMessage()+":"+(wrapped.getCause()==cause)
    +":"+runtime.getMessage()+":"+illegal.getMessage()+":"+(illegal.getCause()==cause);
  }
 }`
	want := campaignRuntimeJavaOracle(t, "ExceptionSignatures", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestSignatures(t *testing.T) {if got:=Run();got!=%q {t.Fatalf("got %%q want %%q",got,%q)}}`, want, want))
}
