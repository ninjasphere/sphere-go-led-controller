package ui

import (
	"image"
	"log"
	"net/rpc"
	"strings"
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/ninjasphere/driver-go-gestic/gestic"
	"github.com/ninjasphere/go-ninja/logger"
)

type OnOffPane struct {
	log *logger.Logger

	devices []*rpc.Client

	state         bool
	onStateChange func(bool)

	onImage  *Image
	offImage *Image

	ignoringGestures bool
}

func NewOnOffPane(offImage string, onImage string, onStateChange func(bool), mqtt *mqtt.MqttClient, thingType string) *OnOffPane {

	devices, err := getChannelClients(thingType, "on-off", mqtt)
	if err != nil {
		log.Fatalf("Failed to get on-off devices", err)
	}

	log := logger.GetLogger("OnOffPane")
	log.Infof("Pane got %d on/off devices", len(devices))

	return &OnOffPane{
		onImage:       loadImage(onImage),
		offImage:      loadImage(offImage),
		onStateChange: onStateChange,
		log:           log,
		devices:       devices,
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
			_ = device.Go("turnOn", nil, nil, nil)
		} else {
			_ = device.Go("turnOff", nil, nil, nil)
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
