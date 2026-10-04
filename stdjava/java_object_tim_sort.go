/*
 * Copyright (c) 2009, 2013, Oracle and/or its affiliates. All rights reserved.
 * Copyright 2009 Google Inc.  All Rights Reserved.
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
 *
 * This code is free software; you can redistribute it and/or modify it
 * under the terms of the GNU General Public License version 2 only, as
 * published by the Free Software Foundation.  Oracle designates this
 * particular file as subject to the "Classpath" exception as provided
 * by Oracle in the LICENSE file that accompanied this code.
 *
 * This code is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License
 * version 2 for more details (a copy is included in the LICENSE file that
 * accompanied this code).
 *
 * You should have received a copy of the GNU General Public License version
 * 2 along with this work; if not, write to the Free Software Foundation,
 * Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1301 USA.
 *
 * Please contact Oracle, 500 Oracle Parkway, Redwood Shores, CA 94065 USA
 * or visit www.oracle.com if you need additional information or have any
 * questions.
 */

package stdjava

import "math/bits"

// Translated OpenJDK TimSort to Go on 2026-10-03.
// Go translation of OpenJDK jdk21u TimSort.java, jdk-21.0.6+7,
// commit 7069f193f1f8c61869fc68a36c17f3a9a7b7b2a0. See
// NOTICE.java-object-timsort and LICENSE.java-object-timsort.
// The comparator receives the operands read at the comparison site, including
// saved pivots and temporary-run elements. A panic leaves completed writes in
// the caller's array; it is neither intercepted nor followed by merge cleanup.
const javaTimMinMerge = 32
const javaTimMinGallop = 7

type javaObjectTimSort[E any] struct {
	a               []E
	compare         func(E, E) int32
	minGallop       int
	tmp             []E
	stackSize       int
	runBase, runLen []int
}

func sortJavaObjectArray[E any](a []E, compare func(E, E) int32) {
	n := len(a)
	if n < 2 {
		return
	}
	if n < javaTimMinMerge {
		run := javaTimCountRun(a, 0, n, compare)
		javaTimBinarySort(a, 0, n, run, compare)
		return
	}
	tlen := 256
	if n < 512 {
		tlen = n >> 1
	}
	stackLen := 49
	if n < 120 {
		stackLen = 5
	} else if n < 1542 {
		stackLen = 10
	} else if n < 119151 {
		stackLen = 24
	}
	ts := javaObjectTimSort[E]{a: a, compare: compare, minGallop: javaTimMinGallop, tmp: make([]E, tlen), runBase: make([]int, stackLen), runLen: make([]int, stackLen)}
	minRun := javaTimMinRun(n)
	lo, remaining := 0, n
	for remaining != 0 {
		run := javaTimCountRun(a, lo, n, compare)
		if run < minRun {
			force := minRun
			if remaining <= minRun {
				force = remaining
			}
			javaTimBinarySort(a, lo, lo+force, lo+run, compare)
			run = force
		}
		ts.runBase[ts.stackSize] = lo
		ts.runLen[ts.stackSize] = run
		ts.stackSize++
		ts.mergeCollapse()
		lo += run
		remaining -= run
	}
	ts.mergeForceCollapse()
}

func javaTimBinarySort[E any](a []E, lo, hi, start int, c func(E, E) int32) {
	if start == lo {
		start++
	}
	for ; start < hi; start++ {
		pivot := a[start]
		left, right := lo, start
		for left < right {
			mid := int((uint(left) + uint(right)) >> 1)
			if c(pivot, a[mid]) < 0 {
				right = mid
			} else {
				left = mid + 1
			}
		}
		switch start - left {
		case 2:
			a[left+2] = a[left+1]
			a[left+1] = a[left]
		case 1:
			a[left+1] = a[left]
		default:
			copy(a[left+1:start+1], a[left:start])
		}
		a[left] = pivot
	}
}

