package foreign;
public class Value implements Comparable<Value> {public int key;public Value(int k){key=k;}public int compareTo(Value value){return key-value.key;}}
