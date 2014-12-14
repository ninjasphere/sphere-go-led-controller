package ui

import (
	"image"
	"image/color"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
)

type ClockPane struct{}

func NewClockPane() *ClockPane {
	return &ClockPane{}
}

func (p *ClockPane) Gesture(gesture *gestic.GestureMessage) {

}

func (p *ClockPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	text := time.Now().Format("15:04")

	width := O4b03b.Font.DrawString(img, 0, 0, text, color.Black)

	start := 8 - int((float64(width) / float64(2)))

	O4b03b.Font.DrawString(img, start, 6, text, color.White)

	return img, nil
}

func (p *ClockPane) IsDirty() bool {
	return true
}
