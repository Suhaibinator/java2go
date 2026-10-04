package contract.binder;
interface Mapper<adapter> { adapter apply(adapter value); default adapter identity(adapter value) { return apply(value); } }
public class Main { public static void main(String[] args) { Mapper<String> mapper = value -> value; System.out.println(mapper.identity("binder")); } }
