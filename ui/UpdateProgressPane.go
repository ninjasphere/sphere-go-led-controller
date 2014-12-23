package ui

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
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

func (p *UpdateProgressPane) IsEnabled() bool {
	return true
}

func (p *UpdateProgressPane) Gesture(gesture *gestic.GestureMessage) {
}

func (p *UpdateProgressPane) Render() (*image.RGBA, error) {
	frame := image.NewRGBA(image.Rect(0, 0, 16, 16))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}}, image.ZP, draw.Src)

	draw.Draw(frame, frame.Bounds(), p.loopingImage.GetNextFrame(), image.Point{0, 0}, draw.Over)
	draw.Draw(frame, frame.Bounds(), p.progressImage.GetPositionFrame(p.progress, true), image.Point{0, 0}, draw.Over)

	return frame, nil
}

func (p *UpdateProgressPane) IsDirty() bool {
	return true
}
