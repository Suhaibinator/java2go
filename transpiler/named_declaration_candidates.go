package transpiler

import (
	"bytes"
	"os"
	"strings"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
)

// Exact source attestation must be cheaper than the live scans it replaces.
// Larger files conservatively retire this optional optimization for the entire
// invocation; the original lookup tiers remain the authority on that path.
const declarationSourceAttestationLimit = 256

// These tables contain direct, ordered declaration candidates only. They never
// answer lexical binding, ancestry, accessibility, imports or admission.
type namedDeclarationRegion struct {
	file     *symbol.FileScope
	first    map[string]*symbol.ClassScope
	complete bool
}

type namedDeclarationPackage struct {
	scope     *symbol.PackageScope
	first     map[string]*symbol.ClassScope
	duplicate map[string]bool
	complete  bool
}

type namedDeclarationCandidates struct {
	invocation *syntaxInvocation
	session    *cliownership.Session
	graph      *symbol.GlobalSymbols
	stdout     *os.File
	stderr     *os.File
	logger     *log.Logger
	formatter  log.Formatter
	retired    bool
	files      map[*symbol.FileScope]namedDeclarationRegion
	owners     map[*symbol.ClassScope]namedDeclarationRegion
	packages   map[string]namedDeclarationPackage
}

func (facts *namedDeclarationCandidates) retire() {
	if facts != nil {
		facts.retired = true
		facts.files, facts.owners, facts.packages = nil, nil, nil
	}
}

// A reset may be observed through an alias whose holder predates attachment.
// Retire the retained invocation too, without retiring a fresh current owner.
func retireStaleDeclarationInvocation(ctx Ctx) {
	if invocation := ctx.syntaxInvocation; invocation != nil && invocation.graph != symbol.GlobalScope {
		invocation.declarations.retire()
	}
}

func (facts *namedDeclarationCandidates) usable() bool {
	if facts == nil || facts.retired {
		return false
	}
	invocation := facts.invocation
	if activeResolutionFiles == nil || activeResolutionFiles.declarations != facts ||
		activeResolutionFiles.graph != facts.graph || invocation == nil || !invocation.active || !invocation.indexable ||
		invocation.declarations != facts || invocation.session != facts.session ||
		!facts.session.Active() || invocation.graph != facts.graph || facts.graph != symbol.GlobalScope ||
		os.Stdout != facts.stdout || os.Stderr != facts.stderr || !standaloneSyntaxConfigurationOK(os.Stdout) ||
		log.StandardLogger() != facts.logger || facts.logger.Formatter != facts.formatter {
		facts.retire()
		return false
	}
	return true
}

// Checked files live only for this callback-free accessor. Reusing one exact
// check for pointer-identical active/context/queried files never crosses a call,
// render, invocation or session boundary.
type namedDeclarationAccess struct {
	facts   *namedDeclarationCandidates
	checked [3]*symbol.FileScope
}

func (access *namedDeclarationAccess) currentFile(file *symbol.FileScope) bool {
	if file == nil || access.facts == nil {
		return false
	}
	for _, checked := range access.checked {
		if checked == file {
			return true
		}
	}
	if !access.facts.currentFile(file) {
		return false
	}
	for index, checked := range access.checked {
		if checked == nil {
			access.checked[index] = file
			break
		}
	}
	return true
}

func declarationCandidates(ctx ...Ctx) *namedDeclarationCandidates {
	return declarationAccess(ctx...).facts
}

func declarationAccess(ctx ...Ctx) namedDeclarationAccess {
	// The currently executing render owns authority. A retained outer context
	// cannot bypass a nested public render or another owned invocation.
	if activeResolutionFiles == nil {
		return namedDeclarationAccess{}
	}
	facts := activeResolutionFiles.declarations
	if facts == nil || !facts.usable() {
		return namedDeclarationAccess{}
	}
	access := namedDeclarationAccess{facts: facts}
	if file := activeResolutionFiles.renderFile; file != nil && !access.currentFile(file) {
		return namedDeclarationAccess{}
	}
	if len(ctx) > 0 && ctx[0].syntaxInvocation != nil && ctx[0].syntaxInvocation != facts.invocation {
		return namedDeclarationAccess{}
	}
	if len(ctx) > 0 && ctx[0].currentFile != nil && !access.currentFile(ctx[0].currentFile) {
		return namedDeclarationAccess{}
	}
	return access
}

