package ui

import (
	"image"
	"image/color"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"time"
)

type PairingLayout struct {
	currentPane Pane
	conn        *ninja.Connection
	log         *logger.Logger
}

func NewPairingLayout(c *ninja.Connection) *PairingLayout {
	startSearchTasks(c)

	layout := &PairingLayout{
		log:  logger.GetLogger("PaneLayout"),
		conn: c,
	}
	layout.ShowIcon("loading.gif")

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

func (l *PairingLayout) Render() (*image.RGBA, error) {
	if l.currentPane != nil {
		return l.currentPane.Render()
	}

	return &image.RGBA{}, nil
}
