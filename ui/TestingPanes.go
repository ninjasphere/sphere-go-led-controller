package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

type ColorPane struct {
	image  *image.RGBA
	color  func() color.Color
	draw   func()
	bounds func() image.Rectangle
}

func NewColorPane(in color.Color) *ColorPane {
	pane := &ColorPane{
		color: func() color.Color {
			return in
		},
		image: image.NewRGBA(image.Rect(0, 0, width, height)),
	}
	pane.draw = func() {
		draw.Draw(pane.image, pane.bounds(), &image.Uniform{pane.color()}, image.ZP, draw.Src)
	}
	pane.bounds = func() image.Rectangle {
		return pane.image.Bounds()
	}
	return pane
}

func NewFadingColorPane(in color.Color, d time.Duration) *ColorPane {

	pane := NewColorPane(in)
	start := time.Now()
	pane.color = func() color.Color {
		n := time.Now().Sub(start)
		ratio := 1.0
		if n < d {
			ratio = float64(n) / float64(d)
		}
		r, g, b, a := in.RGBA()
		return color.RGBA{
			R: uint8(uint16((1.0-ratio)*float64(r)) >> 8),
			G: uint8(uint16((1.0-ratio)*float64(g)) >> 8),
			B: uint8(uint16((1.0-ratio)*float64(b)) >> 8),
			A: uint8(a),
		}
	}
	return pane
}

// creates a pane that fades and shrinks towards the center as time progresses
func NewFadingShrinkingColorPane(in color.Color, d time.Duration) *ColorPane {

	pane := NewFadingColorPane(in, d)
	basicDraw := pane.draw
	start := time.Now()
	black := color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 0,
	}

	pane.bounds = func() image.Rectangle {
		n := time.Now().Sub(start)
		dim := 0
		if d > n && d > 0 {
			dim = int(float64(d-n) * 8.0 / float64(d))
		}
		rect := image.Rectangle{
			Min: image.Point{
				X: 8 - dim,
				Y: 8 - dim,
			},
			Max: image.Point{
				X: 8 + dim,
				Y: 8 + dim,
			},
		}
		return rect
	}

	pane.draw = func() {
		draw.Draw(pane.image, pane.image.Bounds(), &image.Uniform{black}, image.ZP, draw.Src)
		basicDraw()
	}

	return pane
}

func (p *ColorPane) Gesture(gesture *gestic.GestureMessage) {

}

func (p *ColorPane) Render() (*image.RGBA, error) {
	p.draw()
	return p.image, nil
}

func (p *ColorPane) IsDirty() bool {
	return false
}

type TextScrollPane struct {
	text      string
	textWidth int
	position  int
	start     time.Time
}

func NewTextScrollPane(text string) *TextScrollPane {

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	width := O4b03b.Font.DrawString(img, 0, 0, text, color.Black)
	//log.Infof("Text '%s' width: %d", text, width)

	return &TextScrollPane{
		text:      text,
		textWidth: width,
		position:  17,
		start:     time.Now(),
	}
}

func (p *TextScrollPane) Gesture(gesture *gestic.GestureMessage) {

}

func (p *TextScrollPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	p.position = p.position - 1
	if p.position < -p.textWidth {
		p.position = 17
	}

	//log.Printf("Rendering text '%s' at position %d", p.text, p.position)

	O4b03b.Font.DrawString(img, p.position, 0, p.text, color.White)

	elapsed := time.Now().Sub(p.start)

	elapsedSeconds := int(elapsed.Seconds())

	O4b03b.Font.DrawString(img, 0, 5, "Hey! :)", color.RGBA{0, 255, 255, 255})

	O4b03b.Font.DrawString(img, 0, 11, "02", color.RGBA{255, 0, 0, 255})

	O4b03b.Font.DrawString(img, 9, 11, fmt.Sprintf("%0d", elapsedSeconds), color.RGBA{255, 0, 0, 255})

	O4b03b.Font.DrawString(img, 8, 11, ":", color.RGBA{255, 255, 255, 255})

	return img, nil
}

func (p *TextScrollPane) IsDirty() bool {
	return true
}

type PairingCodePane struct {
	text      string
	textWidth int
	image     util.Image
}

func NewPairingCodePane(text string) *PairingCodePane {

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	width := O4b03b.Font.DrawString(img, 0, 0, text, color.Black)
	//log.Printf("Text '%s' width: %d", text, width)

	return &PairingCodePane{
		text:      text,
		textWidth: width,
		image:     util.LoadImage(util.ResolveImagePath("code-underline.gif")),
	}
}

func (p *PairingCodePane) Gesture(gesture *gestic.GestureMessage) {

}

func (p *PairingCodePane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	//log.Printf("Rendering text '%s'", p.text)

	start := 8 - int((float64(p.textWidth) / float64(2)))

	draw.Draw(img, img.Bounds(), p.image.GetNextFrame(), image.ZP, draw.Over)

	O4b03b.Font.DrawString(img, start, 4, p.text, color.White)

	return img, nil
}

func (p *PairingCodePane) IsDirty() bool {
	return true
}
