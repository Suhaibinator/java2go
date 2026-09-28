package campaign

// Frozen campaign oracles use the JVM's default disabled assertion mode.
// Pin the matching Go mode explicitly rather than inheriting a user's setting.
func goProgramCommand(binary string, args []string) []string {
	return append([]string{"env", "JAVA2GO_ASSERTIONS=false", binary}, args...)
}
