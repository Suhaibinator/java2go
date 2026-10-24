package namespace;
public class SourceTypePrefixOracle<T>{
 static class java{static class lang{static class Class<U>{U value;Class(U value){this.value=value;}}}}
 java.lang.Class<T> source;
 SourceTypePrefixOracle(T value){source=new java.lang.Class<>(value);}
 public static void main(String[] args){SourceTypePrefixOracle<String> app=new SourceTypePrefixOracle<>("source");System.out.println(app.source.value);}
}