func javaTimCountRun[E any](a []E, lo, hi int, c func(E, E) int32) int {
	runHi := lo + 1
	if runHi == hi {
		return 1
	}
	descending := c(a[runHi], a[lo]) < 0
	runHi++
	if descending {
		for runHi < hi && c(a[runHi], a[runHi-1]) < 0 {
			runHi++
		}
		for left, right := lo, runHi-1; left < right; left, right = left+1, right-1 {
			v := a[left]
			a[left] = a[right]
			a[right] = v
		}
	} else {
		for runHi < hi && c(a[runHi], a[runHi-1]) >= 0 {
			runHi++
		}
	}
	return runHi - lo
}

func javaTimMinRun(n int) int {
	r := 0
	for n >= javaTimMinMerge {
		r |= n & 1
		n >>= 1
	}
	return n + r
}

func (ts *javaObjectTimSort[E]) mergeCollapse() {
	for ts.stackSize > 1 {
		n := ts.stackSize - 2
		if n > 0 && ts.runLen[n-1] <= ts.runLen[n]+ts.runLen[n+1] || n > 1 && ts.runLen[n-2] <= ts.runLen[n]+ts.runLen[n-1] {
			if ts.runLen[n-1] < ts.runLen[n+1] {
				n--
			}
		} else if n < 0 || ts.runLen[n] > ts.runLen[n+1] {
			break
		}
		ts.mergeAt(n)
	}
}
func (ts *javaObjectTimSort[E]) mergeForceCollapse() {
	for ts.stackSize > 1 {
		n := ts.stackSize - 2
		if n > 0 && ts.runLen[n-1] < ts.runLen[n+1] {
			n--
		}
		ts.mergeAt(n)
	}
}
func (ts *javaObjectTimSort[E]) mergeAt(i int) {
	base1, len1, base2, len2 := ts.runBase[i], ts.runLen[i], ts.runBase[i+1], ts.runLen[i+1]
	ts.runLen[i] = len1 + len2
	if i == ts.stackSize-3 {
		ts.runBase[i+1] = ts.runBase[i+2]
		ts.runLen[i+1] = ts.runLen[i+2]
	}
	ts.stackSize--
	k := javaTimGallopRight(ts.a[base2], ts.a, base1, len1, 0, ts.compare)
	base1 += k
	len1 -= k
	if len1 == 0 {
		return
	}
	len2 = javaTimGallopLeft(ts.a[base1+len1-1], ts.a, base2, len2, len2-1, ts.compare)
	if len2 == 0 {
		return
	}
	if len1 <= len2 {
		ts.mergeLo(base1, len1, base2, len2)
	} else {
		ts.mergeHi(base1, len1, base2, len2)
	}
}

// The source calculation uses a signed Java int; explicitly saturate when its
// doubled offset overflows rather than relying on the host's wider int.
func javaTimNextOffset(offset, maximum int) int {
	next := int32(uint32(offset)*2 + 1)
	if next <= 0 {
		return maximum
	}
	return int(next)
}

