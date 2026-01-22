package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystem(t *testing.T) {
	r := NewSystem()
	require.NotNil(t, r)
	_, ok := r.(*System)
	assert.True(t, ok)
}

func TestSystemResolve(t *testing.T) {
	r := NewSystem()

	t.Run("resolve localhost", func(t *testing.T) {
		ipv4, ipv6, err := r.Resolve("localhost")
		require.NoError(t, err)
		// At least one should be non-nil
		assert.True(t, ipv4 != nil || ipv6 != nil, "should resolve to at least one IP")
	})

	// Note: Testing invalid domain resolution is unreliable because
	// some ISPs/networks hijack DNS for non-existent domains
}
