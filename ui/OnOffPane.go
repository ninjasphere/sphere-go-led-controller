package ui

import (
	"image"
	"log"
	"strings"
	"time"

	"github.com/ninjasphere/go-gestic"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
)

type OnOffPane struct {
	log  *logger.Logger
	conn *ninja.Connection

	devices []*ninja.ServiceClient

	state         bool
	onStateChange func(bool)

	onImage  *Image
	offImage *Image

	ignoringGestures bool
}

func NewOnOffPane(offImage string, onImage string, onStateChange func(bool), conn *ninja.Connection, thingType string) *OnOffPane {

	devices, err := getChannelServices(thingType, "on-off", conn)
	if err != nil {
		log.Fatalf("Failed to get %s devices: %s", err, err)
	}

	log := logger.GetLogger("OnOffPane")
	log.Infof("Pane got %d on/off devices", len(devices))

	return &OnOffPane{
		onImage:       loadImage(onImage),
		offImage:      loadImage(offImage),
		onStateChange: onStateChange,
		log:           log,
		devices:       devices,
		conn:          conn,
	}
}

func (p *OnOffPane) Gesture(gesture *gestic.GestureData) {
	if p.ignoringGestures {
		return
	}

	if strings.Contains(gesture.Touch.Name(), "Tap") {
		p.log.Infof("Tap!")

		p.ignoringGestures = true

		go func() {
			time.Sleep(time.Millisecond * 250)
			p.ignoringGestures = false
		}()

		p.SetState(!p.state)
	}
}

func (p *OnOffPane) SetState(state bool) {
	p.state = state
	for _, device := range p.devices {
		if state {
			device.Call("turnOn", nil, nil, 0)
		} else {
			device.Call("turnOff", nil, nil, 0)
		}
	}
	p.onStateChange(state)
}

func (p *OnOffPane) Render() (*image.RGBA, error) {
	if p.state {
		return p.onImage.GetNextFrame(), nil
	}
	return p.offImage.GetNextFrame(), nil
}

func (p *OnOffPane) IsDirty() bool {
	return true
}
