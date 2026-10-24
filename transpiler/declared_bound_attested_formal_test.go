package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestDeclaredBoundAttestedFormalRecord(t *testing.T) {
	for _, owner := range []string{"java.lang.Class", "java.lang.reflect.TypeVariable"} {
		t.Run(owner, func(t *testing.T) {
			r, ok := certifiedJDK21DeclarationBounds(owner)
			if !ok || len(r.parameters) != 1 {
				t.Fatal("actual generated declaration missing")
			}
			for _, change := range []string{"slot", "bound"} {
				copy := r
				copy.parameters = append([]jdk21GenericFormalBound{}, r.parameters...)
				if change == "slot" {
					copy.parameters[0].declaration = &symbol.TypeParamDeclaration{SourceName: r.parameters[0].declaration.SourceName}
				}
				if change == "bound" {
					copy.parameters[0].classBoundAbsent = false
					copy.parameters[0].classBound = &declaredBoundRecipe{kind: "nominal", externalName: "java.lang.String"}
				}
				if _, ok := certifiedJDK21DeclarationBoundsWithRecords(owner, map[string]jdk21GenericDeclarationBoundRecord{owner: copy}, jdk21GenericDeclarationBoundCertificates); ok {
					t.Errorf("detached %s accepted", change)
				}
			}
		})
	}
}
