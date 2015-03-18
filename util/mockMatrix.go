package util

import (
	"fmt"
	"io"
)

const (
	stateCmd   = 0 // expecting a command byte
	stateData  = 1 // expecting exactly 768 data bytes
	stateSwap  = 2 // expecting a read or a command byte
	stateClose = 3 // expecting nothing

	maxWrite = 768 // the max number of bytes received in the stateData state
)

type mockMatrix struct {
	state int // the current state of the connection
	count int // number of bytes received since entering the data state
}

// Answers a mock for the matrix that simulates a real led matrix.
func newMockMatrix() io.ReadWriteCloser {
	return &mockMatrix{
		state: stateCmd,
		count: 0,
	}
}

// Answers 'F' when the mock matrix is in the swap state, otherwise answers empty.
func (m *mockMatrix) Read(p []byte) (n int, err error) {
	switch m.state {
	case stateSwap:
		if len(p) == 0 {
			return 0, fmt.Errorf("insufficient capacity")
		} else {
			p[0] = 'F'
			m.count = 0
			m.state = stateCmd
			return 1, nil
		}
	case stateClose:
		return 0, fmt.Errorf("stream is closed")
	default:
		return 0, nil
	}
}

// Adjusts the state of the mock matrix connection according to bytes received.
//
// Valid sequences are zero or more iterations of:
//
// []byte{cmdWriteBuffer}, [768]byte, []byte{cmdSwapBuffers}...
//
// All other sequences will result in the connection moving into a closed state.
func (m *mockMatrix) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	} else {
	buffer:
		for i, c := range p {
			switch m.state {
			case stateCmd:
				switch c {
				case stateData:
					m.count = 0
					m.state = stateData
				case stateSwap:
					m.state = stateSwap
				default:
					old := *m
					m.state = stateClose
					return i + 1, fmt.Errorf("unexpected command received while waiting for command: 0x%02x, %v", c, old)
				}
			case stateData:
				if i == 0 && m.count+len(p) <= maxWrite {
					// optimization for common case where the write length is less than or equal to the full 768 bytes
					m.count += len(p)
					if m.count == maxWrite {
						m.state = stateCmd
					}
					break buffer
				}
				m.count++
				if m.count == maxWrite {
					m.state = stateCmd
				}
			case stateSwap:
				switch c {
				case stateData:
					m.state = stateData
					m.count = 0
				default:
					old := *m
					m.state = stateClose
					return i + 1, fmt.Errorf("unexpected byte received (0x%02x) while in swap state %v", c, old)
				}
			case stateClose:
				return 0, fmt.Errorf("stream is closed")
			}
		}
		return len(p), nil
	}

}

// Moves the connection into the closed state.
func (m *mockMatrix) Close() error {
	m.state = stateClose
	return nil
}
