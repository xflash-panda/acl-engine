package outbound

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFastOpenConn_ConcurrentClose(t *testing.T) {
	// Bug: Close() uses RLock and calls close(readyChan).
	// Two concurrent Close() calls can double-close the channel, causing panic.
	c := &fastOpenConn{
		network:   "tcp",
		address:   "127.0.0.1:80",
		readyChan: make(chan struct{}),
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// All goroutines call Close() concurrently - should not panic
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = c.Close()
		}()
	}

	wg.Wait()
	// If we get here without panic, the test passes.
	// Second close should return net.ErrClosed
	err := c.Close()
	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestFastOpenConn_SetDeadlineRace(t *testing.T) {
	// Bug: SetDeadline/SetReadDeadline/SetWriteDeadline use RLock but
	// write to c.deadline/c.readDeadline/c.writeDeadline fields,
	// causing data races under concurrent access.
	//
	// Run with -race to detect:
	//   go test -race -run TestFastOpenConn_SetDeadlineRace
	c := &fastOpenConn{
		network:   "tcp",
		address:   "127.0.0.1:80",
		readyChan: make(chan struct{}),
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = c.SetDeadline(time.Now().Add(time.Second))
		}()
		go func() {
			defer wg.Done()
			_ = c.SetReadDeadline(time.Now().Add(time.Second))
		}()
		go func() {
			defer wg.Done()
			_ = c.SetWriteDeadline(time.Now().Add(time.Second))
		}()
	}

	wg.Wait()
	_ = c.Close()
}
