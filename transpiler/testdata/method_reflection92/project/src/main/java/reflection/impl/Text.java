package reflection.impl;
import reflection.api.Base;
public class Text extends Base<String> {
    public int calls = 0;
    public String value() { calls++; return "leaf"; }
    public String echo(String value) { calls++; return "echo:" + value; }
}
