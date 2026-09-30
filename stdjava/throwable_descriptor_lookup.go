package stdjava

import "strings"

// RuntimeThrowableDescriptor exposes nominal runtime metadata without implying
// a concrete Go constructor exists. Compiler processes initialize runtime types
// before translating source; generated source registrations live in the separate
// executable. Source-marked entries are excluded even when queried in that process.
// Qualified names require an exact match; ambiguous simple names are rejected.
func RuntimeThrowableDescriptor(name string) (id, parent TypeID, known bool) {
	javaTypeRegistry.RLock()
	defer javaTypeRegistry.RUnlock()
	qualified := strings.Contains(name, ".")
	for candidate, descriptor := range javaTypeRegistry.types {
		if descriptor.source || !strings.HasPrefix(string(candidate), "java.") {
			continue
		}
		if qualified {
			if string(candidate) != name {
				continue
			}
		} else if string(candidate)[strings.LastIndex(string(candidate), ".")+1:] != name {
			continue
		}
		seen := map[TypeID]bool{}
		current := candidate
		for current != "" && !seen[current] {
			if current == ThrowableTypeID {
				if known {
					return "", "", false
				}
				id, parent, known = candidate, descriptor.super, true
				break
			}
			seen[current] = true
			edge, exists := javaTypeRegistry.types[current]
			if !exists || edge.source {
				break
			}
			current = edge.super
		}
	}
	return id, parent, known
}

// CaughtAsType matches resolved Java identities, including source declarations
// whose simple name shadows a runtime exception. CaughtAs retains its native
// legacy simple-name contract for callers without nominal metadata.
func CaughtAsType(recovered any, expected TypeID) bool {
	actual, known := ObjectDynamicType(recovered)
	return known && JavaTypeAssignable(actual, expected)
}
