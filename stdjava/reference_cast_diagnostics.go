package stdjava

import "fmt"

// referenceCastException reports a bytecode reference-cast rejection using only
// declared identities and their published runtime locations. Class.cast has a
// separate message contract and must not use this constructor.
func referenceCastException(actual, requested TypeID) ClassCastException {
	return NewClassCastException(referenceCastMessage(actual, requested, ClassRuntimeLocationOf(actual), ClassRuntimeLocationOf(requested)))
}

func referenceCastMessage(actual, requested TypeID, actualLocation, requestedLocation ClassRuntimeLocation) string {
	// Incomplete provenance cannot justify a JVM module/loader diagnostic. Keep
	// the existing rejection text for that unsupported metadata frontier.
	if actual == "" || requested == "" || !actualLocation.Known || !requestedLocation.Known {
		return fmt.Sprintf("Java value %s is not assignable to %s", actual, requested)
	}
	actualName, requestedName := javaClassBinaryName(actual), javaClassBinaryName(requested)
	message := "class " + actualName + " cannot be cast to class " + requestedName
	domain := func(location ClassRuntimeLocation) string {
		module := "unnamed module"
		if location.ModuleName != "" {
			module = "module " + location.ModuleName
		}
		return module + " of loader '" + location.LoaderName + "'"
	}
	if actualLocation == requestedLocation {
		return message + " (" + actualName + " and " + requestedName + " are in " + domain(actualLocation) + ")"
	}
	return message + " (" + actualName + " is in " + domain(actualLocation) + "; " + requestedName + " is in " + domain(requestedLocation) + ")"
}
