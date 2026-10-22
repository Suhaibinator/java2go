package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

type sourceOriginAncestryEdge struct {
	javaType string
	scope    *symbol.ClassScope
}

// Prove the nominal paths before completing generic origins. Completing an
// inherited member's enclosing slots can itself ask for ancestry bindings;
// a lexical enclosing class is not thereby an ancestor of the member class.
func sourceOriginAncestryPaths(start, owner *symbol.ClassScope, ctx Ctx) (map[*symbol.ClassScope][]sourceOriginAncestryEdge, bool) {
	if start == nil || owner == nil {
		return nil, false
	}
	if start == owner {
		return nil, true
	}
	edges := map[*symbol.ClassScope][]sourceOriginAncestryEdge{}
	seen := map[*symbol.ClassScope]bool{}
	queue := []*symbol.ClassScope{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || seen[current] || current == owner {
			continue
		}
		seen[current] = true
		file := findFileScopeForClassScope(current, ctx)
		if file == nil {
			// A late scope needs the existing live lexical/tree certificate.
			// The caller's file alone cannot own an unrelated declaration.
			if !intrinsicLexicalDeclaringFile(current, ctx.currentFile, ctx) {
				continue
			}
			file = ctx.currentFile
		}
		declaring := classHeaderTypeCtx(current, ctx)
		declaring.currentFile = file
		declaring.syntheticTypeParameters = nil
		parameters := symbol.VisibleTypeParamBindings(current.TypeParameters)
		for _, written := range append([]string{current.Superclass}, current.ImplementedInterfaces...) {
			if strings.TrimSpace(written) == "" {
				continue
			}
			base, _ := parseJavaTypeString(written)
			if parameters[base] != nil {
				continue
			}
			parent := resolveClassScopeByQualifiedName(declaring, base)
			if parent != nil {
				edges[current] = append(edges[current], sourceOriginAncestryEdge{javaType: written, scope: parent})
				queue = append(queue, parent)
			}
		}
	}
	// Reachability is declaration identity only; no argument, bound or capture
	// origin is needed to close cycles or discard branches outside the query.
	reaches := map[*symbol.ClassScope]bool{owner: true}
	for changed := true; changed; {
		changed = false
		for current, parents := range edges {
			if reaches[current] {
				continue
			}
			for _, parent := range parents {
				if reaches[parent.scope] {
					reaches[current], changed = true, true
					break
				}
			}
		}
	}
	if !reaches[start] {
		return nil, false
	}
	for current, parents := range edges {
		var selected []sourceOriginAncestryEdge
		for _, parent := range parents {
			if reaches[parent.scope] {
				selected = append(selected, parent)
			}
		}
		edges[current] = selected
	}
	return edges, true
}

func sourceOriginAncestryComplete(origin inferredJavaTypeOrigin) bool {
	if !completeJavaTypeOrigin(origin) {
		return false
	}
	for _, argument := range origin.arguments {
		if !sourceOriginAncestryComplete(argument) {
			return false
		}
	}
	return true
}
