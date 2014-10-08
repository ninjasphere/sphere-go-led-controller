package ui

import (
	"image"

	"github.com/ninjasphere/go-gestic"
)

type ImagePane struct {
	image *Image
}

func NewImagePane(image string) *ImagePane {
	return &ImagePane{
		image: loadImage(image),
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