// Observe the exact original parse anchor and copied source before using a
// queried file/owner, including cached absence. Observed replacement retires
// the entire batch permanently; restoring the old source cannot revive it.
func (facts *namedDeclarationCandidates) currentFile(file *symbol.FileScope) bool {
	if file == nil {
		return false
	}
	if len(file.Source) > declarationSourceAttestationLimit {
		facts.retire()
		return false
	}
	syntax := facts.invocation.identifiers[file]
	if syntax == nil {
		return false
	}
	if file.ParsedRoot() != syntax.anchor || !bytes.Equal(file.Source, syntax.source) {
		facts.retire()
		return false
	}
	return true
}

// Construction uses the exact node pointers already observed by the syntax
// walk. No ancestry proof, source-name guess or additional native walk is used.
func newNamedDeclarationCandidates(invocation *syntaxInvocation) *namedDeclarationCandidates {
	if invocation == nil || !invocation.active || !invocation.indexable || !invocation.session.Active() ||
		invocation.graph != symbol.GlobalScope || !standaloneSyntaxConfigurationOK(os.Stdout) {
		return nil
	}
	facts := &namedDeclarationCandidates{invocation: invocation, session: invocation.session, graph: invocation.graph,
		stdout: os.Stdout, stderr: os.Stderr, logger: log.StandardLogger(), formatter: log.StandardLogger().Formatter}
	// Inspect live registered source lengths before creating or reading any
	// declaration table. Admission never retries within this invocation.
	for file := range invocation.files {
		if file != nil && len(file.Source) > declarationSourceAttestationLimit {
			facts.retire()
			return facts
		}
	}
	facts.files = make(map[*symbol.FileScope]namedDeclarationRegion)
	facts.owners = make(map[*symbol.ClassScope]namedDeclarationRegion)
	facts.packages = make(map[string]namedDeclarationPackage)
	for packageName, pkg := range facts.graph.Packages {
		entry := namedDeclarationPackage{scope: pkg, first: make(map[string]*symbol.ClassScope), duplicate: make(map[string]bool), complete: pkg != nil}
		if pkg != nil {
			for _, file := range pkg.Files {
				region := namedDeclarationRegion{complete: false}
				if file != nil {
					if _, duplicateFile := facts.files[file]; duplicateFile {
						entry.complete = false
					} else {
						region = facts.buildFile(file, packageName)
						facts.files[file] = region
					}
				}
				entry.complete = entry.complete && region.complete
				for name, owner := range region.first {
					if _, duplicate := entry.first[name]; duplicate {
						entry.duplicate[name] = true
					} else {
						entry.first[name] = owner
					}
				}
			}
		}
		facts.packages[packageName] = entry
	}
	return facts
}

func (facts *namedDeclarationCandidates) buildFile(file *symbol.FileScope, packageName string) namedDeclarationRegion {
	region := namedDeclarationRegion{file: file, first: make(map[string]*symbol.ClassScope)}
	owned, known := facts.invocation.files[file]
	syntax := facts.invocation.identifiers[file]
	if !known || owned.Symbols != file || syntax == nil || syntax.anchor != file.ParsedRoot() ||
		file.Package != packageName || !bytes.Equal(syntax.source, file.Source) {
		return region
	}
	region.complete = true
	for _, top := range file.TopLevelClasses {
		complete := facts.buildOwner(top, nil, file, syntax)
		region.complete = region.complete && complete
		if top != nil && top.Class != nil {
			if _, exists := region.first[top.Class.OriginalName]; !exists {
				region.first[top.Class.OriginalName] = top
			}
		}
	}
	return region
}

