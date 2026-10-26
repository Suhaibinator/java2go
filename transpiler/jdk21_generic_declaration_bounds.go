package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// These facts describe generic declarations only. They do not add a parent,
// expose a JDK API, or replace a supplied source/executable binder.
type jdk21GenericDeclarationBoundRecord struct {
	certificate string
	classSHA    string
	parameters  []jdk21GenericFormalBound
}

type jdk21GenericFormalBound struct {
	declaration      *symbol.TypeParamDeclaration
	classBoundAbsent bool
	classBound       *declaredBoundRecipe
	interfaceBounds  []*declaredBoundRecipe
	known            bool
}

type declaredBoundRecipe struct {
	kind         string
	externalName string
	parameter    *symbol.TypeParamDeclaration
	arguments    []*declaredBoundRecipe
	variance     string
	bound        *declaredBoundRecipe
	arrayRank    int
}

func certifiedJDK21DeclarationBounds(owner string) (jdk21GenericDeclarationBoundRecord, bool) {
	return certifiedJDK21DeclarationBoundsWithRecords(owner, jdk21GenericDeclarationBounds, jdk21GenericDeclarationBoundCertificates)
}

func certifiedJDK21DeclarationBoundsWithRecords(owner string, records map[string]jdk21GenericDeclarationBoundRecord, certificates map[string]string) (jdk21GenericDeclarationBoundRecord, bool) {
	record, ok := records[owner]
	if !ok || record.classSHA == "" || record.classSHA != jdk21GenericDeclarationBoundClassSHAs[owner] || record.certificate == "" || record.certificate != certificates[owner] || !jdk21RawReferenceCanonicalOwner(owner) {
		return jdk21GenericDeclarationBoundRecord{}, false
	}
	// Certificates identify immutable generated declarations, not caller-created
	// replacement slots or recipes that happen to carry the same certificate.
	attested, present := jdk21GenericDeclarationBounds[owner]
	if !present || record.certificate != attested.certificate || record.classSHA != attested.classSHA || len(record.parameters) != len(attested.parameters) {
		return jdk21GenericDeclarationBoundRecord{}, false
	}
	seen := map[*symbol.TypeParamDeclaration]bool{}
	for i, p := range record.parameters {
		proof := attested.parameters[i]
		if p.declaration != proof.declaration || p.classBoundAbsent != proof.classBoundAbsent || p.classBound != proof.classBound || p.known != proof.known || len(p.interfaceBounds) != len(proof.interfaceBounds) {
			return jdk21GenericDeclarationBoundRecord{}, false
		}
		for j, bound := range p.interfaceBounds {
			if bound != proof.interfaceBounds[j] {
				return jdk21GenericDeclarationBoundRecord{}, false
			}
		}
		if p.declaration == nil || seen[p.declaration] {
			return jdk21GenericDeclarationBoundRecord{}, false
		}
		seen[p.declaration] = true
	}
	if parent, present := jdk21GenericParentRecords[owner]; present {
		if parent.certificate == "" || parent.certificate != jdk21GenericParentCertificates[owner] || len(parent.parameters) != len(record.parameters) {
			return jdk21GenericDeclarationBoundRecord{}, false
		}
		for i, p := range parent.parameters {
			if p != record.parameters[i].declaration {
				return jdk21GenericDeclarationBoundRecord{}, false
			}
		}
	}
	return record, true
}
