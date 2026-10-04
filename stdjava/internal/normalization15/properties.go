// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package normalization15

import (
	"encoding/binary"
	"unicode/utf8"
)

const (
	headerLenMask   = 0x1F // extract the length value from the header byte (31 => 33)
	headerFlagsMask = 0xE0 // extract the qcInfo bits from the header byte
)

// Properties provides access to normalization properties of a rune.
type Properties struct {
	size  uint8  // length of UTF-8 encoding of this rune
	ccc   uint8  // leading canonical combining class (ccc if not decomposition)
	tccc  uint8  // trailing canonical combining class (ccc if not decomposition)
	nLead uint8  // number of leading non-starters.
	flags qcInfo // quick check flags
	index uint16
}

type qcInfo uint8

func (p Properties) isYesC() bool { return p.flags&0x10 == 0 }
func (p Properties) isYesD() bool { return p.flags&0x4 == 0 }

func (p Properties) combinesBackward() bool { return p.flags&0x8 != 0 } // == isMaybe

// Decomposition returns the decomposition for the underlying rune
// or nil if there is none.
func (p Properties) Decomposition() []byte {
	// TODO: create the decomposition for Hangul?
	if p.index == 0 {
		return nil
	}
	i := p.index
	n := decomps[i] & headerLenMask
	if n == 31 {
		n = 33
	}
	i++
	return decomps[i : i+uint16(n)]
}

// Size returns the length of UTF-8 encoding of the rune.
func (p Properties) Size() int {
	return int(p.size)
}

// CCC returns the canonical combining class of the underlying rune.
func (p Properties) CCC() uint8 {
	if p.index >= firstCCCZeroExcept {
		return 0
	}
	return ccc[p.ccc]
}

// LeadCCC returns the CCC of the first rune in the decomposition.
// If there is no decomposition, LeadCCC equals CCC.
func (p Properties) LeadCCC() uint8 {
	return ccc[p.ccc]
}

// TrailCCC returns the CCC of the last rune in the decomposition.
// If there is no decomposition, TrailCCC equals CCC.
func (p Properties) TrailCCC() uint8 {
	return ccc[p.tccc]
}

const recompShift = 21

func buildRecompMap() {
	recompMap = make(map[uint64]rune, len(recompMapPacked)/8)
	var buf [8]byte
	for i := 0; i < len(recompMapPacked); i += 8 {
		copy(buf[:], recompMapPacked[i:i+8])
		e := binary.BigEndian.Uint64(buf[:])
		recompMap[e>>recompShift] = rune(e & (1<<recompShift - 1))
	}
}

// combine returns the combined rune or 0 if it doesn't exist.
//
// The caller is responsible for calling
// recompMapOnce.Do(buildRecompMap) sometime before this is called.
func combine(a, b rune) rune {
	key := uint64(a)<<recompShift | uint64(b)
	if recompMap == nil {
		panic("caller error") // see func comment
	}
	return recompMap[key]
}

func compInfo(v uint16, sz int) Properties {
	if sz == 0 {
		return Properties{flags: 0x80, size: 1}
	}
	if v == 0 {
		return Properties{size: uint8(sz)}
	} else if v >= 0x8000 {
		p := Properties{
			size:  uint8(sz),
			ccc:   uint8(v),
			tccc:  uint8(v),
			flags: qcInfo(v>>8) & 0x3f,
		}
		if p.ccc > 0 || p.combinesBackward() {
			p.nLead = uint8(p.flags & 0x3)
		}
		return p
	}
	// has decomposition
	h := decomps[v]
	f := (qcInfo(h&headerFlagsMask) >> 2) | 0x4
	p := Properties{size: uint8(sz), flags: f, index: v}
	if v >= firstCCC {
		n := uint16(h & headerLenMask)
		if n == 31 {
			n = 33
		}
		v += n + 1
		c := decomps[v]
		p.tccc = c >> 2
		p.flags |= qcInfo(c & 0x3)
		if v >= firstLeadingCCC {
			p.nLead = c & 0x3
			if v >= firstStarterWithNLead {
				// We were tricked. Remove the decomposition.
				p.flags &= 0x03
				p.index = 0
				return p
			}
			p.ccc = decomps[v+1]
		}
	}
	return p
}

// properties uses the fixed Unicode15 trie; no host Unicode tables participate.
func properties(r rune, compatibility bool) Properties {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	if compatibility {
		return compInfo(nfkcData.lookup(buf[:n]))
	}
	return compInfo(nfcData.lookup(buf[:n]))
}
