package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Walk every suffix segment with the ordinary member resolver. Empty segments
// remain significant, and each step retains the caller's declaration context.
func directMemberTypeSuffixPath(owner *symbol.ClassScope, suffix string, ctx Ctx) *symbol.ClassScope {
	for {
		part, remaining, more := strings.Cut(suffix, ".")
		owner = memberTypeInHierarchy(owner, part, ctx)
		if owner == nil || !more {
			return owner
		}
		suffix = remaining
	}
}
