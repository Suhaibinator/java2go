package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDirectOwnerTypeParameterErasure_FirstBoundRules(t *testing.T) {
	tests := []struct {
		name       string
		parameters func() []symbol.TypeParam
		target     int
		want       string
	}{
		{
			name: "unbounded erases to Object",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", nil)}
			},
			want: "Object",
		},
		{
			name: "interface first bound",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Numbered"}})}
			},
			want: "Numbered",
		},
		{
			name: "concrete first bound",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Base"}})}
			},
			want: "Base",
		},
		{
			name: "dependent bound follows declaration identity",
			parameters: func() []symbol.TypeParam {
				parameters := []symbol.TypeParam{
					symbol.NewTypeParam("B", []symbol.JavaType{{Original: "Root"}}),
					symbol.NewTypeParam("T", []symbol.JavaType{{Original: "B"}}),
				}
				symbol.BindTypeParameterBounds(parameters, parameters)
				return parameters
			},
			target: 1,
			want:   "Root",
		},
		{
			name: "intersection uses only first bound erasure",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{
					{Original: "Base"},
					{Original: "Ranked"},
				})}
			},
			want: "Base",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parameters := test.parameters()
			target := parameters[test.target]
			owner := &symbol.ClassScope{TypeParameters: parameters}
			definition := &symbol.Definition{
				OriginalType:        target.Name,
				DirectTypeParameter: target.Declaration,
			}

			got, ok := directOwnerTypeParameterErasure(owner, definition)
			if !ok {
				t.Fatal("direct owner type parameter was not recognized")
			}
			if got != test.want {
				t.Fatalf("erasure = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDirectOwnerTypeParameterErasure_RequiresBareRankZeroUse(t *testing.T) {
	parameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Root"}})
	owner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{parameter}}

	tests := []struct {
		name     string
		javaType string
		direct   *symbol.TypeParamDeclaration
		wantOK   bool
	}{
		{name: "bare", javaType: "T", direct: parameter.Declaration, wantOK: true},
		{name: "array", javaType: "T[]", direct: parameter.Declaration},
		{name: "nested", javaType: "Box<T>", direct: parameter.Declaration},
		{name: "qualified spelling", javaType: "pkg.T", direct: parameter.Declaration},
		{name: "missing provenance", javaType: "T"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ok := directOwnerTypeParameterErasure(owner, &symbol.Definition{
				OriginalType:        test.javaType,
				DirectTypeParameter: test.direct,
			})
			if ok != test.wantOK {
				t.Fatalf("recognized = %t, want %t", ok, test.wantOK)
			}
		})
	}
}

func TestDirectOwnerTypeParameterErasure_ClassParameterBeatsNameShadowing(t *testing.T) {
	classT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "ClassRoot"}})
	methodT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "MethodRoot"}})
	owner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{classT}}

	classDefinition := &symbol.Definition{
		OriginalType:        "T",
		DirectTypeParameter: classT.Declaration,
	}
	if got, ok := directOwnerTypeParameterErasure(owner, classDefinition); !ok || got != "ClassRoot" {
		t.Fatalf("class-owned T erasure = %q, %t; want ClassRoot, true", got, ok)
	}

	shadowingMethodDefinition := &symbol.Definition{
		OriginalType:        "T",
		DirectTypeParameter: methodT.Declaration,
	}
	if got, ok := directOwnerTypeParameterErasure(owner, shadowingMethodDefinition); ok {
		t.Fatalf("same-named method T was treated as owner T with erasure %q", got)
	}

	legacyOwner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{{
		Name:   "T",
		Bounds: []symbol.JavaType{{Original: "LegacyRoot"}},
	}}}
	if got, ok := directOwnerTypeParameterErasure(legacyOwner, &symbol.Definition{OriginalType: "T"}); ok {
		t.Fatalf("identity-free legacy T was treated as proven owner T with erasure %q", got)
	}
}
