package p; public class Base<T>{public T value;public int reads;public Base(T value){this.value=value;}public T read(){reads++;return value;}public void write(T value){this.value=value;}}
