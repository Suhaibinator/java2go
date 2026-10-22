package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// registeredStaticIntrinsicDeclarationContext attests the existing registry
// Definition pointer and its unambiguous supported canonical owner. It grants no
// invocation applicability: actual arguments still use their caller context.
func registeredStaticIntrinsicDeclarationContext(def *symbol.Definition) (Ctx, bool) {
	if def == nil || !def.IsStatic || len(def.TypeParameters) == 0 {
		return Ctx{}, false
	}
	owner := ""
	for key, signatures := range staticIntrinsicImportSignatures {
		if key.methodName != def.OriginalName {
			continue
		}
		registered := false
		for _, signature := range signatures {
			registered = registered || signature == def
		}
		if !registered {
			continue
		}
		if _, callable := staticIntrinsics[key]; !callable {
			return Ctx{}, false
		}
		matchedOwner := false
		for _, family := range intrinsicOwners {
			for canonical, supported := range family {
				if supported && intrinsicOwnerKey(canonical) == key.typeName {
					if owner != "" && owner != canonical {
						return Ctx{}, false
					}
					owner = canonical
					matchedOwner = true
				}
			}
		}
		if !matchedOwner {
			return Ctx{}, false
		}
	}
	if owner == "" {
		return Ctx{}, false
	}
	return Ctx{localScope: def}, true
}
