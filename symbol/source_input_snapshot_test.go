package symbol

import (
	"context"
	"runtime"
	"slices"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func snapshotOwner(t *testing.T, source []byte) *FileScope {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		t.Fatal(err)
	}
	file := ParseSymbols(tree.RootNode(), source)
	file.SourceFilename = "Owned.java"
	return file
}
func snapshotAttach(t *testing.T, file *FileScope) *SourceInputSnapshot {
	t.Helper()
	snapshot := NewSourceInputSnapshot(file, file.SourceFilename, RegisteredMemorySourceInput, file.Source, file.OriginalSource, file.SourceOffsets)
	file.AttachSourceInputSnapshot(snapshot)
	return snapshot
}
func snapshotValid(t *testing.T, snapshot *SourceInputSnapshot) {
	t.Helper()
	if err := snapshot.Validate(); err != nil {
		t.Fatal("expected current valid input:", err)
	}
}
func snapshotInvalid(t *testing.T, snapshot *SourceInputSnapshot) {
	t.Helper()
	if snapshot.Validate() == nil {
		t.Fatal("changed or invalid input validated")
	}
}
func snapshotPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("invalid attachment did not panic")
		}
	}()
	action()
}
func snapshotIdentityOffsets(n int) []uint32 {
	offsets := make([]uint32, n+1)
	for i := range offsets {
		offsets[i] = uint32(i)
	}
	return offsets
}

func TestSourceInputSnapshotOwnerAttachment(t *testing.T) {
	var absent *FileScope
	if absent.SourceInputSnapshot() != nil || (&FileScope{}).SourceInputSnapshot() != nil {
		t.Fatal("nil/synthetic file borrowed snapshot")
	}
	snapshotInvalid(t, (*SourceInputSnapshot)(nil))
	file := snapshotOwner(t, []byte("class Owned {}"))
	if file.SourceInputSnapshot() != nil {
		t.Fatal("direct symbol parsing attached an input implicitly")
	}
	snapshot := snapshotAttach(t, file)
	snapshotValid(t, snapshot)
	copied := *file
	if copied.SourceInputSnapshot() != nil {
		t.Fatal("FileScope copy borrowed original owner's attachment")
	}
	foreign := snapshotOwner(t, slices.Clone(file.Source))
	snapshotPanics(t, func() { foreign.AttachSourceInputSnapshot(snapshot) })
	if foreign.SourceInputSnapshot() != nil {
		t.Fatal("failed foreign attachment changed state")
	}
	snapshotPanics(t, func() { file.AttachSourceInputSnapshot(snapshot) })
	snapshotPanics(t, func() {
		file.AttachSourceInputSnapshot(NewSourceInputSnapshot(file, file.SourceFilename, 0, file.Source, nil, nil))
	})
	snapshotPanics(t, func() { absent.AttachSourceInputSnapshot(snapshot) })
	snapshotPanics(t, func() { foreign.AttachSourceInputSnapshot(nil) })
	saved := *file
	*file = *foreign
	snapshotInvalid(t, snapshot)
	*file = saved
	snapshotValid(t, snapshot)
	if file.SourceInputSnapshot() != snapshot {
		t.Fatal("restoration changed original attachment")
	}
}

func TestSourceInputSnapshotExactRoot(t *testing.T) {
	file := snapshotOwner(t, []byte("class Owned {}"))
	snapshot := snapshotAttach(t, file)
	root := file.ParsedRoot()
	twin := snapshotOwner(t, slices.Clone(file.Source)).ParsedRoot()
	copied := *root
	if !copied.Equal(root) {
		t.Fatal("copied node must be Node.Equal for this pointer-exact control")
	}
	for _, replacement := range []*sitter.Node{nil, twin, &copied} {
		file.parsedRoot = replacement
		snapshotInvalid(t, snapshot)
		file.parsedRoot = root
		snapshotValid(t, snapshot)
	}
	pending := snapshotOwner(t, []byte("class Pending {}"))
	before := NewSourceInputSnapshot(pending, pending.SourceFilename, 0, pending.Source, nil, nil)
	pendingRoot := pending.parsedRoot
	pending.parsedRoot = twin
	snapshotPanics(t, func() { pending.AttachSourceInputSnapshot(before) })
	if pending.sourceInput != nil {
		t.Fatal("failed stale-root attach changed state")
	}
	pending.parsedRoot = pendingRoot
	pending.AttachSourceInputSnapshot(before)
	snapshotValid(t, before)
}

