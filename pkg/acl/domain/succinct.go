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
// Returns an array where ranks[i] = number of 1 bits in words[0:i].
// The array has len(words)+1 entries, with ranks[len(words)] = total ones.
func indexRank64(words []uint64) []int32 {
	idx := make([]int32, len(words)+1)
	n := int32(0)
	for i := 0; i < len(words); i++ {
		idx[i] = n
		n += int32(bits.OnesCount64(words[i])) // #nosec G115 -- OnesCount64 max is 64
	}
	idx[len(words)] = n // Total count at the end
	return idx
}

// select32R64 finds the position of the i-th 1 bit using select index.
// The select index samples every 32 ones, so we use i>>5 to find the starting word.
func select32R64(words []uint64, sindex, rindex []int32, i int32) (int32, int32) {
	if i < 0 || len(sindex) == 0 {
		return 0, 0
	}

	l := int32(len(words))

	// Use select index to find starting word (samples every 32 ones)
	sidx := i >> 5 // #nosec G115 -- index fits in int32
	if sidx >= int32(len(sindex)) {
		sidx = int32(len(sindex)) - 1
	}
	wordI := sindex[sidx] >> 6

	// Linear search to find the word containing the i-th one
	// rindex has len(words)+1 entries, so rindex[wordI+1] is safe
	for wordI < l && rindex[wordI+1] <= i {
		wordI++
	}

	if wordI >= l {
		return l << 6, l << 6
	}

	// Find the position within the word
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
// The select index stores the bit position of every 32nd one bit,
// enabling O(1) lookup to find the word containing any given one bit.
func indexSelect32R64(words []uint64) ([]int32, []int32) {
	ranks := indexRank64(words)

	// Build select index by scanning all bits
	// Sample every 32nd one bit (when ith % 32 == 0)
	l := len(words) << 6
	selects := make([]int32, 0, len(words))

	ith := int32(-1)
	for i := 0; i < l; i++ {
		if words[i>>6]&(1<<uint(i&63)) != 0 {
			ith++
			if ith%32 == 0 {
				selects = append(selects, int32(i)) // #nosec G115 -- bit position fits in int32
			}
		}
	}

	// Ensure we have at least one entry
	if len(selects) == 0 {
		selects = append(selects, 0)
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
