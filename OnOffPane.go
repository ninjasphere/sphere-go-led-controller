package led

import "image"

type OnOffPane struct {
	state bool

	onImage  *Image
	offImage *Image

	onStateChange func(bool)
}

func NewOnOffPane(onImage string, offImage string, onStateChange func(bool)) *OnOffPane {
	return &OnOffPane{
		onImage:       loadImage(onImage),
		offImage:      loadImage(offImage),
		onStateChange: onStateChange,
	}
}

func (p *OnOffPane) SetState(state bool) {
	p.state = state
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
