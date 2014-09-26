package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"time"

	"github.com/ninjasphere/go-gestic"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
)

type ColorPane struct {
	image  *image.RGBA
	colors []color.Color
}

func NewColorPane(colors ...color.Color) *ColorPane {
	return &ColorPane{
		colors: colors,
		image:  image.NewRGBA(image.Rect(0, 0, width, height)),
	}
}

func (p *ColorPane) Gesture(gesture *gestic.GestureData) {

}

func (p *ColorPane) Render() (*image.RGBA, error) {
	draw.Draw(p.image, p.image.Bounds(), &image.Uniform{p.colors[0]}, image.ZP, draw.Src)
	return p.image, nil
}

func (p *ColorPane) IsDirty() bool {
	return len(p.colors) > 1
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
	log.Printf("Text '%s' width: %d", text, width)

	return &TextScrollPane{
		text:      text,
		textWidth: width,
		position:  17,
		start:     time.Now(),
	}
}

func (p *TextScrollPane) Gesture(gesture *gestic.GestureData) {

}

func (p *TextScrollPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	p.position = p.position - 1
	if p.position < -p.textWidth {
		p.position = 17
	}

	log.Printf("Rendering text '%s' at position %d", p.text, p.position)

	O4b03b.Font.DrawString(img, p.position, 0, p.text, color.White)

	elapsed := time.Now().Sub(p.start)

	elapsedSeconds := int(elapsed.Seconds())

	O4b03b.Font.DrawString(img, 0, 5, "Elliot", color.RGBA{0, 255, 255, 255})

	O4b03b.Font.DrawString(img, 0, 11, "02", color.RGBA{255, 0, 0, 255})

	O4b03b.Font.DrawString(img, 9, 11, fmt.Sprintf("%0d", elapsedSeconds), color.RGBA{255, 0, 0, 255})

	O4b03b.Font.DrawString(img, 8, 11, ":", color.RGBA{255, 255, 255, 255})

	return img, nil
}

func (p *TextScrollPane) IsDirty() bool {
	return true
}

//	//blue := color.RGBA{0, 0, 255, 255}
