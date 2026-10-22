package transpiler

// Both names already carry declaration-origin authority. Raw classfile edges
// are sufficient only for scalar endpoints with no supplied type arguments;
// typed parent substitution and inference retain their separate proof paths.
func jdk21RawReferenceOriginAssignable(actual, formal inferredJavaTypeOrigin) bool {
	for _, value := range []inferredJavaTypeOrigin{actual, formal} {
		if value.unresolved || !completeJavaTypeOrigin(value) || value.externalName == "" || value.nominalScope != nil || value.parameter != nil || value.arrayRank != 0 || value.wildcard || len(value.arguments) != 0 || !jdk21RawReferenceCanonicalOwner(value.externalName) {
			return false
		}
	}
	return jdk21RawParentRecordsAssignable(actual.externalName, formal.externalName, jdk21MemberOwners, jdk21MemberOwners)
}

func jdk21RawReferenceCanonicalOwner(owner string) bool {
	if member, known := jdk21MemberDeclarations[owner]; known {
		return member.binaryName == owner && jdk21MemberEligible(member)
	}
	declaration, presence := jdk21NominalQualified(owner)
	return presence == jdk21NominalPresent && declaration.binaryName == owner && jdk21NominalEligible(declaration)
}

// The production graph and attestation are the same immutable generated JDK
// profile. Separate arguments let tests verify refusal of stale or altered
// records without mutating that profile. Only certified classfile parent edges
// participate; member names, source names and Go representation never do.
func jdk21RawParentRecordsAssignable(actual, formal string, owners, attested map[string]jdk21MemberOwner) bool {
	if _, known := owners[formal]; !known {
		return false
	}
	states := map[string]uint8{}
	found := false
	var visit func(string) bool
	visit = func(owner string) bool {
		if states[owner] == 1 {
			return false
		}
		if states[owner] == 2 {
			return true
		}
		record, known := owners[owner]
		proof, certified := attested[owner]
		if !known || !certified || record.certificate == "" || record.certificate != proof.certificate || record.superclass != proof.superclass || len(record.interfaces) != len(proof.interfaces) {
			return false
		}
		for index, parent := range record.interfaces {
			if parent != proof.interfaces[index] {
				return false
			}
		}
		states[owner] = 1
		if owner == formal {
			found = true
		}
		if record.superclass != "" && !visit(record.superclass) {
			return false
		}
		for _, parent := range record.interfaces {
			if !visit(parent) {
				return false
			}
		}
		states[owner] = 2
		return true
	}
	return visit(actual) && found
}
