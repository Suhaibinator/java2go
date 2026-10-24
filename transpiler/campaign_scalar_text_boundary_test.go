package transpiler

import (
	"os"
	"testing"
)

func campaignScalarTextSource(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/campaign_scalar_boundary/" + name + ".java")
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}
func TestCampaignScalarTextBoundaryJDK21(t *testing.T) {
	campaignBigNumberOracle(t, "ScalarTextProbe", campaignScalarTextSource(t, "ScalarTextProbe"))
}
func TestCampaignArithmeticMessageBoundaryJDK21(t *testing.T) {
	campaignBigNumberOracle(t, "ArithmeticMessageProbe", campaignScalarTextSource(t, "ArithmeticMessageProbe"))
}
