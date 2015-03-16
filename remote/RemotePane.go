package remote

import (
	"encoding/gob"
	"image"
	"io"
	"net"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/logger"
)

type Pane struct {
	Disconnected chan bool
	log          *logger.Logger
	conn         net.Conn
	incoming     *gob.Decoder
	outgoing     *gob.Encoder
	enabled      bool
}

type Outgoing struct {
	FrameRequested bool
	Gesture        *gestic.GestureMessage
}

type Incoming struct {
	Image *image.RGBA
	Err   error
}

func NewPane(conn net.Conn) *Pane {
	pane := &Pane{
		conn:         conn,
		log:          logger.GetLogger("Pane"),
		Disconnected: make(chan bool, 1),
		incoming:     gob.NewDecoder(conn),
		outgoing:     gob.NewEncoder(conn),
		enabled:      true,
	}

	return pane
}

func (p *Pane) IsEnabled() bool {
	return p.enabled
}

func (p *Pane) Gesture(gesture *gestic.GestureMessage) {
	if gesture.Gesture.Gesture != gestic.GestureNone || gesture.Touch.Active() || gesture.Tap.Active() || gesture.DoubleTap.Active() {
		p.out(Outgoing{false, gesture})
	}
}

func (p *Pane) out(msg Outgoing) {
	err := p.outgoing.Encode(msg)
	if err != nil {
		if err == io.EOF {
			p.enabled = false
			p.Disconnected <- true
		} else {
			p.log.Errorf("Failed to gob encode outgoing remote message: %s", err)
		}
	}
}

func (p *Pane) Render() (*image.RGBA, error) {

	p.out(Outgoing{true, nil})

	var msg Incoming
	err := p.incoming.Decode(&msg)

	if err != nil {
		if err == io.EOF {
			p.enabled = false
			p.Disconnected <- true
		}

		return nil, nil
	}

	p.log.Debugf("Got incoming remote message")

	return msg.Image, msg.Err
}

func (p *Pane) IsDirty() bool {
	return true
}
