package transpiler

import "testing"

func TestCampaignScalarNaNRawJDK21(t *testing.T) {
	campaignBigNumberOracle(t, "ScalarNaNRawProbe", campaignScalarTextSource(t, "ScalarNaNRawProbe"))
}
