package ui

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

type PairingColorPane struct {
	image util.Image
	color color.Color
}

func NewPairingColorPane(maskImage string, color color.Color) *PairingColorPane {
	return &PairingColorPane{
		image: &util.MaskImage{util.LoadImage(maskImage)},
		color: color,
	}
}

func (p *PairingColorPane) IsEnabled() bool {
	return true
}

func (p *PairingColorPane) KeepAwake() bool {
	return false
}

func (p *PairingColorPane) Gesture(gesture *gestic.GestureMessage) {
}

func (p *PairingColorPane) Render() (*image.RGBA, error) {

	frame := image.NewRGBA(image.Rect(0, 0, width, height))

	draw.Draw(frame, frame.Bounds(), &image.Uniform{p.color}, image.ZP, draw.Src)
	draw.Draw(frame, frame.Bounds(), p.image.GetNextFrame(), image.ZP, draw.Over)

	return frame, nil
}

func (p *PairingColorPane) IsDirty() bool {
	return true
}
