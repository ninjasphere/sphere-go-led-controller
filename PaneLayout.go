package led

import (
	"image"
	"image/draw"
	"log"
	"math"
	"time"

	"github.com/ninjasphere/driver-go-gestic/gestic"
	"github.com/ninjasphere/go-ninja/logger"
)

const width = 16
const height = 16

type PaneLayout struct {
	image       *image.RGBA
	panning     bool
	currentPane int
	targetPane  int
	panes       []Pane
	tween       *Tween
	log         *logger.Logger

	fps *Tick
}

func NewPaneLayout() *PaneLayout {
	pane := &PaneLayout{
		image: image.NewRGBA(image.Rect(0, 0, width, height)),
		fps: &Tick{
			name: "Pane FPS",
		},
		log: logger.GetLogger("PaneLayout"),
	}
	pane.fps.start()

	gestic.ResetDevice()

	reader := gestic.NewReader(logger.GetLogger("Gestic"), func(g *gestic.GestureData) {

		log.Printf("Gesture %s", g.Gesture.Name())

		if g.Gesture.Name() == "EastToWest" {
			pane.PanLeft()
		}

		if g.Gesture.Name() == "WestToEast" {
			pane.PanRight()
		}

		if pane.tween == nil {
			pane.panes[pane.currentPane].Gesture(g)
		}
	})

	go reader.Start()

	return pane
}

type Pane interface {
	IsDirty() bool
	Render() (*image.RGBA, error)
	Gesture(*gestic.GestureData)
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

	l.log.Infof("Rendering pane %d with pixel offset %d and panning to %d", l.currentPane, position, l.targetPane)

	// Render the current image at the current position
	currentImage, err := l.panes[l.currentPane].Render()

	if err != nil {
		return nil, err
	}

	if position != 0 {
		fade := (16.00 - math.Abs(float64(position))) / 16.00
		log.Println("Fade amount %f", fade)

		faded := image.NewRGBA(image.Rect(0, 0, width, height))

		for i := 0; i < len(currentImage.Pix); i = i + 4 {
			//log.Println(i)
			faded.Pix[i] = uint8(float64(currentImage.Pix[i]) * fade)
			faded.Pix[i+1] = uint8(float64(currentImage.Pix[i+1]) * fade)
			faded.Pix[i+2] = uint8(float64(currentImage.Pix[i+2]) * fade)
		}

		draw.Draw(l.image, l.image.Bounds(), faded, image.Point{0, 0}, draw.Src)
	} else {
		draw.Draw(l.image, l.image.Bounds(), currentImage, image.Point{position, 0}, draw.Src)
	}

	if position != 0 {
		// We have the target pane to draw too

		targetImage, err := l.panes[l.targetPane].Render()
		if err != nil {
			return nil, err
		}

		var targetPosition int
		if position < 0 {
			// Panning right
			targetPosition = width + position
		} else {
			// Panning left
			targetPosition = position - width
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

	l.log.Infof("panning from pane %d to %d", l.currentPane, l.targetPane)

	l.tween = &Tween{
		From:     0,
		Start:    time.Now(),
		Duration: time.Millisecond * 250,
	}

	if delta > 0 {
		l.tween.To = -width
	} else {
		l.tween.To = width
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
	name  string
}

func (t *Tick) tick() {
	t.count++
}

func (t *Tick) start() {
	go func() {
		for {
			time.Sleep(time.Second)
			log.Printf("%s - %d", t.name, t.count)
			t.count = 0
		}
	}()
}
