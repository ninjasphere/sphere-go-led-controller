package ui

import (
	"image"
	"strings"
	"time"

	"github.com/ninjasphere/driver-go-gestic/gestic"
	"github.com/ninjasphere/go-ninja/logger"
)

type OnOffPane struct {
	log *logger.Logger

	state         bool
	onStateChange func(bool)

	onImage  *Image
	offImage *Image

	ignoringGestures bool
}

func NewOnOffPane(onImage string, offImage string, onStateChange func(bool)) *OnOffPane {
	return &OnOffPane{
		onImage:       loadImage(onImage),
		offImage:      loadImage(offImage),
		onStateChange: onStateChange,
		log:           logger.GetLogger("OnOffPane"),
	}
}

func (p *OnOffPane) Gesture(gesture *gestic.GestureData) {
	if p.ignoringGestures {
		return
	}

	if strings.Contains(gesture.Touch.Name(), "Tap") {
		p.log.Infof("Tap!")
		p.SetState(!p.state)

		p.ignoringGestures = true

		go func() {
			time.Sleep(time.Millisecond * 250)
			p.ignoringGestures = false
		}()
	}
}

func (p *OnOffPane) SetState(state bool) {
	p.state = state
	p.onStateChange(state)
}

func (p *OnOffPane) Render() (*image.RGBA, error) {
	if p.state {
		return p.onImage.GetFrame(), nil
	}
	return p.offImage.GetFrame(), nil
}

func (p *OnOffPane) IsDirty() bool {
	return true
}
