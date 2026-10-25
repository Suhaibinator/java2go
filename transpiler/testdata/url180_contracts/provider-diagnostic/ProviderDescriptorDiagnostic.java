import java.net.URL;
public final class ProviderDescriptorDiagnostic {
 public static void main(String[] args) {
  String[] inputs={"file:/tmp/provider-probe","jar:file:/tmp/provider.jar!/entry","http://Example.invalid/a","https://Example.invalid/a","catalog-unregistered:value","jar:http://Example.invalid/a.jar!/entry"};
  for(String input:inputs) {try{System.out.println(input+" OK "+new URL(input).toExternalForm());}catch(Throwable failure){System.out.println(input+" ERR "+failure.getClass().getName()+":"+failure.getMessage());}}
 }
}
