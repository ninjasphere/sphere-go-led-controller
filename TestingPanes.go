package led

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/ninjasphere/driver-go-gestic/gestic"
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

//	//blue := color.RGBA{0, 0, 255, 255}
