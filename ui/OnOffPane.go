package ui

import (
	"image"
	"log"
	"strings"
	"time"

	"github.com/ninjasphere/driver-go-gestic/gestic"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/rpc"
)

type OnOffPane struct {
	log *logger.Logger
	rpc *rpc.Client

	devices []string

	state         bool
	onStateChange func(bool)

	onImage  *Image
	offImage *Image

	ignoringGestures bool
}

func NewOnOffPane(offImage string, onImage string, onStateChange func(bool), rpcClient *rpc.Client, thingType string) *OnOffPane {

	devices, err := getChannelIds(thingType, "on-off", rpcClient)
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
		rpc:           rpcClient,
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
			p.rpc.Call(device, "turnOn", nil, nil)
		} else {
			p.rpc.Call(device, "turnOff", nil, nil)
		}
	}
	p.onStateChange(state)
}

func (p *OnOffPane) Render() (*image.RGBA, error) {
	if p.state {
		return p.onImage.GetFrame(), nil
	}
	return p.offImage.GetFrame(), nil
}

func (p *OnOffPane) IsDirty() bool {
	return true
}
