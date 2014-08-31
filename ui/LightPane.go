package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"net/rpc"
	"strings"
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/driver-go-gestic/gestic"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/logger"
)

type LightPane struct {
	log *logger.Logger

	onOffDevices []*rpc.Client
	colorDevices []*rpc.Client

	onOffState            bool
	onOnOffStateChange    func(bool)
	ignoringOnOffGestures bool

	colorState         float64
	onColorStateChange func(float64)

	onImage  *Image
	offImage *Image
}

const colorRotateSpeed = 0.0025

func NewLightPane(offImage string, onImage string, onOnOffStateChange func(bool), onColorStateChange func(float64), mqtt *mqtt.MqttClient) *LightPane {

	onOffDevices, err := getChannelClients("light", "on-off", mqtt)
	if err != nil {
		log.Fatalf("Failed to get on-off devices", err)
	}

	colorDevices, err := getChannelClients("light", "core.batching", mqtt)
	if err != nil {
		log.Fatalf("Failed to get on-off devices", err)
	}

	log := logger.GetLogger("LightPane")
	log.Infof("Pane got %d on/off devices", len(onOffDevices))

	log.Infof("Pane got %d color devices", len(colorDevices))

	return &LightPane{
		onImage:            loadImage(onImage),
		offImage:           loadImage(offImage),
		onOnOffStateChange: onOnOffStateChange,
		onColorStateChange: onColorStateChange,
		log:                log,
		onOffDevices:       onOffDevices,
		colorDevices:       colorDevices,
	}
}

func (p *LightPane) Gesture(gesture *gestic.GestureData) {

	if gesture.Coordinates.X > 0 && gesture.Coordinates.Y > 0 && gesture.Coordinates.Z > 0 {

		col := p.colorState + colorRotateSpeed
		if col >= 1 {
			col = 0
		}
		p.SetColorState(col)
		p.log.Infof("Color wheel %f", p.colorState)
	}

	if p.ignoringOnOffGestures {
		return
	}

	if strings.Contains(gesture.Touch.Name(), "Tap") {
		p.log.Infof("Tap!")

		p.ignoringOnOffGestures = true

		go func() {
			time.Sleep(time.Millisecond * 250)
			p.ignoringOnOffGestures = false
		}()

		p.SetOnOffState(!p.onOffState)
	}
}

func (p *LightPane) SetOnOffState(state bool) {
	p.onOffState = state
	for _, device := range p.onOffDevices {
		if state {
			_ = device.Go("turnOn", nil, nil, nil)
		} else {
			_ = device.Go("turnOff", nil, nil, nil)
		}
	}
	p.onOnOffStateChange(state)
}

func (p *LightPane) SetColorState(state float64) {
	p.colorState = state

	sat := 0.6

	for _, device := range p.colorDevices {

		colorState := &channels.ColorState{
			Mode:       "hue",
			Hue:        &p.colorState,
			Saturation: &sat,
		}
		transition := 300
		brightness := 1.0

		_ = device.Go("setBatch", &devices.LightDeviceState{
			Color:      colorState,
			Transition: &transition,
			Brightness: &brightness,
		}, nil, nil)

	}
	p.onColorStateChange(state)
}

func (p *LightPane) Render() (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))

	c := colorful.Hsv(p.colorState*360, 1, 1)

	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.RGBA{uint8(c.R * 255), uint8(c.G * 255), uint8(c.B * 255), 255}}, image.ZP, draw.Src)

	var frame *image.RGBA
	if p.onOffState {
		frame = p.onImage.GetFrame()
	} else {
		frame = p.offImage.GetFrame()
	}

	draw.Draw(canvas, canvas.Bounds(), frame, image.ZP, draw.Over)

	return canvas, nil
}

func (p *LightPane) IsDirty() bool {
	return true
}
