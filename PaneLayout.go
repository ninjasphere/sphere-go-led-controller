package led

import (
	"image"
	"image/draw"
	"math"
	"time"

	"github.com/ninjasphere/go-ninja/logger"
)

var log = logger.GetLogger("PaneLayout")

const WIDTH = 16
const HEIGHT = 16

type PaneLayout struct {
	image       *image.RGBA
	panning     bool
	currentPane int
	targetPane  int
	panes       []Pane
	tween       *Tween

	fps *Tick
}

func NewPaneLayout() *PaneLayout {
	pane := &PaneLayout{
		image: image.NewRGBA(image.Rect(0, 0, 16, 16)),
		fps:   &Tick{},
	}
	pane.fps.start()
	return pane
}

type Pane interface {
	IsDirty() bool
	Render() (*image.RGBA, error)
}

func (l *PaneLayout) AddPane(pane Pane) {
	l.panes = append(l.panes, pane)
}

func (l *PaneLayout) IsDirty() bool {
	return true
}

func (l *PaneLayout) Render() (*image.RGBA, error) {
	l.fps.tick()
	var position = 0
	if l.tween != nil {
		var done bool
		position, done = l.tween.Update()
		if done {
			l.tween = nil
			l.currentPane = l.targetPane
			position = 0
		}
	}

	log.Infof("Rendering pane %d with pixel offset %d and panning to %d", l.currentPane, position, l.targetPane)

	// Render the current image at the current position
	currentImage, err := l.panes[l.currentPane].Render()
	if err != nil {
		return nil, err
	}

	draw.Draw(l.image, l.image.Bounds(), currentImage, image.Point{position, 0}, draw.Src)

	if position != 0 {
		// We have the target pane to draw too

		targetImage, err := l.panes[l.targetPane].Render()
		if err != nil {
			return nil, err
		}

		var targetPosition int
		if position < 0 {
			// Panning right
			targetPosition = WIDTH + position
		} else {
			// Panning left
			targetPosition = position - WIDTH
		}

		draw.Draw(l.image, l.image.Bounds(), targetImage, image.Point{targetPosition, 0}, draw.Src)
	}

	return l.image, nil
}

func (l *PaneLayout) PanLeft() {
	l.panBy(-1)
}

func (l *PaneLayout) PanRight() {
	l.panBy(1)
}

func (l *PaneLayout) panBy(delta int) {
	l.currentPane = l.targetPane
	l.targetPane += delta
	if l.targetPane < 0 {
		l.targetPane = len(l.panes) + l.targetPane
	}
	if l.targetPane > (len(l.panes) - 1) {
		l.targetPane = l.targetPane - (len(l.panes) - 1)
	}

	log.Infof("panning from pane %d to %d", l.currentPane, l.targetPane)

	l.tween = &Tween{
		From:     0,
		Start:    time.Now(),
		Duration: time.Millisecond * 800,
	}

	if delta > 0 {
		l.tween.To = WIDTH
	} else {
		l.tween.To = -WIDTH
	}

}

type Tween struct {
	From     int
	To       int
	Ease     func(value float64) float64
	Start    time.Time
	Duration time.Duration
}

func (t *Tween) Update() (int, bool) {
	position := float64(time.Now().Sub(t.Start)) / float64(t.Duration)

	if position > 1 {
		// we're done
		return t.To, true
	}

	if t.Ease != nil {
		position = t.Ease(position)
	}

	value := int(math.Floor((float64(t.To-t.From) * position) + float64(t.From)))

	return value, value == t.To
}

type Tick struct {
	count int
}

func (t *Tick) tick() {
	t.count++
}

func (t *Tick) start() {
	go func() {
		for {
			time.Sleep(time.Second)
			log.Infof("Ops/s %d", t.count)
			t.count = 0
		}
	}()
}
