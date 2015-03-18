package remote

import (
	"encoding/gob"
	"fmt"
	"image"
	"io"
	"net"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/logger"
)

type pane interface {
	IsEnabled() bool
	KeepAwake() bool
	Render() (*image.RGBA, error)
	Gesture(*gestic.GestureMessage)
}

type Matrix struct {
	Disconnected chan bool
	log          *logger.Logger
	conn         net.Conn
	incoming     *gob.Decoder
	outgoing     *gob.Encoder
	pane         pane
}

func NewMatrix(pane pane, conn net.Conn) *Matrix {

	matrix := &Matrix{
		conn:         conn,
		log:          logger.GetLogger("Matrix"),
		Disconnected: make(chan bool, 1),
		incoming:     gob.NewDecoder(conn),
		outgoing:     gob.NewEncoder(conn),
		pane:         pane,
	}

	defer func() {
		matrix.outgoing.Encode(&Incoming{Err: fmt.Errorf("Goodbye!")})
	}()

	go matrix.start()

	return matrix
}

func (m *Matrix) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
	m.Disconnected <- true
}

func (m *Matrix) start() {

	for {
		var msg Outgoing
		err := m.incoming.Decode(&msg)

		if err != nil {
			if err == io.EOF {
				m.log.Warningf("Lost connection to led controller: %s", err)
			} else {
				m.log.Errorf("Error communicating with led controller: %s", err)
			}

			m.Close()
			break
		}

		if msg.Gesture != nil {
			m.pane.Gesture(msg.Gesture)
		}

		if msg.FrameRequested {
			//m.log.Debugf("Rendering pane...")
			img, err := m.pane.Render()

			if err != nil {
				m.log.Errorf("Pane returned an error: %s", err)
			}

			if err := m.outgoing.Encode(&Incoming{img, err, m.pane.KeepAwake()}); err != nil {
				m.log.Errorf("Remote matrix error: %s. Disconnecting.", err)
				m.Close()
				break
			}

			//m.log.Debugf("Sent frame")
		}
	}
}
