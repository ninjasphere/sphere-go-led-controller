package remote

import (
	"encoding/gob"
	"image"
	"io"
	"net"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/logger"
)

var log = logger.GetLogger("remote")

type pane interface {
	IsEnabled() bool
	KeepAwake() bool
	Render() (*image.RGBA, error)
	Gesture(*gestic.GestureMessage)
}

type lockable interface {
	Locked() bool
}

type Matrix struct {
	Disconnected chan bool
	log          *logger.Logger
	conn         net.Conn
	incoming     *gob.Decoder
	outgoing     *gob.Encoder
	pane         pane
}

func NewTCPMatrix(pane pane, host string) *Matrix {

	matrix := NewMatrix(pane)

	// Connect to the led controller remote pane interface

	go func() {
		for {
			tcpAddr, err := net.ResolveTCPAddr("tcp", host)
			if err != nil {
				println("ResolveTCPAddr failed:", err.Error())
			} else {

				conn, err := net.DialTCP("tcp", nil, tcpAddr)
				if err != nil {
					log.Errorf("Dial failed: %s", err)
				} else {

					log.Infof("Connected")

					go matrix.start(conn)

					<-matrix.Disconnected

					log.Infof("Disconnected")
				}
			}
			log.Infof("Waiting to reconnect")
			time.Sleep(time.Second / 2)
		}
	}()

	return matrix
}

func NewMatrix(pane pane) *Matrix {

	matrix := &Matrix{
		log:          logger.GetLogger("Matrix"),
		Disconnected: make(chan bool, 1),
		pane:         pane,
	}

	return matrix
}

func (m *Matrix) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
	m.Disconnected <- true
}

func (m *Matrix) start(conn net.Conn) {

	m.conn = conn
	m.incoming = gob.NewDecoder(m.conn)
	m.outgoing = gob.NewEncoder(m.conn)

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

			var locked = false
			if lockablePane, ok := m.pane.(lockable); ok {
				locked = lockablePane.Locked()
			}

			if err := m.outgoing.Encode(&Incoming{img, err, m.pane.KeepAwake(), locked}); err != nil {
				m.log.Errorf("Remote matrix error: %s. Disconnecting.", err)
				m.Close()
				break
			}

			//m.log.Debugf("Sent frame")
		}
	}
}
