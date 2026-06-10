package stdjava

import "strconv"

// This file implements the boxed-type parsing statics (Integer.parseInt,
// Long.parseLong, Double.parseDouble, Boolean.parseBoolean). Java's parse
// methods return an unwrapped primitive and throw NumberFormatException on bad
// input; the Go equivalents return an error, so these helpers panic on failure
// to preserve the "parse or throw" contract. In this codebase a Java int is an
// int32, so ParseInt returns int32.

// ParseInt parses a base-10 int32, matching Integer.parseInt / Integer.valueOf.
func ParseInt(s string) int32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		panic(err)
	}
	return int32(v)
}

// ParseLong parses a base-10 int64, matching Long.parseLong / Long.valueOf.
func ParseLong(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseDouble parses a float64, matching Double.parseDouble / Double.valueOf.
func ParseDouble(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseBoolean reports whether s equals "true" ignoring case, matching
// Boolean.parseBoolean (which returns false for any other value rather than
// throwing).
func ParseBoolean(s string) bool {
	return len(s) == 4 &&
		(s[0] == 't' || s[0] == 'T') &&
		(s[1] == 'r' || s[1] == 'R') &&
		(s[2] == 'u' || s[2] == 'U') &&
		(s[3] == 'e' || s[3] == 'E')
}
