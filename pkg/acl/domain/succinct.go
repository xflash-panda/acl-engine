package domain

import (
	"math/bits"
	"unicode/utf8"
)

// succinctSet is a memory-efficient trie implementation using bitmaps.
// Adapted from sing-box's domain matcher implementation.
// Original: github.com/sagernet/sing/common/domain
type succinctSet struct {
	leaves      []uint64 // Bitmap marking leaf nodes (domain terminations)
	labelBitmap []uint64 // Bitmap marking node boundaries
	labels      []byte   // Character labels for trie edges
	ranks       []int32  // Rank index for fast bitmap operations
	selects     []int32  // Select index for fast bitmap operations
}

// newSuccinctSet constructs a succinct trie from a sorted list of keys.
func newSuccinctSet(keys []string) *succinctSet {
	if len(keys) == 0 {
		return &succinctSet{}
	}

	ss := &succinctSet{}
	lIdx := 0

	// Queue element: start index, end index, column position
	type qElt struct{ s, e, col int }
	queue := []qElt{{0, len(keys), 0}}

	// Build trie using BFS
	for i := 0; i < len(queue); i++ {
		elt := queue[i]

		// Check if this is a leaf node (key ends at this column)
		if elt.col == len(keys[elt.s]) {
			elt.s++
			setBit(&ss.leaves, i, 1)
		}

		// Process all children with same prefix
		for j := elt.s; j < elt.e; {
			frm := j
			// Find range of keys with same character at this column
			for ; j < elt.e && keys[j][elt.col] == keys[frm][elt.col]; j++ {
			}
			// Add child node to queue
			queue = append(queue, qElt{frm, j, elt.col + 1})
			ss.labels = append(ss.labels, keys[frm][elt.col])
			setBit(&ss.labelBitmap, lIdx, 0) // 0 = edge, 1 = node end
			lIdx++
		}
		setBit(&ss.labelBitmap, lIdx, 1) // Mark end of this node's edges
		lIdx++
	}

	ss.init() // Build rank/select indices
	return ss
}

// init builds the rank and select indices for efficient bitmap navigation.
func (ss *succinctSet) init() {
	ss.selects, ss.ranks = indexSelect32R64(ss.labelBitmap)
}

// setBit sets the i-th bit in the bitmap to v (0 or 1).
func setBit(bm *[]uint64, i int, v int) {
	// Expand bitmap if necessary
	for i>>6 >= len(*bm) {
		*bm = append(*bm, 0)
	}
	(*bm)[i>>6] |= uint64(v) << (i & 63) // #nosec G115 -- i&63 is always 0-63
}

// getBit returns the i-th bit from the bitmap.
func getBit(bm []uint64, i int) uint64 {
	if i>>6 >= len(bm) {
		return 0
	}
	return bm[i>>6] & (1 << (i & 63)) // #nosec G115 -- i&63 is always 0-63
}

// countZeros counts the number of 0 bits before position i.
func countZeros(bm []uint64, ranks []int32, i int) int {
	ones, _ := rank64(bm, ranks, int32(i)) // #nosec G115 -- bitmap index fits in int32
	return i - int(ones)
}

// selectIthOne finds the position of the i-th 1 bit.
func selectIthOne(bm []uint64, ranks, selects []int32, i int) int {
	pos, _ := select32R64(bm, selects, ranks, int32(i)) // #nosec G115 -- index fits in int32
	return int(pos)
}

// rank64 returns the number of 1 bits before position i.
func rank64(words []uint64, rindex []int32, i int32) (int32, int32) {
	if i < 0 {
		return 0, 0
	}
	wordI := i >> 6
	if int(wordI) >= len(words) {
		if len(rindex) > 0 {
			return rindex[len(rindex)-1], 0
		}
		return 0, 0
	}
	j := uint32(i & 63) // #nosec G115 -- i&63 is always 0-63, fits in uint32
	n := rindex[wordI]
	w := words[wordI]
	// Use hardware POPCNT instruction via bits.OnesCount64
	c1 := n + int32(bits.OnesCount64(w&((1<<j)-1))) // #nosec G115 -- OnesCount64 max is 64
	return c1, int32(w>>j) & 1                      // #nosec G115 -- result is 0 or 1
}

// indexRank64 builds a rank index for the bitmap.
func indexRank64(words []uint64) []int32 {
	idx := make([]int32, len(words))
	n := int32(0)
	for i := 0; i < len(words); i++ {
		idx[i] = n
		n += int32(bits.OnesCount64(words[i])) // #nosec G115 -- OnesCount64 max is 64
	}
	return idx
}

// select32R64 finds the position of the i-th 1 bit using select index.
func select32R64(words []uint64, sindex, rindex []int32, i int32) (int32, int32) {
	if i < 0 || len(sindex) == 0 {
		return 0, 0
	}

	// Use select index to narrow down search range
	wordI := int32(0)
	if int(i>>8) < len(sindex) {
		wordI = sindex[i>>8]
	}

	// Linear search within the narrowed range
	rindexLen := int32(len(rindex)) // #nosec G115 -- rindex length fits in int32
	for wordI < rindexLen && rindex[wordI] <= i {
		wordI++
	}
	if wordI > 0 {
		wordI--
	}

	if int(wordI) >= len(words) {
		return 0, 0
	}

	n := i - rindex[wordI]
	w := words[wordI]

	// Find the n-th 1 bit within the word
	for n > 0 {
		w &= w - 1 // Clear lowest 1 bit
		n--
	}

	bitPos := int32(bits.TrailingZeros64(w)) // #nosec G115 -- TrailingZeros64 max is 64
	return (wordI << 6) + bitPos, 1
}

// indexSelect32R64 builds both rank and select indices.
func indexSelect32R64(words []uint64) ([]int32, []int32) {
	ranks := indexRank64(words)

	// Build select index (one entry per 256 ones)
	totalOnes := int32(0)
	if len(ranks) > 0 {
		totalOnes = ranks[len(ranks)-1]
		if len(words) > 0 {
			totalOnes += int32(bits.OnesCount64(words[len(words)-1])) // #nosec G115 -- OnesCount64 max is 64
		}
	}

	selectSize := (int(totalOnes) >> 8) + 1
	selects := make([]int32, selectSize)
	selectsLen := int32(len(selects)) // #nosec G115 -- selects length fits in int32

	onesCount := int32(0)
	for i := 0; i < len(words); i++ {
		wordOnes := int32(bits.OnesCount64(words[i])) // #nosec G115 -- OnesCount64 max is 64
		for onesCount>>8 < selectsLen && ranks[i] <= onesCount && onesCount < ranks[i]+wordOnes {
			selects[onesCount>>8] = int32(i) // #nosec G115 -- loop index fits in int32
			onesCount += 256
		}
		if i < len(words)-1 {
			onesCount = ranks[i+1]
		}
	}

	return selects, ranks
}

// reverseDomain reverses a domain string for trie storage.
// Example: "google.com" -> "moc.elgoog"
func reverseDomain(domain string) string {
	l := len(domain)
	b := make([]byte, l)
	for i := 0; i < l; {
		r, n := utf8.DecodeRuneInString(domain[i:])
		i += n
		utf8.EncodeRune(b[l-i:], r)
	}
	return string(b)
}
