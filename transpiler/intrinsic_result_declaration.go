package transpiler

import "strings"

// Fixed intrinsic result types describe library declarations, not source names
// in the caller's file. This catalog is independent of intrinsic init ordering
// and intentionally does not consult source imports or the dispatch-owner map.
// Argument-derived results retain their separate declaration/inference paths.
var intrinsicResultDeclarations = func() map[string]string {
	result := map[string]string{}
	for pkg, names := range map[string]string{
		"java.lang":            "Object String StringBuilder StringBuffer Class Number Boolean Byte Short Character Integer Long Float Double Throwable Thread Runnable Appendable",
		"java.util":            "Date TimeZone Calendar GregorianCalendar Locale List Set Map Comparator Optional OptionalInt OptionalLong OptionalDouble IntSummaryStatistics LongSummaryStatistics DoubleSummaryStatistics",
		"java.util.concurrent": "ExecutorService TimeUnit",
		"java.util.stream":     "Stream IntStream LongStream DoubleStream",
		"java.lang.reflect":    "Constructor Field Method Type ParameterizedType GenericArrayType WildcardType TypeVariable GenericDeclaration",
		"java.io":              "File InputStream Writer StringWriter OutputStreamWriter BufferedWriter FileWriter PrintWriter",
		"java.nio":             "ByteBuffer ByteOrder",
		"java.nio.charset":     "Charset",
		"java.nio.file":        "Path StandardOpenOption",
		"java.security":        "MessageDigest",
	} {
		for _, name := range strings.Fields(names) {
			result[name] = pkg + "." + name
		}
	}
	return result
}()

// canonicalIntrinsicResultType qualifies nominal nodes in a fixed result's
// arrays, generic arguments, and wildcard bounds. Unknown identifiers remain
// unchanged: in particular a declared result variable T is not a caller class.
// Fully qualified names retain their original package identity.
func canonicalIntrinsicResultType(javaType string) string {
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	suffix := strings.Repeat("[]", rank)
	for _, prefix := range []string{"? extends ", "? super "} {
		if strings.HasPrefix(base, prefix) {
			return prefix + canonicalIntrinsicResultType(strings.TrimPrefix(base, prefix)) + suffix
		}
	}
	base, arguments := parseJavaTypeString(base)
	if canonical, known := intrinsicResultDeclarations[base]; known {
		base = canonical
	}
	if len(arguments) > 0 {
		for index, argument := range arguments {
			arguments[index] = canonicalIntrinsicResultType(argument)
		}
		base += "<" + strings.Join(arguments, ", ") + ">"
	}
	return base + suffix
}

// declaredIntrinsicResultShell attaches the library declaration to a derived
// result's outer type. Its arguments already carry their inference-site
// identities and must not be reinterpreted as library declarations.
func declaredIntrinsicResultShell(owner string, arguments ...string) string {
	result := canonicalIntrinsicResultType(owner)
	if len(arguments) > 0 {
		result += "<" + strings.Join(arguments, ", ") + ">"
	}
	return result
}
