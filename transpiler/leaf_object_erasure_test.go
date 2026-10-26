package transpiler

import "testing"

func TestLeafObjectErasurePlanIsAtomic(t *testing.T) {
	for _, test := range []struct {
		name, member, extra string
		want                bool
	}{
		{name: "bare members", want: true},
		{name: "nested field", member: "java.util.List<T> nested;"},
		{name: "array field", member: "T[] array;"},
		{name: "nested method body", member: "Object nested(){return new Box<T>(value);}"},
		{name: "private member descriptor", member: "private T hidden(){return value;}"},
		{name: "shadowing generic method", member: "<T> T shadow(T input){return input;}"},
		{name: "subclass", extra: "class Child extends Box<String>{Child(){super(\"x\");}}"},
		{name: "anonymous subclass", extra: "class Use{Object make(){return new Box<String>(\"x\"){};}}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, `class Box<T>{T value;Box(T value){this.value=value;}T read(){return value;}void write(T value){this.value=value;}`+test.member+`}`+test.extra)
			owner := helper.File.Symbols.FindClassScope("Box")
			if got := leafObjectMemberErasureEligible(owner, helper.Ctx); got != test.want {
				t.Fatalf("leaf plan = %v, want %v", got, test.want)
			}
			for _, field := range owner.Fields {
				if field.OriginalName == "value" {
					_, got := directOwnerInterfaceErasure(owner, field, helper.Ctx)
					if got != test.want {
						t.Fatalf("field migrated without complete plan: %v, want %v", got, test.want)
					}
				}
			}
			for _, method := range owner.Methods {
				if method.OriginalName == "read" {
					_, got := directOwnerOrdinaryMethodInterfaceErasure(owner, method, helper.Ctx)
					if got != test.want {
						t.Fatalf("method migrated without complete plan: %v, want %v", got, test.want)
					}
				}
			}
		})
	}
}