func javaTimGallopLeft[E any](key E, a []E, base, length, hint int, c func(E, E) int32) int {
	last, offset := 0, 1
	if c(key, a[base+hint]) > 0 {
		maximum := length - hint
		for offset < maximum && c(key, a[base+hint+offset]) > 0 {
			last = offset
			offset = javaTimNextOffset(offset, maximum)
		}
		if offset > maximum {
			offset = maximum
		}
		last += hint
		offset += hint
	} else {
		maximum := hint + 1
		for offset < maximum && c(key, a[base+hint-offset]) <= 0 {
			last = offset
			offset = javaTimNextOffset(offset, maximum)
		}
		if offset > maximum {
			offset = maximum
		}
		previous := last
		last = hint - offset
		offset = hint - previous
	}
	last++
	for last < offset {
		m := last + ((offset - last) >> 1)
		if c(key, a[base+m]) > 0 {
			last = m + 1
		} else {
			offset = m
		}
	}
	return offset
}
func javaTimGallopRight[E any](key E, a []E, base, length, hint int, c func(E, E) int32) int {
	offset, last := 1, 0
	if c(key, a[base+hint]) < 0 {
		maximum := hint + 1
		for offset < maximum && c(key, a[base+hint-offset]) < 0 {
			last = offset
			offset = javaTimNextOffset(offset, maximum)
		}
		if offset > maximum {
			offset = maximum
		}
		previous := last
		last = hint - offset
		offset = hint - previous
	} else {
		maximum := length - hint
		for offset < maximum && c(key, a[base+hint+offset]) >= 0 {
			last = offset
			offset = javaTimNextOffset(offset, maximum)
		}
		if offset > maximum {
			offset = maximum
		}
		last += hint
		offset += hint
	}
	last++
	for last < offset {
		m := last + ((offset - last) >> 1)
		if c(key, a[base+m]) < 0 {
			offset = m
		} else {
			last = m + 1
		}
	}
	return offset
}

func (ts *javaObjectTimSort[E]) mergeLo(base1, len1, base2, len2 int) {
	a := ts.a
	tmp := ts.ensureCapacity(len1)
	cursor1, cursor2, dest := 0, base2, base1
	copy(tmp[:len1], a[base1:base1+len1])
	a[dest] = a[cursor2]
	dest++
	cursor2++
	len2--
	if len2 == 0 {
		copy(a[dest:dest+len1], tmp[cursor1:cursor1+len1])
		return
	}
	if len1 == 1 {
		copy(a[dest:dest+len2], a[cursor2:cursor2+len2])
		a[dest+len2] = tmp[cursor1]
		return
	}
	c, minGallop := ts.compare, ts.minGallop
outer:
	for {
		count1, count2 := 0, 0
		for {
			if c(a[cursor2], tmp[cursor1]) < 0 {
				a[dest] = a[cursor2]
				dest++
				cursor2++
				count2++
				count1 = 0
				len2--
				if len2 == 0 {
					break outer
				}
			} else {
				a[dest] = tmp[cursor1]
				dest++
				cursor1++
				count1++
				count2 = 0
				len1--
				if len1 == 1 {
					break outer
				}
			}
			if count1|count2 >= minGallop {
				break
			}
		}
		for {
			count1 = javaTimGallopRight(a[cursor2], tmp, cursor1, len1, 0, c)
			if count1 != 0 {
				copy(a[dest:dest+count1], tmp[cursor1:cursor1+count1])
				dest += count1
				cursor1 += count1
				len1 -= count1
				if len1 <= 1 {
					break outer
				}
			}
			a[dest] = a[cursor2]
			dest++
			cursor2++
			len2--
			if len2 == 0 {
				break outer
			}
			count2 = javaTimGallopLeft(tmp[cursor1], a, cursor2, len2, 0, c)
			if count2 != 0 {
				copy(a[dest:dest+count2], a[cursor2:cursor2+count2])
				dest += count2
				cursor2 += count2
				len2 -= count2
				if len2 == 0 {
					break outer
				}
			}
			a[dest] = tmp[cursor1]
			dest++
			cursor1++
			len1--
			if len1 == 1 {
				break outer
			}
			minGallop--
			if count1 < javaTimMinGallop && count2 < javaTimMinGallop {
				break
			}
		}
		if minGallop < 0 {
			minGallop = 0
		}
		minGallop += 2
	}
	ts.minGallop = max(1, minGallop)
	switch len1 {
	case 1:
		copy(a[dest:dest+len2], a[cursor2:cursor2+len2])
		a[dest+len2] = tmp[cursor1]
	case 0:
		javaTimContractViolation()
	default:
		copy(a[dest:dest+len1], tmp[cursor1:cursor1+len1])
	}
}

