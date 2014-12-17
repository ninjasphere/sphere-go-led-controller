package ui

import (
	"image"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

type ImagePane struct {
	image util.Image
}

func NewImagePane(image string) *ImagePane {
	return &ImagePane{
		image: util.LoadImage(image),
	}
}

func (p *ImagePane) IsEnabled() bool {
	return true
}

func (p *ImagePane) Gesture(gesture *gestic.GestureMessage) {
}

func (p *ImagePane) Render() (*image.RGBA, error) {
	return p.image.GetNextFrame(), nil
}

func (p *ImagePane) IsDirty() bool {
	return true
}
