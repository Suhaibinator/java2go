package transpiler

import "testing"

func TestCampaignBigIntegerArithmeticJDK21(t *testing.T) {
	const source = `import java.math.BigInteger;
public class CampaignBigIntegerArithmetic {
 static int calls=0,cleaned=0;
 static BigInteger operand(){calls++;return new BigInteger("2");}
 public static String run(){BigInteger a=new BigInteger("18446744073709551617"),b=new BigInteger("-4294967297");BigInteger product=a.multiply(b),sum=a.add(b),difference=a.subtract(b);String result=a.toString()+":"+b.toString()+":"+product.toString()+":"+sum.toString()+":"+difference.toString()+":"+product.divide(new BigInteger("11")).toString()+":"+product.mod(new BigInteger("97")).toString()+":"+product.bitLength()+":"+new BigInteger("-1").bitLength()+":"+new BigInteger("-2147483648").bitLength()+":"+new BigInteger("-18446744073709551616").bitLength()+":"+new BigInteger("-18446744073709551617").bitLength();try{a.divide(BigInteger.valueOf(0));}catch(ArithmeticException e){result+="|"+e.getMessage();}finally{cleaned++;}try{a.mod(BigInteger.valueOf(-1));}catch(ArithmeticException e){result+="|"+e.getMessage();}finally{cleaned++;}try{BigInteger missing=null;missing.add(operand());}catch(NullPointerException e){result+="|null";}finally{cleaned++;}try{a.subtract(null);}catch(NullPointerException e){result+="|argument";}finally{cleaned++;}return result+"|"+calls+":"+cleaned;}
}`
	campaignBigNumberOracle(t, "CampaignBigIntegerArithmetic", source)
}
