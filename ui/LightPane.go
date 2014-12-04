package ui

import (
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

var onOffRate = &throttle{delay: time.Millisecond * 250}
var colorRate = &throttle{delay: time.Millisecond * 500}

const colorRotateSpeed = 0.0015

type LightPane struct {
	log  *logger.Logger
	conn *ninja.Connection

	onOffDevices []*ninja.ServiceClient
	colorDevices []*ninja.ServiceClient

	onOffState         bool
	onOnOffStateChange func(bool)

	colorState         float64
	onColorStateChange func(float64)

	onImage  util.Image
	offImage util.Image
}

func NewLightPane(offImage string, onImage string, onOnOffStateChange func(bool), onColorStateChange func(float64), conn *ninja.Connection) *LightPane {

	log := logger.GetLogger("LightPane")
	pane := &LightPane{
		onImage:            util.LoadImage(onImage),
		offImage:           util.LoadImage(offImage),
		onOnOffStateChange: onOnOffStateChange,
		onColorStateChange: onColorStateChange,
		log:                log,
		onOffDevices:       make([]*ninja.ServiceClient, 0),
		colorDevices:       make([]*ninja.ServiceClient, 0),
		conn:               conn,
	}

	getChannelServicesContinuous("light", "on-off", func(devices []*ninja.ServiceClient, err error) {
		if err != nil {
			log.Infof("Failed to update on-off devices: %s", err)
		} else {
			log.Infof("Pane got %d on/off devices", len(devices))
			pane.onOffDevices = devices
		}
	})

	getChannelServicesContinuous("light", "core.batching", func(devices []*ninja.ServiceClient, err error) {
		if err != nil {
			log.Infof("Failed to update batching devices: %s", err)
		} else {
			log.Infof("Pane got %d batching devices", len(devices))
			pane.colorDevices = devices
		}
	})

	return pane
}

func (p *LightPane) Gesture(gesture *gestic.GestureMessage) {

	col := p.colorState + colorRotateSpeed
	if col >= 1 {
		col = 0
	}
	p.colorState = col

	if !onOffRate.busy && colorRate.try() {

		p.SetColorState(col)
		p.log.Infof("Color wheel %f", col)

	} else {
		//p.log.Infof("Ignoring Color wheel... Remaining time: %d\n", remaining)
	}

	if gesture.Tap.Active() {
		if onOffRate.try() {
			p.log.Infof("Tap!")

			p.SetOnOffState(!p.onOffState)
		} else {
			//p.log.Infof("Ignoring Tap... Remaining time: %d\n", remaining)
		}
	}

}

func (p *LightPane) SetOnOffState(state bool) {
	p.onOffState = state
	p.SendOnOffToDevices()
	go p.onOnOffStateChange(state)
}

func (p *LightPane) SetColorState(state float64) {
	p.colorState = state

	p.SendColorToDevices()
	go p.onColorStateChange(state)
}

func (p *LightPane) SendOnOffToDevices() {

	if p.onOffState {
		p.log.Infof("Turning lights on")
	} else {
		p.log.Infof("Turning lights off")
	}

	for _, device := range p.onOffDevices {

		if p.onOffState {
			device.Call("turnOn", nil, nil, 0)
		} else {
			device.Call("turnOff", nil, nil, 0)
		}

	}
}

func (p *LightPane) SendColorToDevices() {
	sat := 0.6

	for _, device := range p.colorDevices {

		colorState := &channels.ColorState{
			Mode:       "hue",
			Hue:        &p.colorState,
			Saturation: &sat,
		}
		transition := 500
		brightness := 1.0

		device.Call("setBatch", &devices.LightDeviceState{
			OnOff:      &p.onOffState,
			Color:      colorState,
			Transition: &transition,
			Brightness: &brightness,
		}, nil, 0)

	}
}

func (p *LightPane) Render() (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))

	c := colorful.Hsv(p.colorState*360, 1, 1)

	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.RGBA{uint8(c.R * 255), uint8(c.G * 255), uint8(c.B * 255), 255}}, image.ZP, draw.Src)

	var frame *image.RGBA
	if p.onOffState {
		frame = p.onImage.GetNextFrame()
	} else {
		frame = p.offImage.GetNextFrame()
	}

	draw.Draw(canvas, canvas.Bounds(), frame, image.ZP, draw.Over)

	return canvas, nil
}

func (p *LightPane) IsDirty() bool {
	return true
}

type throttle struct {
	delay time.Duration
	busy  bool
}

func (t *throttle) try() bool {
	if t.busy {
		return false
	}

	t.busy = true
	go func() {
		time.Sleep(t.delay)
		t.busy = false
	}()
	return true
}
