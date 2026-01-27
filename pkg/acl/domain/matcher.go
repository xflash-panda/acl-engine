package domain

import (
	"sort"
	"strings"
)

const (
	// Special labels for domain matching
	prefixLabel = '\r' // Marks suffix patterns (e.g., ".google.com")
	rootLabel   = '\n' // Marks root domain patterns (e.g., "google.com")
)

// Matcher provides efficient domain name matching using succinct trie.
type Matcher struct {
	set *succinctSet
}

// NewMatcher creates a new domain matcher from domain lists.
// domains: exact domain matches
// domainSuffix: suffix matches (e.g., "google.com" matches "*.google.com")
func NewMatcher(domains []string, domainSuffix []string) *Matcher {
	if len(domains) == 0 && len(domainSuffix) == 0 {
		return &Matcher{set: &succinctSet{}}
	}

	domainList := make([]string, 0, len(domains)+len(domainSuffix))
	seen := make(map[string]bool, len(domains)+len(domainSuffix))

	// Process suffix domains
	for _, domain := range domainSuffix {
		domain = strings.ToLower(domain)
		if seen[domain] {
			continue
		}
		seen[domain] = true

		if strings.HasPrefix(domain, ".") {
			// Domain starts with dot: only match subdomains
			domainList = append(domainList, reverseDomain(string(prefixLabel)+domain))
		} else {
			// Domain without dot: match both exact and subdomains
			// This uses rootLabel to enable flexible matching
			domainList = append(domainList, reverseDomain(string(rootLabel)+domain))
		}
	}

	// Process exact domains
	for _, domain := range domains {
		domain = strings.ToLower(domain)
		if seen[domain] {
			continue
		}
		seen[domain] = true
		domainList = append(domainList, reverseDomain(domain))
	}

	// Sort for trie construction
	sort.Strings(domainList)

	return &Matcher{set: newSuccinctSet(domainList)}
}

// Match checks if the given domain matches any rule.
func (m *Matcher) Match(domain string) bool {
	if m.set == nil || len(m.set.labels) == 0 {
		return false
	}
	domain = strings.ToLower(domain)
	return m.has(reverseDomain(domain))
}

// has performs the actual matching on the reversed domain.
func (m *Matcher) has(key string) bool {
	if len(m.set.labelBitmap) == 0 || len(m.set.labels) == 0 {
		return false
	}

	var nodeId, bmIdx int

	// Traverse the trie character by character
	for i := 0; i < len(key); i++ {
		currentChar := key[i]

		// Check all edges from current node
		for {
			// Check if we've reached the end of this node's edges
			if getBit(m.set.labelBitmap, bmIdx) != 0 {
				return false // No matching edge found
			}

			// Bounds check for labels array
			labelIdx := bmIdx - nodeId
			if labelIdx < 0 || labelIdx >= len(m.set.labels) {
				return false
			}

			nextLabel := m.set.labels[labelIdx]

			// Check for suffix match marker
			if nextLabel == prefixLabel {
				return true // Found suffix match
			}

			// Check for root domain marker
			if nextLabel == rootLabel {
				nextNodeId := countZeros(m.set.labelBitmap, m.set.ranks, bmIdx+1)
				hasNext := getBit(m.set.leaves, nextNodeId) != 0
				// If current char is dot and node is leaf, we have subdomain match
				if currentChar == '.' && hasNext {
					return true
				}
			}

			// Found matching character
			if nextLabel == currentChar {
				break
			}

			bmIdx++
		}

		// Move to next node
		nodeId = countZeros(m.set.labelBitmap, m.set.ranks, bmIdx+1)
		if nodeId <= 0 {
			return false
		}
		bmIdx = selectIthOne(m.set.labelBitmap, m.set.ranks, m.set.selects, nodeId-1) + 1
	}

	// Check if we're at a leaf node (exact match)
	if getBit(m.set.leaves, nodeId) != 0 {
		return true
	}

	// Check for suffix/root markers after consuming all input
	for {
		if getBit(m.set.labelBitmap, bmIdx) != 0 {
			return false
		}

		labelIdx := bmIdx - nodeId
		if labelIdx < 0 || labelIdx >= len(m.set.labels) {
			return false
		}

		nextLabel := m.set.labels[labelIdx]
		if nextLabel == prefixLabel || nextLabel == rootLabel {
			return true
		}
		bmIdx++
	}
}
