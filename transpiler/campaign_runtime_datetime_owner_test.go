package transpiler

import "testing"

func TestCampaignRuntimeDateOwnerClassLiteral(t *testing.T) {
	for _, test := range []struct{ name, imports, owner string }{
		{"ExplicitSQLDateOwner", "import java.sql.Date;", "Date"},
		{"WildcardSQLDateOwner", "import java.sql.*;", "Date"},
		{"QualifiedSQLDateOwner", "", "java.sql.Date"},
		{"ExplicitUtilDateOwner", "import java.util.Date;", "Date"},
		{"WildcardUtilDateOwner", "import java.util.*;", "Date"},
		{"QualifiedUtilWithSQLImport", "import java.sql.Date;", "java.util.Date"},
		{"QualifiedSQLWithUtilImport", "import java.util.Date;", "java.sql.Date"},
		{"ExplicitSQLOverUtilWildcard", "import java.util.*;import java.sql.Date;", "Date"},
		{"ExplicitUtilOverSQLWildcard", "import java.sql.*;import java.util.Date;", "Date"},
	} {
		t.Run(test.name, func(t *testing.T) {
			campaignDateTimeOracle(t, test.name, test.imports+"public class "+test.name+"{public static String run(){return "+test.owner+".class.getName();}}")
		})
	}
}

func TestCampaignRuntimeDateOwnerConstruction(t *testing.T) {
	for _, test := range []struct{ name, imports, owner string }{
		{"ExplicitSQLDateConstruction", "import java.sql.Date;", "Date"},
		{"WildcardSQLDateConstruction", "import java.sql.*;", "Date"},
		{"QualifiedSQLDateConstruction", "", "java.sql.Date"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := test.imports + "public class " + test.name + "{public static String run(){Object value=new " + test.owner + "(0L);return value.getClass().getName();}}"
			campaignDateTimeOracle(t, test.name, source)
		})
	}
}

func TestCampaignRuntimeDateOwnerSourceShadow(t *testing.T) {
	const source = `class Date {public long getTime(){return 17L;}}
 public class DateOwnerShadow {public static String run(){Date source=new Date();java.util.Date external=new java.util.Date(42L);Object value=external;return source.getTime()+":"+external.getTime()+":"+Date.class.getName()+":"+java.util.Date.class.getName()+":"+value.getClass().getName();}}`
	campaignDateTimeOracle(t, "DateOwnerShadow", source)
}

func TestCampaignRuntimeDateOwnerExplicitImportBeatsPackageSource(t *testing.T) {
	const local = `package owner; public class Date {public Date(long value){}}`
	const main = `package owner; import java.sql.Date; public class Main {public static void main(String[] args){Object value=new Date(0L);System.out.print(value.getClass().getName());}}`
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml":                       `<project><modelVersion>4.0.0</modelVersion><groupId>owner</groupId><artifactId>dates</artifactId><version>1</version></project>`,
		"src/main/java/owner/Date.java": local,
		"src/main/java/owner/Main.java": main,
	}, "owner.Main", "java.sql.Date")
}
