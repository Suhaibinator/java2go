package symbol

import (
	"bytes"
	"fmt"
	"slices"
	"unicode/utf8"

	sitter "github.com/smacker/go-tree-sitter"
)

type SourceInputKind uint8

const (
	RegisteredMemorySourceInput SourceInputKind = iota
	PhysicalSourceInput
)

// SourceInputSnapshot retains immutable FileScope.Source bytes and their original
// input/offset domain. The source domain can differ from transformed ParseAST
// input. Binding an existing root does not prove arbitrary constructor bytes
// produced that root, or certify graph membership, declarations or source roles.
// Validate checks current inputs afresh; callers must not cache its success.
type SourceInputSnapshot struct {
	owner            *FileScope
	root             *sitter.Node
	name             string
	kind             SourceInputKind
	parser, original []byte
	offsets          []uint32
}

func NewSourceInputSnapshot(owner *FileScope, name string, kind SourceInputKind, parser, original []byte, offsets []uint32) *SourceInputSnapshot {
	raw := original
	if raw == nil {
		raw = parser
	}
	return &SourceInputSnapshot{
		owner:    owner,
		root:     owner.ParsedRoot(),
		name:     name,
		kind:     kind,
		parser:   slices.Clone(parser),
		original: slices.Clone(raw),
		offsets:  slices.Clone(offsets),
	}
}
func (f *FileScope) AttachSourceInputSnapshot(s *SourceInputSnapshot) {
	if f == nil || s == nil || s.owner != f || f.sourceInput != nil || s.root != f.ParsedRoot() {
		panic(fmt.Errorf("source cohort: invalid/rebound source attachment"))
	}
	f.sourceInput = s
}
func (f *FileScope) SourceInputSnapshot() *SourceInputSnapshot {
	if f == nil || f.sourceInput == nil || f.sourceInput.owner != f {
		return nil
	}
	return f.sourceInput
}
func (s *SourceInputSnapshot) Validate() error {
	if s == nil || s.owner == nil || s.root == nil || s.owner.ParsedRoot() != s.root || s.owner.sourceInput != s {
		return fmt.Errorf("source cohort: missing original parsed source snapshot")
	}
	if s.owner.SourceFilename != s.name {
		return fmt.Errorf("source cohort: registered filename changed")
	}
	if s.name == "" || !utf8.ValidString(s.name) || bytes.IndexByte([]byte(s.name), 0) >= 0 || s.kind > PhysicalSourceInput {
		return fmt.Errorf("source cohort: invalid registered source name/kind")
	}
	root := s.owner.ParsedRoot()
	if root.Type() != "program" || root.StartByte() > root.EndByte() || root.EndByte() > uint32(len(s.parser)) || !bytes.Equal(s.owner.Source, s.parser) {
		return fmt.Errorf("source cohort: stale parser source/root")
	}
	raw := s.owner.OriginalSource
	if raw == nil {
		raw = s.owner.Source
	}
	if !bytes.Equal(raw, s.original) || !slices.Equal(s.offsets, s.owner.SourceOffsets) || !utf8.Valid(s.original) || !utf8.Valid(s.parser) {
		return fmt.Errorf("source cohort: stale/invalid original source or offset map")
	}
	// Without an offset map there is no claimed byte-domain transformation.
	// A supplied map proves structural coverage only, not parse provenance.
	if len(s.offsets) == 0 && !bytes.Equal(s.parser, s.original) {
		return fmt.Errorf("source input: differing original source requires offsets")
	}
	if len(s.offsets) != 0 {
		if len(s.offsets) != len(s.parser)+1 || s.offsets[0] != 0 || s.offsets[len(s.offsets)-1] != uint32(len(s.original)) {
			return fmt.Errorf("source cohort: invalid original offset coverage")
		}
		for i, v := range s.offsets {
			if v > uint32(len(s.original)) || (i > 0 && v < s.offsets[i-1]) {
				return fmt.Errorf("source cohort: invalid original offsets")
			}
		}
	}
	return nil
}