func (ts *javaObjectTimSort[E]) mergeHi(base1, len1, base2, len2 int) {
	a := ts.a
	tmp := ts.ensureCapacity(len2)
	copy(tmp[:len2], a[base2:base2+len2])
	cursor1, cursor2, dest := base1+len1-1, len2-1, base2+len2-1
	a[dest] = a[cursor1]
	dest--
	cursor1--
	len1--
	if len1 == 0 {
		copy(a[dest-(len2-1):dest+1], tmp[:len2])
		return
	}
	if len2 == 1 {
		dest -= len1
		cursor1 -= len1
		copy(a[dest+1:dest+1+len1], a[cursor1+1:cursor1+1+len1])
		a[dest] = tmp[cursor2]
		return
	}
	c, minGallop := ts.compare, ts.minGallop
outer:
	for {
		count1, count2 := 0, 0
		for {
			if c(tmp[cursor2], a[cursor1]) < 0 {
				a[dest] = a[cursor1]
				dest--
				cursor1--
				count1++
				count2 = 0
				len1--
				if len1 == 0 {
					break outer
				}
			} else {
				a[dest] = tmp[cursor2]
				dest--
				cursor2--
				count2++
				count1 = 0
				len2--
				if len2 == 1 {
					break outer
				}
			}
			if count1|count2 >= minGallop {
				break
			}
		}
		for {
			count1 = len1 - javaTimGallopRight(tmp[cursor2], a, base1, len1, len1-1, c)
			if count1 != 0 {
				dest -= count1
				cursor1 -= count1
				len1 -= count1
				copy(a[dest+1:dest+1+count1], a[cursor1+1:cursor1+1+count1])
				if len1 == 0 {
					break outer
				}
			}
			a[dest] = tmp[cursor2]
			dest--
			cursor2--
			len2--
			if len2 == 1 {
				break outer
			}
			count2 = len2 - javaTimGallopLeft(a[cursor1], tmp, 0, len2, len2-1, c)
			if count2 != 0 {
				dest -= count2
				cursor2 -= count2
				len2 -= count2
				copy(a[dest+1:dest+1+count2], tmp[cursor2+1:cursor2+1+count2])
				if len2 <= 1 {
					break outer
				}
			}
			a[dest] = a[cursor1]
			dest--
			cursor1--
			len1--
			if len1 == 0 {
				break outer
			}
			minGallop--
			if count1 < javaTimMinGallop && count2 < javaTimMinGallop {
				break
			}
		}
		if minGallop < 0 {
			minGallop = 0
		}
		minGallop += 2
	}
	ts.minGallop = max(1, minGallop)
	switch len2 {
	case 1:
		dest -= len1
		cursor1 -= len1
		copy(a[dest+1:dest+1+len1], a[cursor1+1:cursor1+1+len1])
		a[dest] = tmp[cursor2]
	case 0:
		javaTimContractViolation()
	default:
		copy(a[dest-(len2-1):dest+1], tmp[:len2])
	}
}

func javaTimTempCapacity(minimum, arrayLength int) int {
	newSize := uint64(1) << bits.Len(uint(minimum))
	if newSize == 0 || newSize > uint64(1<<31-1) {
		return minimum
	}
	return min(int(newSize), arrayLength>>1)
}
func (ts *javaObjectTimSort[E]) ensureCapacity(minimum int) []E {
	if len(ts.tmp) < minimum {
		ts.tmp = make([]E, javaTimTempCapacity(minimum, len(ts.a)))
	}
	return ts.tmp
}
func javaTimContractViolation() {
	panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16([]uint16{'C', 'o', 'm', 'p', 'a', 'r', 'i', 's', 'o', 'n', ' ', 'm', 'e', 't', 'h', 'o', 'd', ' ', 'v', 'i', 'o', 'l', 'a', 't', 'e', 's', ' ', 'i', 't', 's', ' ', 'g', 'e', 'n', 'e', 'r', 'a', 'l', ' ', 'c', 'o', 'n', 't', 'r', 'a', 'c', 't', '!'})))
}
