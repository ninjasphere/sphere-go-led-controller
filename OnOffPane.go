package led

import (
	"image"
	"log"
	"strings"
	"time"

	"github.com/ninjasphere/driver-go-gestic/gestic"
)

type OnOffPane struct {
	state bool

	onImage  *Image
	offImage *Image

	onStateChange func(bool)

	ignoringGestures bool
}

func NewOnOffPane(onImage string, offImage string, onStateChange func(bool)) *OnOffPane {
	return &OnOffPane{
		onImage:       loadImage(onImage),
		offImage:      loadImage(offImage),
		onStateChange: onStateChange,
	}
}

func (p *OnOffPane) Gesture(gesture *gestic.GestureData) {
	if p.ignoringGestures {
		log.Println("IGNORING GESTURES")
		return
	}

	if strings.Contains(gesture.Touch.Name(), "Tap") {
		log.Println("GOT A TAP")
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
