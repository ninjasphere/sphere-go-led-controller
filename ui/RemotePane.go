package ui

import (
	"encoding/gob"
	"image"
	"io"
	"net"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/logger"
)

type RemotePane struct {
	Disconnected chan bool
	log          *logger.Logger
	conn         net.Conn
	incoming     *gob.Decoder
	outgoing     *gob.Encoder
	enabled      bool
}

type Outgoing struct {
	Gesture *gestic.GestureMessage
}

type Incoming struct {
	Image *image.RGBA
	Err   error
}

func NewRemotePane(conn net.Conn) *RemotePane {

	spew.Dump("new Remote pane", conn)

	pane := &RemotePane{
		conn:         conn,
		log:          logger.GetLogger("RemotePane"),
		Disconnected: make(chan bool, 1),
		incoming:     gob.NewDecoder(conn),
		outgoing:     gob.NewEncoder(conn),
		enabled:      true,
	}

	return pane
}

func (p *RemotePane) IsEnabled() bool {
	return p.enabled
}

func (p *RemotePane) Gesture(gesture *gestic.GestureMessage) {
	err := p.outgoing.Encode(gesture)
	if err != nil {
		if err == io.EOF {
			p.enabled = false
			p.Disconnected <- true
		} else {
			p.log.Fatalf("Failed to gob encode gesture: %s", err)
		}
	}
}

func (p *RemotePane) Render() (*image.RGBA, error) {
	var msg Incoming
	err := p.incoming.Decode(&msg)

	if err != nil {
		if err == io.EOF {
			p.enabled = false
			p.Disconnected <- true
		}

		return nil, err
	}

	spew.Dump("Got incoming remote message", msg)

	return msg.Image, msg.Err
}

func (p *RemotePane) IsDirty() bool {
	return true
}
