package ui

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

const wakeDuration = time.Millisecond * 500
const sleepTimeout = time.Second * 10
const sleepDuration = time.Second * 3

type PaneLayout struct {
	currentPane int
	targetPane  int
	panes       []Pane
	lastGesture time.Time

	panTween *Tween

	awake     bool
	fadeTween *Tween
	wake      chan (bool)

	log *logger.Logger

	fps      *Tick
	gestures *Tick
}

func NewPaneLayout() (*PaneLayout, chan (bool)) {
	pane := &PaneLayout{
		fps: &Tick{
			name: "Pane FPS",
		},
		gestures: &Tick{
			name: "Gestures/sec",
		},
		wake: make(chan bool),
		log:  logger.GetLogger("PaneLayout"),
	}
	pane.fps.start()
	pane.gestures.start()

	gestic.ResetDevice()

	firstGesture := true

	// Check for sleep timeout
	go func() {
		for {
			time.Sleep(time.Millisecond * 50)
			if pane.awake && time.Since(pane.lastGesture) > sleepTimeout {
				pane.Sleep()
			}
		}
	}()

	reader := gestic.NewReader(logger.GetLogger("Gestic"), func(g *gestic.GestureData) {

		if firstGesture {
			firstGesture = false
			return
		}
		pane.gestures.tick()

		pane.lastGesture = time.Now()

		//spew.Dump(g)

		// If we're asleep, wake up
		if !pane.awake {
			pane.Wake()
			return
		}

		// Ignore all gestures while we're fading in or out
		if pane.fadeTween == nil {

			if g.Gesture.Name() == "EastToWest" {
				pane.panBy(1)
				pane.log.Infof("East to west, panning by 1")
			}

			if g.Gesture.Name() == "WestToEast" {
				pane.panBy(-1)
				pane.log.Infof("West to east, panning by -1")
			}

			if pane.panTween == nil {
				pane.panes[pane.currentPane].Gesture(g)
			}
		}
	})

	go reader.Start()

	return pane, pane.wake
}

type Pane interface {
	Render() (*image.RGBA, error)
	Gesture(*gestic.GestureData)
}

func (l *PaneLayout) Wake() {

	l.log.Infof("Waking up")

	currentFade := 0.0

	if l.fadeTween != nil {
		currentFade, _ = l.fadeTween.Update()
	}

	l.awake = true

	l.fadeTween = &Tween{
		From:     currentFade,
		To:       1,
		Start:    time.Now(),
		Duration: wakeDuration, // Alter duration if not starting at 0?
		Ease:     easeOutQuint,
	}
	l.wake <- true
}

func (l *PaneLayout) Sleep() {
	l.log.Infof("Going to sleep")
	l.awake = false

	l.fadeTween = &Tween{
		From:     1,
		To:       0,
		Start:    time.Now(),
		Duration: sleepDuration,
	}
}

func (l *PaneLayout) AddPane(pane Pane) {
	l.panes = append(l.panes, pane)
}

func (l *PaneLayout) IsDirty() bool {
	return true
}

func (l *PaneLayout) Render() (*image.RGBA, chan (bool), error) {

	frame := image.NewRGBA(image.Rect(0, 0, width, height))

	l.fps.tick()

	if l.fadeTween != nil {
		_, done := l.fadeTween.Update()

		if done {
			l.fadeTween = nil
		}
	}

	if !l.awake && l.fadeTween == nil {
		log.Println("Sending blank frame and wake chan")
		return frame, l.wake, nil
	}

	var position = 0
	if l.panTween != nil {
		var done bool
		floatPosition, done := l.panTween.Update()
		position = int(math.Floor(floatPosition))
		if done {
			l.panTween = nil
			l.currentPane = l.targetPane
			position = 0
		}
	}

	if position != 0 {
		l.log.Infof("Rendering pane %d with pixel offset %d and panning to %d", l.currentPane, position, l.targetPane)
	}

	// Render the current image at the current position
	currentImage, err := l.panes[l.currentPane].Render()

	if err != nil {
		return nil, nil, err
	}

	draw.Draw(frame, frame.Bounds(), currentImage, image.Point{position, 0}, draw.Src)

	if position != 0 {
		// We have the target pane to draw too

		targetImage, err := l.panes[l.targetPane].Render()
		if err != nil {
			return nil, nil, err
		}

		var targetPosition int
		if position < 0 {
			// Panning right
			targetPosition = width + position
		} else {
			// Panning left
			targetPosition = position - width
		}

		draw.Draw(frame, frame.Bounds(), targetImage, image.Point{targetPosition, 0}, draw.Src)
	}

	if l.fadeTween != nil {
		// We're fading in or out...

		fade, _ := l.fadeTween.Update()
		if fade > 1 {
			fade = 1
		}

		for i := 0; i < len(frame.Pix); i = i + 4 {
			//log.Println(i)
			frame.Pix[i] = uint8(float64(frame.Pix[i]) * fade)
			frame.Pix[i+1] = uint8(float64(frame.Pix[i+1]) * fade)
			frame.Pix[i+2] = uint8(float64(frame.Pix[i+2]) * fade)
		}
	}

	return frame, nil, nil
}

func (l *PaneLayout) panBy(delta int) {
	l.currentPane = l.targetPane
	l.targetPane += delta
	if l.targetPane < 0 {
		l.targetPane = (len(l.panes) - 1)
	}
	if l.targetPane > (len(l.panes) - 1) {
		l.targetPane = 0
	}

	l.log.Infof("panning from pane %d to %d", l.currentPane, l.targetPane)

	l.panTween = &Tween{
		From:     0,
		Start:    time.Now(),
		Duration: time.Millisecond * 250,
	}

	if delta > 0 {
		l.panTween.To = width
	} else {
		l.panTween.To = -width
	}

}

type Tween struct {
	From     float64
	To       float64
	Ease     func(t float64, b float64, c float64, d float64) float64
	Start    time.Time
	Duration time.Duration
}

func (t *Tween) Update() (float64, bool) {
	position := float64(time.Now().Sub(t.Start)) / float64(t.Duration)

	if position > 1 {
		// we're done
		return float64(t.To), true
	}

	if t.Ease != nil {
		position = t.Ease(0.0, t.From, position-t.From, float64(t.Duration))
	}

	value := (float64(t.To-t.From) * position) + float64(t.From)

	return value, value == t.To
}

// from http://gizma.com/easing/
// t - start time
// b - start value
// c - change in value
// d - duration
func easeOutQuint(t float64, b float64, c float64, d float64) float64 {
	t /= d
	t--
	return c*(t*t*t*t*t+1) + b
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
