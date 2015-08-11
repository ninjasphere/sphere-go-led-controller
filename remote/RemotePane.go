package remote

import (
	"encoding/gob"
	"fmt"
	"image"
	"net"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
)

// This is the maximum time we will wait for a frame before disconnecting the remote pane
var remotePaneTimeout = config.Duration(time.Second, "led.remote.paneTimeout")

type Pane struct {
	Disconnected   chan bool
	log            *logger.Logger
	conn           net.Conn
	incoming       *gob.Decoder
	outgoing       *gob.Encoder
	enabled        bool
	incomingFrames chan *Incoming
	keepAwake      bool
	locked         bool
}

type Outgoing struct {
	FrameRequested bool
	Gesture        *gestic.GestureMessage
}

type Incoming struct {
	Image     *image.RGBA
	Err       error
	KeepAwake bool
	Locked    bool
}

func NewPane(conn net.Conn) *Pane {

	pane := &Pane{
		conn:           conn,
		log:            logger.GetLogger("Pane"),
		Disconnected:   make(chan bool, 1),
		incoming:       gob.NewDecoder(conn),
		outgoing:       gob.NewEncoder(conn),
		enabled:        true,
		incomingFrames: make(chan *Incoming, 1),
	}

	// Ping the remote pane continuously so we can see if it's disappeared.
	// This is kinda dumb.
	go func() {
		for {
			if !pane.enabled {
				break
			}
			pane.out(Outgoing{})
			time.Sleep(time.Second)
		}
	}()

	go pane.listen()

	return pane
}

func (p *Pane) IsEnabled() bool {
	return p.enabled
}

func (p *Pane) KeepAwake() bool {
	return p.keepAwake
}

func (p *Pane) Locked() bool {
	return p.locked
}

func (p *Pane) Gesture(gesture *gestic.GestureMessage) {
	if !p.enabled {
		return
	}

	if gesture.Gesture.Gesture != gestic.GestureNone || gesture.Touch.Active() || gesture.Tap.Active() || gesture.DoubleTap.Active() || gesture.AirWheel.Active {
		p.out(Outgoing{false, gesture})
	}
}

func (p *Pane) out(msg Outgoing) error {
	err := p.outgoing.Encode(msg)
	if err != nil {
		p.log.Errorf("Failed to gob encode outgoing remote message: %s", err)
		p.Close()
	}
	return err
}

func (p *Pane) listen() {
	for {
		var msg Incoming
		err := p.incoming.Decode(&msg)

		//p.log.Debugf("Got an incoming message")

		if err != nil {
			p.Close()
			break
		}

		p.keepAwake = msg.KeepAwake
		p.locked = msg.Locked

		p.incomingFrames <- &msg
	}
}

func (p *Pane) Render() (*image.RGBA, error) {

	if !p.enabled {
		return nil, fmt.Errorf("This remote pane has disconnected.")
	}

	err := p.out(Outgoing{true, nil})

	if err != nil {
		return nil, err
	}

	select {
	case msg := <-p.incomingFrames:
		//p.log.Debugf("Got incoming remote message")

		return msg.Image, msg.Err
	case <-time.After(remotePaneTimeout):
		p.log.Errorf("Remote pane timed out")
		p.Close()

		return nil, fmt.Errorf("Remote pane timed out")
	}

}

func (p *Pane) Close() {
	if p.enabled {
		p.enabled = false
		p.conn.Close()
		p.Disconnected <- true
	}
}

func (p *Pane) IsDirty() bool {
	return true
}
