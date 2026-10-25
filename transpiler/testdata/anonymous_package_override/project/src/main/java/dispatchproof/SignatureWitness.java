package dispatchproof;
import right.Token;
public class SignatureWitness {public static void main(String[] args){var value=new SignatureBase(){public int choose(Token value){return 2;}};System.out.println(value.route()+":"+value.choose(new Token()));}}
