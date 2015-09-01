package ui

import (
	"image"
	"math"

	"github.com/ninjasphere/draw2d"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/config"
)

var enableGesturePane = config.Bool(false, "led.gestures.testPane")

type GesturePane struct {
	last *gestic.GestureMessage
}

func NewGesturePane() *GesturePane {
	return &GesturePane{}
}

func (p *GesturePane) IsEnabled() bool {
	return enableGesturePane
}

func (p *GesturePane) KeepAwake() bool {
	return false
}

func (p *GesturePane) Gesture(gesture *gestic.GestureMessage) {
	p.last = gesture
}

func (p *GesturePane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	if p.last != nil {

		x := math.Floor(float64(p.last.Position.X)/float64(0xffff)*float64(16)) + 0.5
		y := math.Floor(float64(p.last.Position.Y)/float64(0xffff)*float64(16)) + 0.5
		z := math.Floor(float64(p.last.Position.Z)/float64(0xffff)*float64(16)) + 0.5

		r, _ := colorful.Hex("#FF000")
		g, _ := colorful.Hex("#00FF00")
		b, _ := colorful.Hex("#0000FF")

		gc := draw2d.NewGraphicContext(img)

		gc.SetStrokeColor(r)
		gc.MoveTo(0, x)
		gc.LineTo(16, x)
		gc.Stroke()

		gc.SetStrokeColor(g)
		gc.MoveTo(y, 0)
		gc.LineTo(y, 16)
		gc.Stroke()

		gc.SetStrokeColor(b)
		gc.MoveTo(16-z, 0)
		gc.LineTo(16-z, 16)
		gc.Stroke()
	}

	return img, nil
}

func (p *GesturePane) IsDirty() bool {
	return true
}
