package transpiler

import "testing"

// Keep original Java witnesses intact and enforce strict mode for this paired gate.
func TestCampaignCharsStrictJVMParity(t *testing.T) {
    previousStrict := diagnostics.strict
    setStrictMode(true)
    t.Cleanup(func() { setStrictMode(previousStrict) })
    t.Run("UTF16", TestNumericStreams_CanonicalStringCharsBehavior)
    t.Run("InvocationNull", TestNumericStreams_CanonicalStringCharsNullBehavior)
}
