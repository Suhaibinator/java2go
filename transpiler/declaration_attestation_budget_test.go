package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
)

// These probes use only APIs shared with PUBLIC198. A declaration rename is a
// deterministic path probe after an over-budget invocation must have selected
// live fallback. It is not a claim that admitted immutable topology is mutable.
func attestationBudgetSource199(size int, name string) string {
	source := "package p; class " + name + " {static class Member {}}"
	return source + "/*" + strings.Repeat("x", size-len(source)-4) + "*/"
}

func attestationBudgetOwned199(t *testing.T, sources []string, run func(*qualifiedLookup195Fixture, Ctx)) {
	t.Helper()
	f := qualifiedLookup195Parsed(t, sources)
	session := cliownership.New()
	defer session.Close()
	invocation := newSyntaxInvocation(session, f.files)
	if invocation == nil {
		t.Fatal("actual parsed owned invocation unavailable")
	}
	defer invocation.close()
	ctx := f.ctx.Clone()
	ctx.syntaxInvocation = invocation
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	previous := activeResolutionFiles
	activeResolutionFiles = sourceOwnershipIndex(ctx)
	defer func() { activeResolutionFiles = previous }()
	run(f, ctx)
}

func attestationBudgetLiveProbe199(t *testing.T, f *qualifiedLookup195Fixture, ctx Ctx, label string) {
	t.Helper()
	file, owner := f.files[0].Symbols, f.files[0].Symbols.BaseClass
	old := owner.Class.OriginalName
	owner.Class.OriginalName = old + "Renamed"
	// All APIs must re-enter original live scans once the index is ineligible.
	if got := declaredFileType(owner.Class.OriginalName, file, ctx); got != owner {
		t.Errorf("ATTESTATION_BUDGET_LIVE_FALLBACK: %s file renamed got%p want%p", label, got, owner)
	}
	if got := declaredFileType(old, file, ctx); got != nil {
		t.Errorf("ATTESTATION_BUDGET_LIVE_FALLBACK: %s file old name retained%p", label, got)
	}
	if got := findQualifiedSourceClass("p." + owner.Class.OriginalName); got != owner {
		t.Errorf("ATTESTATION_BUDGET_LIVE_FALLBACK: %s qualified renamed got%p want%p", label, got, owner)
	}
	if got := resolveClassScopeByQualifiedName(ctx, owner.Class.OriginalName+".Member"); got != owner.Subclasses[0] {
		t.Errorf("ATTESTATION_BUDGET_LIVE_FALLBACK: %s resolver renamed path got%p want%p", label, got, owner.Subclasses[0])
	}
	member := owner.Subclasses[0]
	member.Class.OriginalName = "MemberRenamed"
	if got := memberTypeInHierarchy(owner, "MemberRenamed", ctx); got != member {
		t.Errorf("ATTESTATION_BUDGET_LIVE_FALLBACK: %s direct renamed got%p want%p", label, got, member)
	}
	if got := memberTypeInHierarchy(owner, "Member", ctx); got != nil {
		t.Errorf("ATTESTATION_BUDGET_LIVE_FALLBACK: %s direct old name retained%p", label, got)
	}
}

func TestDeclarationAttestationBudget199PortableRed(t *testing.T) {
	for _, size := range []int{257, 1019, 30281, 105458, 121911} {
		t.Run(fmtSizeBudget199(size), func(t *testing.T) {
			attestationBudgetOwned199(t, []string{attestationBudgetSource199(size, "Use")}, func(f *qualifiedLookup195Fixture, ctx Ctx) {
				attestationBudgetLiveProbe199(t, f, ctx, "initial-over-cap")
			})
		})
	}
	t.Run("exact-256-source-replacement-restore", func(t *testing.T) {
		attestationBudgetOwned199(t, []string{attestationBudgetSource199(256, "Use")}, func(f *qualifiedLookup195Fixture, ctx Ctx) {
			file, owner := f.files[0].Symbols, f.files[0].Symbols.BaseClass
			if got := declaredFileType("Use", file, ctx); got != owner {
				t.Fatal("256-byte initial lookup lost")
			}
			original := file.Source
			file.Source = append(append([]byte(nil), original...), ' ')
			if got := declaredFileType("Use", file, ctx); got != owner {
				t.Fatal("replacement fallback lost live declaration")
			}
			file.Source = original
			attestationBudgetLiveProbe199(t, f, ctx, "over-cap-observed-then-restore")
		})
	})
	for _, otherPackage := range []bool{false, true} {
		name := "mixed-same-package"
		large := attestationBudgetSource199(257, "Peer")
		if otherPackage {
			name = "mixed-other-package"
			large = strings.Replace(large, "package p;", "package q;", 1)
		}
		t.Run(name, func(t *testing.T) {
			attestationBudgetOwned199(t, []string{attestationBudgetSource199(256, "Use"), large}, func(f *qualifiedLookup195Fixture, ctx Ctx) {
				attestationBudgetLiveProbe199(t, f, ctx, name+"-small-active-first")
			})
		})
		t.Run(name+"-source-shortened-restored", func(t *testing.T) {
			attestationBudgetOwned199(t, []string{attestationBudgetSource199(256, "Use"), large}, func(f *qualifiedLookup195Fixture, ctx Ctx) {
				file := f.files[1].Symbols
				original := file.Source
				file.Source = []byte("package p; class Peer {}")
				if got := declaredFileType("Use", f.files[0].Symbols, ctx); got != f.files[0].Symbols.BaseClass {
					t.Fatal("shortened-source fallback lost small-file declaration")
				}
				// Restoration is not a fresh invocation or an admission retry.
				file.Source = original
				attestationBudgetLiveProbe199(t, f, ctx, name+"-restore")
			})
		})
	}
}

func fmtSizeBudget199(value int) string {
	const digits = "0123456789"
	var out [20]byte
	index := len(out)
	for value > 0 {
		index--
		out[index] = digits[value%10]
		value /= 10
	}
	return string(out[index:])
}
