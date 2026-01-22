package outbound

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReject(t *testing.T) {
	ob := NewReject()
	require.NotNil(t, ob)

	t.Run("DialTCP returns error", func(t *testing.T) {
		conn, err := ob.DialTCP(&Addr{Host: "example.com", Port: 443})
		assert.Nil(t, conn)
		assert.Error(t, err)
		assert.Equal(t, errRejected, err)
	})

	t.Run("DialUDP returns error", func(t *testing.T) {
		conn, err := ob.DialUDP(&Addr{Host: "example.com", Port: 53})
		assert.Nil(t, conn)
		assert.Error(t, err)
		assert.Equal(t, errRejected, err)
	})
}