func (facts *namedDeclarationCandidates) buildOwner(owner, enclosing *symbol.ClassScope, file *symbol.FileScope, syntax *syntaxIdentifierFile) bool {
	if owner == nil {
		return false
	}
	if owner.Class == nil || owner.Enclosing != enclosing || owner.Class.DeclarationNode == nil ||
		syntax.declarations[owner.Class.DeclarationNode] == "" {
		facts.owners[owner] = namedDeclarationRegion{file: file}
		return false
	}
	if _, repeated := facts.owners[owner]; repeated {
		facts.owners[owner] = namedDeclarationRegion{file: file}
		return false
	}
	region := namedDeclarationRegion{file: file, first: make(map[string]*symbol.ClassScope), complete: true}
	facts.owners[owner] = namedDeclarationRegion{} // Cycles/aliased declarations are unknown.
	for _, child := range owner.Subclasses {
		complete := facts.buildOwner(child, owner, file, syntax)
		region.complete = region.complete && complete
		if child != nil && child.Class != nil {
			if _, exists := region.first[child.Class.OriginalName]; !exists {
				region.first[child.Class.OriginalName] = child
			}
		}
	}
	facts.owners[owner] = region
	return region.complete
}

func indexedDirectDeclaration(owner *symbol.ClassScope, name string, ctx Ctx) (*symbol.ClassScope, bool) {
	access := declarationAccess(ctx)
	facts := access.facts
	if facts == nil {
		return nil, false
	}
	region, known := facts.owners[owner]
	if !known || !access.currentFile(region.file) || !region.complete {
		return nil, false
	}
	return region.first[name], true
}

func indexedFileDeclaration(file *symbol.FileScope, name string, ctx Ctx) (*symbol.ClassScope, bool) {
	access := declarationAccess(ctx)
	facts := access.facts
	if facts == nil {
		return nil, false
	}
	region, known := facts.files[file]
	if !known || !access.currentFile(region.file) || !region.complete {
		return nil, false
	}
	return region.first[name], true
}

func indexedPackageDeclaration(pkg *symbol.PackageScope, name string, ctx Ctx) (*symbol.ClassScope, bool) {
	access := declarationAccess(ctx)
	facts := access.facts
	if facts == nil || ctx.currentFile == nil {
		return nil, false
	}
	entry, known := facts.packages[ctx.currentFile.Package]
	if known && entry.scope != pkg {
		facts.retire()
		return nil, false
	}
	if !known || !entry.complete || entry.duplicate[name] {
		return nil, false
	}
	owner := entry.first[name]
	if owner != nil && !access.currentFile(facts.owners[owner].file) {
		return nil, false
	}
	return owner, true
}

func indexedQualifiedDeclaration(name string) (*symbol.ClassScope, bool) {
	access := declarationAccess()
	facts := access.facts
	if facts == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	for split := strings.LastIndexByte(name, '.'); split >= 0; split = strings.LastIndexByte(name[:split], '.') {
		packageName := name[:split]
		pkg := symbol.GlobalScope.FindPackage(packageName)
		entry, known := facts.packages[packageName]
		if pkg == nil {
			if known && entry.scope != nil {
				facts.retire()
				return nil, false
			}
			continue
		}
		if !known || entry.scope != pkg {
			facts.retire()
			return nil, false
		}
		if !entry.complete {
			return nil, false
		}
		root, suffix, nested := strings.Cut(name[split+1:], ".")
		if entry.duplicate[root] {
			return nil, false
		}
		owner := entry.first[root]
		if owner == nil {
			continue
		}
		if !access.currentFile(facts.owners[owner].file) {
			return nil, false
		}
		for nested {
			var member string
			member, suffix, nested = strings.Cut(suffix, ".")
			region := facts.owners[owner]
			if !region.complete {
				return nil, false
			}
			owner = region.first[member]
			if owner == nil {
				return nil, true // A found owner with a missing child terminates the path.
			}
		}
		return owner, true
	}
	return nil, true
}
