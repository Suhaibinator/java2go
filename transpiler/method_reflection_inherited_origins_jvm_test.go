package transpiler

import "testing"

func TestMethodReflectionInheritedDeclarationOriginsJDK21(t *testing.T) {
	methodReflectionProjectJDK21Main(t, "method_reflection_inherited_origins94", "origin.app.Main")
}
