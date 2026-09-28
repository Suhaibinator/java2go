package transpiler

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func TestCampaignRuntimeDateOwnerUnsupportedConstruction(t *testing.T) {
	for _, test := range []struct{ name, imports, owner string }{
		{"ExplicitSQLDateConstruction", "import java.sql.Date;", "Date"},
		{"WildcardSQLDateConstruction", "import java.sql.*;", "Date"},
		{"QualifiedSQLDateConstruction", "", "java.sql.Date"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := test.imports + "public class " + test.name + "{public static String run(){Object value=new " + test.owner + "(0L);return value.getClass().getName();}}"
			if got := campaignRuntimeJavaOracle(t, test.name, source); got != "java.sql.Date" {
				t.Fatalf("JVM oracle %q", got)
			}
			withCleanDiagnostics(t)
			input := filepath.Join(t.TempDir(), "input")
			writeJavaSource(t, input, test.name+".java", source)
			err := run([]string{"-strict", input}, io.Discard)
			if err == nil || !strings.Contains(err.Error(), "java.sql.Date") {
				t.Fatalf("expected unsupported canonical SQL owner, got %v", err)
			}
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
	input := filepath.Join(t.TempDir(), "input")
	writeJavaSource(t, input, "Date.java", local)
	writeJavaSource(t, input, "Main.java", main)
	home := os.Getenv("JAVA_HOME")
	javac, java := "javac", "java"
	if home != "" {
		javac = filepath.Join(home, "bin", "javac")
		java = filepath.Join(home, "bin", "java")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, javac, "--release", "21", "-d", input, filepath.Join(input, "Date.java"), filepath.Join(input, "Main.java")).CombinedOutput(); err != nil {
		t.Fatalf("JDK21 compile: %v %s", err, out)
	}
	out, err := exec.CommandContext(ctx, java, "-cp", input, "owner.Main").CombinedOutput()
	if err != nil || string(out) != "java.sql.Date" {
		t.Fatalf("JVM oracle %q: %v", out, err)
	}
	withCleanDiagnostics(t)
	err = run([]string{"-strict", input}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "java.sql.Date") {
		t.Fatalf("explicit SQL import must not bind package source Date: %v", err)
	}
}
