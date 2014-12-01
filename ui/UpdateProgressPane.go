package ui

import (
	"image"
	"image/draw"

	"github.com/ninjasphere/go-gestic"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

type UpdateProgressPane struct {
	progressImage util.Image
	loopingImage  util.Image
	progress      float64
}

func NewUpdateProgressPane(progressImage string, loopingImage string) *UpdateProgressPane {
	return &UpdateProgressPane{
		progressImage: util.LoadImage(progressImage),
		loopingImage:  util.LoadImage(loopingImage),
	}
}

func (p *UpdateProgressPane) Gesture(gesture *gestic.GestureData) {
}

func (p *UpdateProgressPane) Render() (*image.RGBA, error) {
	frame := image.NewRGBA(image.Rect(0, 0, 16, 16))

	draw.Draw(frame, frame.Bounds(), p.loopingImage.GetNextFrame(), image.Point{0, 0}, draw.Src)
	draw.Draw(frame, frame.Bounds(), p.progressImage.GetPositionFrame(p.progress, true), image.Point{0, 0}, draw.Src)

	return frame, nil
}

func (p *UpdateProgressPane) IsDirty() bool {
	return true
}
