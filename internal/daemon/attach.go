package daemon

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// AttachMux manages I/O multiplexing between a container's PTY and attached clients.
// It continuously reads from the PTY master so the container never blocks on writes.
// When a client is attached, output is forwarded; otherwise it is discarded.
type AttachMux struct {
	master   *os.File
	mu       sync.Mutex
	conn     net.Conn
	attached bool
	doneCh   chan struct{}
}

// NewAttachMux creates a new multiplexer and starts draining the PTY master.
func NewAttachMux(master *os.File) *AttachMux {
	m := &AttachMux{
		master: master,
		doneCh: make(chan struct{}),
	}
	go m.readLoop()
	return m
}

// readLoop continuously reads from the PTY master.
// Output -> client if attached, otherwise discarded.
func (m *AttachMux) readLoop() {
	defer close(m.doneCh)
	buf := make([]byte, 32*1024)
	for {
		n, err := m.master.Read(buf)
		if n > 0 {
			m.mu.Lock()
			if m.conn != nil {
				// Best-effort write; ignore errors (client may have disconnected)
				m.conn.Write(buf[:n])
			}
			m.mu.Unlock()
		}
		if err != nil {
			// Close attached connection so the client sees the disconnect.
			m.mu.Lock()
			if m.conn != nil {
				m.conn.Close()
			}
			m.mu.Unlock()
			return
		}
	}
}

// Attach connects a client to the container's I/O.
// It blocks until the client disconnects or the container exits.
// Only one client may be attached at a time.
func (m *AttachMux) Attach(conn net.Conn) error {
	select {
	case <-m.doneCh:
		return fmt.Errorf("container has exited")
	default:
	}

	m.mu.Lock()
	if m.attached {
		m.mu.Unlock()
		return fmt.Errorf("another client is already attached")
	}
	m.conn = conn
	m.attached = true
	m.mu.Unlock()

	// Input → PTY master
	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			if _, werr := m.master.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}

	// Detach: clear the connection so readLoop stops forwarding output
	m.mu.Lock()
	m.conn = nil
	m.attached = false
	m.mu.Unlock()

	return nil
}
