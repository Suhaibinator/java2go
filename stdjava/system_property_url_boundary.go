package stdjava

// Property storage is general; URL's admitted handler profile is deliberately
// narrower. Reject unprovided service modes at actual URL use, without rejecting
// valid System property writes or silently pretending a provider was consulted.
func requireSystemPropertyURLProfile() {
	if packages := systemPropertyConfigured("java.protocol.handler.pkgs"); packages != nil && len(packages.units) != 0 {
		panic(NewUnsupportedOperationException("configured URL handler packages are not implemented"))
	}
	// The selected JDK21 default is delayed parsing. Empty and case-insensitive
	// true retain it; Boolean.parseBoolean selects the unprovided eager mode for
	// every other nonempty value. No URL is constructed or network consulted here.
	if delay := systemPropertyConfigured("jdk.net.url.delayParsing"); delay != nil && len(delay.units) != 0 && !urlASCIIEqualFold(delay.units, "true") {
		panic(NewUnsupportedOperationException("configured eager URL parsing is not implemented"))
	}
}
