package model;
import audit.Audit;
public record Value(int number, String label) {
 public Value { Audit.before(this); if(number<0) throw new IllegalArgumentException("negative"); number+=10; label+="!"; }
 public Value(){this(Audit.number("zeroN",1),Audit.label("zeroL","x"));Audit.same("zero",this);}
 public Value(int number){this(number,Audit.label("oneL","n"));Audit.same("one",this);}
 public Value(String label){this(Audit.number("textN",2),label);Audit.same("text",this);}
}
