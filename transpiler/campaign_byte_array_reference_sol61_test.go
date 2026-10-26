package transpiler

import (
	"fmt"
	"os"
	"testing"
)

func runByteArrayCanonicalReferenceSol61(t *testing.T, seed int) {
	t.Helper()
	const fixture = "testdata/byte_array_reference_sol61/"
	source, err := os.ReadFile(fixture + "Main.java")
	if err != nil { t.Fatal(err) }
	pom, err := os.ReadFile(fixture + "pom.xml")
	if err != nil { t.Fatal(err) }
	expected, err := os.ReadFile(fmt.Sprintf("%sseed-%d.stdout", fixture, seed))
	if err != nil { t.Fatal(err) }
	expectedError, err := os.ReadFile(fmt.Sprintf("%sseed-%d.stderr", fixture, seed))
	if err != nil { t.Fatal(err) }
	if len(expectedError) != 0 { t.Fatal("actual JDK oracle has unexpected stderr") }
	// The wrapper supplies the same captured seed without changing Main.java.
	wrapper := fmt.Sprintf(`package probe;
public class OracleEntry {
 public static void main(String[] args) throws Exception {
  Main.main(new String[]{"%d"});
 }
}`, seed)
	files := map[string]string{
		"pom.xml": string(pom),
		"src/main/java/probe/Main.java": string(source),
		"src/main/java/probe/OracleEntry.java": wrapper,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "probe.OracleEntry", string(expected))
}

func TestCampaignByteArrayCanonicalReferenceSeed17Sol61(t *testing.T) {
	runByteArrayCanonicalReferenceSol61(t, 17)
}
func TestCampaignByteArrayCanonicalReferenceSeed41Sol61(t *testing.T) {
	runByteArrayCanonicalReferenceSol61(t, 41)
}
func TestCampaignByteArrayCanonicalReferenceSeed97Sol61(t *testing.T) {
	runByteArrayCanonicalReferenceSol61(t, 97)
}
