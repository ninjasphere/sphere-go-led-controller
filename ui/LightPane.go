package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"strings"
	"time"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/go-gestic"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/logger"
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

	onImage  *Image
	offImage *Image
}

func NewLightPane(offImage string, onImage string, onOnOffStateChange func(bool), onColorStateChange func(float64), conn *ninja.Connection) *LightPane {

	onOffDevices, err := getChannelServices("light", "on-off", conn)
	if err != nil {
		log.Fatalf("Failed to get on-off devices", err)
	}

	colorDevices, err := getChannelServices("light", "core.batching", conn)
	if err != nil {
		log.Fatalf("Failed to get on-off devices", err)
	}

	log := logger.GetLogger("LightPane")
	//log.Infof("Pane got %d on/off devices", len(onOffDevices))

	log.Infof("Pane got %d color devices", len(colorDevices))

	return &LightPane{
		onImage:            loadImage(onImage),
		offImage:           loadImage(offImage),
		onOnOffStateChange: onOnOffStateChange,
		onColorStateChange: onColorStateChange,
		log:                log,
		onOffDevices:       onOffDevices,
		colorDevices:       colorDevices,
		conn:               conn,
	}
}

func (p *LightPane) Gesture(gesture *gestic.GestureData) {

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

	if strings.Contains(gesture.Touch.Name(), "Tap") {
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
			device.Call("turnOn", nil, nil, time.Second)
		} else {
			device.Call("turnOff", nil, nil, time.Second)
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
		}, nil, time.Second)

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
