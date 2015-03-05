package remote

import (
	"encoding/gob"
	"image"
	"io"
	"log"
	"net"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/logger"
)

type pane interface {
	IsEnabled() bool
	Render() (*image.RGBA, error)
	Gesture(*gestic.GestureMessage)
}

type Matrix struct {
	Disconnected chan bool
	log          *logger.Logger
	conn         net.Conn
	incoming     *gob.Decoder
	outgoing     *gob.Encoder
	enabled      bool
	pane         pane
}

func NewMatrix(pane pane, conn net.Conn) *Matrix {

	matrix := &Matrix{
		conn:         conn,
		log:          logger.GetLogger("Matrix"),
		Disconnected: make(chan bool, 1),
		incoming:     gob.NewDecoder(conn),
		outgoing:     gob.NewEncoder(conn),
		enabled:      true,
		pane:         pane,
	}

	go matrix.start()

	return matrix
}

func (m *Matrix) start() {

	for {
		var msg Outgoing
		err := m.incoming.Decode(&msg)

		if err != nil {
			if err == io.EOF {
				log.Fatalf("Lost connection to led controller: %s", err)
			}

			log.Fatalf("Error communicating with led controller: %s", err)
		}

		if msg.Gesture != nil {
			m.pane.Gesture(msg.Gesture)
		}

		if msg.FrameRequested {
			m.log.Debugf("Rendering pane...")
			img, err := m.pane.Render()

			m.outgoing.Encode(&Incoming{img, err})
			m.log.Debugf("Sent frame")
		}
	}
}
