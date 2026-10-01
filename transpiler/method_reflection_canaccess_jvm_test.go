package transpiler

import "testing"

func TestMethodReflectionCanAccessReceiversJDK21(t *testing.T) {
	methodReflectionProjectJDK21Main(t, "method_reflection_canaccess94", "access.app.Main")
}
