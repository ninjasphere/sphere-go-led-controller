package ui

import (
	"image"
	"image/color"

	"time"

	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

type PairingLayout struct {
	progressPane *UpdateProgressPane
	currentPane  Pane
	log          *logger.Logger
	drawing      *image.RGBA
}

func NewPairingLayout() *PairingLayout {

	layout := &PairingLayout{
		log:          logger.GetLogger("PaneLayout"),
		progressPane: NewUpdateProgressPane("./images/update-progress.gif", "./images/update-loop.gif"),
	}
	layout.ShowIcon("loading.gif")

	/*go func() {
		time.Sleep(time.Second * 1)
		layout.ShowDrawing()
		for {
			time.Sleep(time.Millisecond * 2)
			update := []uint8{uint8(rand.Intn(16)), uint8(rand.Intn(16)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255))}
			layout.Draw([][]uint8{update})
		}
	}()*/

	return layout
}

func (l *PairingLayout) ShowColor(c color.Color) {
	l.currentPane = NewColorPane(c)
}

func (l *PairingLayout) ShowFadingColor(c color.Color, d time.Duration) {
	l.currentPane = NewFadingColorPane(c, d)
}

func (l *PairingLayout) ShowFadingShrinkingColor(c color.Color, d time.Duration) {
	l.currentPane = NewFadingShrinkingColorPane(c, d)
}

func (l *PairingLayout) ShowCode(text string) {
	l.currentPane = NewPairingCodePane(text)
}

func (l *PairingLayout) ShowIcon(image string) {
	l.currentPane = NewImagePane("./images/" + image)
}

func (l *PairingLayout) ShowUpdateProgress(progress float64) {
	l.progressPane.progress = progress
	l.currentPane = l.progressPane
}

func (l *PairingLayout) ShowDrawing() {
	l.drawing = image.NewRGBA(image.Rect(0, 0, 16, 16))
	l.currentPane = &ImagePane{
		image: util.NewSingleImage(l.drawing),
	}
}

func (l *PairingLayout) Draw(updates *[][]uint8) {
	for _, update := range *updates {
		offset := l.drawing.PixOffset(int(update[0]), int(update[1]))
		l.drawing.Pix[offset] = update[2]   // R
		l.drawing.Pix[offset+1] = update[3] // G
		l.drawing.Pix[offset+2] = update[4] // B
		l.drawing.Pix[offset+3] = 255       // A
	}
}

func (l *PairingLayout) Render() (*image.RGBA, error) {
	if l.currentPane != nil {
		return l.currentPane.Render()
	}

	return &image.RGBA{}, nil
}