func TestSourceInputSnapshotBytesAndRestoration(t *testing.T) {
	file := snapshotOwner(t, []byte("class Owned {}"))
	file.OriginalSource = slices.Clone(file.Source)
	file.SourceOffsets = snapshotIdentityOffsets(len(file.Source))
	snapshot := snapshotAttach(t, file)
	snapshotValid(t, snapshot)
	parser := slices.Clone(file.Source)
	original := slices.Clone(file.OriginalSource)
	offsets := slices.Clone(file.SourceOffsets)
	for _, mutate := range []func(){
		func() { file.Source[0] ^= 1 },
		func() { file.Source = append(file.Source, ' ') },
		func() { file.OriginalSource[0] ^= 1 },
		func() { file.OriginalSource = append(file.OriginalSource, ' ') },
		func() { file.SourceOffsets[1]++ },
		func() { file.SourceOffsets = nil },
		func() { file.SourceFilename = "Other.java" },
	} {
		mutate()
		snapshotInvalid(t, snapshot)
		file.Source = slices.Clone(parser)
		file.OriginalSource = slices.Clone(original)
		file.SourceOffsets = slices.Clone(offsets)
		file.SourceFilename = "Owned.java"
		snapshotValid(t, snapshot)
	}
	// Exact content replacement is allowed. Source-occurrence backing identity
	// and graph membership remain a future consumer's separate obligations.
	file.Source = slices.Clone(parser)
	snapshotValid(t, snapshot)
}

func TestSourceInputSnapshotDefensiveConstruction(t *testing.T) {
	input := []byte("class Owned {}")
	raw := slices.Clone(input)
	offsets := snapshotIdentityOffsets(len(input))
	file := snapshotOwner(t, slices.Clone(input))
	file.OriginalSource = slices.Clone(raw)
	file.SourceOffsets = slices.Clone(offsets)
	snapshot := NewSourceInputSnapshot(file, file.SourceFilename, 0, input, raw, offsets)
	file.AttachSourceInputSnapshot(snapshot)
	input[0] ^= 1
	raw[1] ^= 1
	offsets[1]++
	snapshotValid(t, snapshot)
}

func TestSourceInputSnapshotOffsetContracts(t *testing.T) {
	for _, raw := range [][]byte{[]byte("class Other {}"), []byte("class OtherLong {}")} {
		for _, noMap := range [][]uint32{nil, {}} {
			file := snapshotOwner(t, []byte("class Owned {}"))
			file.OriginalSource = slices.Clone(raw)
			file.SourceOffsets = noMap
			snapshotInvalid(t, snapshotAttach(t, file))
		}
	}
	for _, noMap := range [][]uint32{nil, {}} {
		file := snapshotOwner(t, []byte("class Owned {}"))
		file.OriginalSource = slices.Clone(file.Source)
		file.SourceOffsets = noMap
		snapshotValid(t, snapshotAttach(t, file))
	}
	for _, defect := range []string{"short", "start", "EOF", "bounds", "decreasing"} {
		file := snapshotOwner(t, []byte("class Owned {}"))
		file.OriginalSource = slices.Clone(file.Source)
		file.SourceOffsets = snapshotIdentityOffsets(len(file.Source))
		switch defect {
		case "short":
			file.SourceOffsets = file.SourceOffsets[:len(file.SourceOffsets)-1]
		case "start":
			file.SourceOffsets[0] = 1
		case "EOF":
			file.SourceOffsets[len(file.SourceOffsets)-1]--
		case "bounds":
			file.SourceOffsets[1] = uint32(len(file.Source) + 1)
		case "decreasing":
			file.SourceOffsets[3] = 0
		}
		snapshotInvalid(t, snapshotAttach(t, file))
	}
}

