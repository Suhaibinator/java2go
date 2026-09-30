package transpiler

import (
	"reflect"
	"testing"
)

func TestParseJavaTypeString_MemberSegmentsPreserveBaseAndArgumentOrder(t *testing.T) {
	tests := []struct {
		javaType string
		wantBase string
		wantArgs []string
	}{
		{
			javaType: "Outer<String>.Child<Impl>",
			wantBase: "Outer.Child",
			wantArgs: []string{"String", "Impl"},
		},
		{
			javaType: "pkg.Outer<Map<String, List<Integer>>>.Child<Impl>",
			wantBase: "pkg.Outer.Child",
			wantArgs: []string{"Map<String, List<Integer>>", "Impl"},
		},
		{
			javaType: "Map<String, List<Integer>>",
			wantBase: "Map",
			wantArgs: []string{"String", "List<Integer>"},
		},
		{
			javaType: "Plain",
			wantBase: "Plain",
			wantArgs: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.javaType, func(t *testing.T) {
			base, arguments := parseJavaTypeString(test.javaType)
			if base != test.wantBase {
				t.Fatalf("base = %q, want %q", base, test.wantBase)
			}
			if !reflect.DeepEqual(arguments, test.wantArgs) {
				t.Fatalf("arguments = %#v, want %#v", arguments, test.wantArgs)
			}
		})
	}
}

func TestParseJavaTypeString_UnbalancedMemberGenericDoesNotReturnPartialArguments(t *testing.T) {
	base, arguments := parseJavaTypeString("Outer<String>.Child<Impl")
	if base != "Outer<String>.Child<Impl" || arguments != nil {
		t.Fatalf("unbalanced parse = %q, %#v", base, arguments)
	}
}
