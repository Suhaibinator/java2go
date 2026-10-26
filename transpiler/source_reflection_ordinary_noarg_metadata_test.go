package transpiler

import "testing"

func TestSourceReflectionOrdinaryNoargMetadataJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
public class Main {
 public static class Implicit {public int value=4;}
 public static class Explicit {public int value; public Explicit(){value=5;}}
 public static void main(String[] args) throws Exception {
  System.out.println("implicit:"+Implicit.class.getConstructor().newInstance().value);
  System.out.println("public:"+Explicit.class.getConstructor().newInstance().value);
  java.lang.reflect.Constructor<Secret> constructor=Secret.class.getDeclaredConstructor();
  try {constructor.newInstance();} catch(IllegalAccessException expected){System.out.println("private:denied");}
  constructor.setAccessible(true);
  System.out.println("private:"+constructor.newInstance().value);
 }
}`, "implicit:4\npublic:5\nprivate:denied\nprivate:6\n", map[string]string{
		"Secret.java": `public class Secret {public int value; private Secret(){value=6;}}`,
	})
}
