package ui

import (
	"image"

	"github.com/ninjasphere/go-gestic"
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

func (p *ImagePane) Gesture(gesture *gestic.GestureData) {
}

func (p *ImagePane) Render() (*image.RGBA, error) {
	return p.image.GetNextFrame(), nil
}

func (p *ImagePane) IsDirty() bool {
	return true
}