func TestSourceInputSnapshotSourceUTF8(t *testing.T) {
	for _, which := range []string{"parser", "original"} {
		t.Run(which, func(t *testing.T) {
			file := snapshotOwner(t, []byte("class Owned {}"))
			file.OriginalSource = slices.Clone(file.Source)
			file.SourceOffsets = snapshotIdentityOffsets(len(file.Source))
			snapshot := snapshotAttach(t, file)
			target := file.Source
			if which == "original" {
				target = file.OriginalSource
			}
			saved := target[0]
			target[0] = 0xff
			snapshotInvalid(t, snapshot)
			target[0] = saved
			snapshotValid(t, snapshot)
			// Fresh bad buffers with matching retained bytes must independently fail
			// UTF8 admission, rather than only tripping a postattachment byte mismatch.
			fresh := snapshotOwner(t, []byte("class Owned {}"))
			fresh.OriginalSource = slices.Clone(fresh.Source)
			fresh.SourceOffsets = snapshotIdentityOffsets(len(fresh.Source))
			if which == "parser" {
				fresh.Source[0] = 0xff
			} else {
				fresh.OriginalSource[0] = 0xff
			}
			snapshotInvalid(t, snapshotAttach(t, fresh))
		})
	}
}

func TestSourceInputSnapshotNamesKindsAndRootRange(t *testing.T) {
	for _, name := range []string{"", "bad\x00name", "bad\xffname"} {
		file := snapshotOwner(t, []byte("class Owned {}"))
		file.SourceFilename = name
		snapshotInvalid(t, snapshotAttach(t, file))
	}
	file := snapshotOwner(t, []byte("class Owned {}"))
	snapshot := NewSourceInputSnapshot(file, file.SourceFilename, SourceInputKind(255), file.Source, nil, nil)
	file.AttachSourceInputSnapshot(snapshot)
	snapshotInvalid(t, snapshot)
	short := snapshotOwner(t, []byte("class Owned {}"))
	short.Source = []byte("x")
	snapshotInvalid(t, snapshotAttach(t, short))
	child := snapshotOwner(t, []byte("class Owned {}"))
	child.parsedRoot = child.parsedRoot.NamedChild(0)
	snapshotInvalid(t, snapshotAttach(t, child))
	synthetic := &FileScope{SourceFilename: "Synthetic.java"}
	snapshotInvalid(t, snapshotAttach(t, synthetic))
	snapshotInvalid(t, NewSourceInputSnapshot(nil, "Missing.java", 0, nil, nil, nil))
}

func TestSourceInputSnapshotLifetimeAndAuthorityBoundary(t *testing.T) {
	snapshot := func() *SourceInputSnapshot {
		file := snapshotOwner(t, []byte("package owned;"))
		return snapshotAttach(t, file)
	}()
	runtime.GC()
	runtime.GC()
	snapshotValid(t, snapshot)
	previous := GlobalScope
	defer func() { GlobalScope = previous }()
	GlobalScope = &GlobalSymbols{Packages: map[string]*PackageScope{}}
	// An input snapshot remains valid without graph membership. This explicitly
	// cannot replace a resolver's live file/class/graph ownership checks.
	snapshotValid(t, snapshot)
	GlobalScope = &GlobalSymbols{Packages: map[string]*PackageScope{}}
	snapshotValid(t, snapshot)
	registered := snapshotOwner(t, []byte("package owned;class Owned {}"))
	input := snapshotAttach(t, registered)
	if GlobalScope.FindPackage("owned") != nil {
		t.Fatal("snapshot construction registered a graph member")
	}
	AddSymbolsToPackage(registered)
	pkg := GlobalScope.FindPackage("owned")
	key := registered.BaseClass.Class.Name
	if pkg.Files[key] != registered {
		t.Fatal("actual registration key does not own the original file")
	}
	if pkg.FindClass("Owned") != registered.BaseClass {
		t.Fatal("actual registration did not preserve declaration identity")
	}
	snapshotValid(t, input)
	delete(pkg.Files, key)
	if pkg.FindClass("Owned") != nil {
		t.Fatal("live graph removal was not observed")
	}
	snapshotValid(t, input)
	foreign := snapshotOwner(t, []byte("package owned;class Owned {}"))
	pkg.Files[key] = foreign
	if pkg.FindClass("Owned") != foreign.BaseClass || foreign.SourceInputSnapshot() != nil {
		t.Fatal("replacement file borrowed the old input snapshot")
	}
	snapshotValid(t, input)
}
