package ui

import (
	"image"
	"image/color"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
)

var systemPaneEnabled = config.Bool(false, "led.systempane.enabled")

var colors = map[string]*color.RGBA{
	"white":  &color.RGBA{255, 255, 255, 0},
	"black":  &color.RGBA{0, 0, 0, 0},
	"red":    &color.RGBA{255, 0, 0, 255},
	"green":  &color.RGBA{0, 255, 0, 255},
	"blue":   &color.RGBA{0, 0, 255, 255},
	"yellow": &color.RGBA{255, 255, 0, 255},
}

type SystemPane struct {
	code  string
	color string
}

func NewSystemPane() Pane {
	return &SystemPane{
		code:  "0000",
		color: "green",
	}
}

func (p *SystemPane) IsEnabled() bool {
	return systemPaneEnabled
}

func (p *SystemPane) KeepAwake() bool {
	return true
}

func (p *SystemPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	O4b03b.Font.DrawString(img, 1, 5, p.code, colors[p.color])
	return img, nil
}

func (p *SystemPane) Gesture(*gestic.GestureMessage) {
}
