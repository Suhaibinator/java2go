package stdjava

import "math/bits"

// BitSet stores independently mutable bit vectors for the supported Java
// constructor, single-bit operations, size queries and clone surface.
type BitSet struct {
	words  []uint64
	sticky bool
}

func NewBitSet(sizes ...int32) *BitSet {
	size := int32(64)
	if len(sizes) != 0 {
		size = sizes[0]
	}
	if size < 0 {
		panic(NewNegativeArraySizeException("negative BitSet size"))
	}
	return &BitSet{words: make([]uint64, (int64(size)+63)/64), sticky: len(sizes) != 0}
}
func (b *BitSet) Set(index int32) {
	if index < 0 {
		panic(NewIndexOutOfBoundsException("negative bit index"))
	}
	word := int(index) / 64
	if word >= len(b.words) {
		length := max(2*len(b.words), word+1)
		grown := make([]uint64, length)
		copy(grown, b.words)
		b.words = grown
		b.sticky = false
	}
	b.words[word] |= uint64(1) << (index % 64)
}
func (b *BitSet) Get(index int32) bool {
	if index < 0 {
		panic(NewIndexOutOfBoundsException("negative bit index"))
	}
	word := int(index) / 64
	return word < len(b.words) && b.words[word]&(uint64(1)<<(index%64)) != 0
}
func (b *BitSet) usedWords() int {
	for i := len(b.words) - 1; i >= 0; i-- {
		if b.words[i] != 0 {
			return i + 1
		}
	}
	return 0
}
func (b *BitSet) Clone() *BitSet {
	if !b.sticky {
		b.words = b.words[:b.usedWords()]
	}
	copyOfWords := make([]uint64, len(b.words))
	copy(copyOfWords, b.words)
	return &BitSet{words: copyOfWords, sticky: b.sticky}
}
func (b *BitSet) Length() int32 {
	words := b.usedWords()
	if words == 0 {
		return 0
	}
	return int32((words-1)*64 + bits.Len64(b.words[words-1]))
}
func (b *BitSet) Size() int32 { return int32(len(b.words) * 64) }
func (b *BitSet) Cardinality() int32 {
	var count int32
	for _, word := range b.words {
		count += int32(bits.OnesCount64(word))
	}
	return count
}
func (*BitSet) JavaDynamicTypeID() TypeID { return "BitSet" }
func init()                               { RegisterJavaType("BitSet", ObjectTypeID, CloneableTypeID, SerializableTypeID) }
