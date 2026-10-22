package transpiler

import (
	"io"
	"os"
	"reflect"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
)

// RunStandalone is reserved for the built CLI's internal ownership capability.
// Source slices and parser trees must not be edited during this closed emission
// phase. Public Run and all extensible writer APIs retain the dynamic path.
func RunStandalone(args []string, session *cliownership.Session) error {
	defer session.Close()
	if !session.Active() || !standaloneSyntaxConfigurationOK(os.Stdout) {
		return Run(args)
	}
	return runInternalWithOwnership(args, os.Stdout, false, session)
}

func standaloneSyntaxConfigurationOK(stdout io.Writer) bool {
	output, native := stdout.(*os.File)
	if !native || output != os.Stdout {
		return false
	}
	logger := log.StandardLogger()
	logOutput, native := logger.Out.(*os.File)
	formatter, standard := logger.Formatter.(*log.TextFormatter)
	if !native || logOutput != os.Stderr || !standard || formatter == nil ||
		formatter.CallerPrettyfier != nil || logger.BufferPool != nil || logger.ExitFunc == nil ||
		reflect.ValueOf(logger.ExitFunc).Pointer() != reflect.ValueOf(os.Exit).Pointer() {
		return false
	}
	for _, hooks := range logger.Hooks {
		if len(hooks) != 0 {
			return false
		}
	}
	return true
}

// syntaxInvocation belongs to one emission batch, including a nested project
// batch. The process session never makes an arbitrary SourceFile immutable.
type syntaxInvocation struct {
	session     *cliownership.Session
	graph       *symbol.GlobalSymbols
	files       map[*symbol.FileScope]parsing.SourceFile
	active      bool
	identifiers map[*symbol.FileScope]*syntaxIdentifierFile
	indexable   bool
}

func newSyntaxInvocation(session *cliownership.Session, files []parsing.SourceFile) *syntaxInvocation {
	if !session.Active() || !standaloneSyntaxConfigurationOK(os.Stdout) {
		return nil
	}
	invocation := &syntaxInvocation{session: session, graph: symbol.GlobalScope, active: true,
		files: make(map[*symbol.FileScope]parsing.SourceFile, len(files))}
	for _, file := range files {
		if file.Symbols == nil || file.Ast == nil || file.Symbols.ParsedRoot() == nil ||
			!file.Ast.Equal(file.Symbols.ParsedRoot()) {
			return nil
		}
		invocation.files[file.Symbols] = file
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file != nil {
				if _, owned := invocation.files[file]; !owned {
					return nil
				}
			}
		}
	}
	invocation.identifiers = make(map[*symbol.FileScope]*syntaxIdentifierFile, len(invocation.files))
	invocation.indexable = true
	for file := range invocation.files {
		invocation.identifiers[file] = buildSyntaxIdentifierFile(file)
		if invocation.identifiers[file] == nil {
			invocation.indexable = false
		}
	}
	return invocation
}

func (invocation *syntaxInvocation) close() {
	if invocation != nil {
		invocation.active = false
	}
}
