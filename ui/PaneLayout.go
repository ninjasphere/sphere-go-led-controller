package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"sync"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
)

const width = 16
const height = 16

var panDuration = config.MustDuration("led.panDuration")
var wakeTransitionDuration = config.MustDuration("led.wakeTransition")
var sleepTransitionDuration = config.MustDuration("led.sleepTransition")
var sleepTimeout = config.MustDuration("led.sleepTimeout")
var forceAllPanes = config.Bool(false, "led.forceAllPanes")

var logGestures = config.Bool(false, "led.gestures.log")
var enableGestures = config.Bool(true, "led.gestures.enable")

type PaneLayout struct {
	currentPane int
	targetPane  int
	panes       []Pane
	lastGesture time.Time

	panTween *Tween
	panLock  sync.Mutex

	awake     bool
	fadeTween *Tween
	wake      chan (bool)

	log *logger.Logger

	gestures *Tick
}

func NewPaneLayout(fakeGestures bool, conn *ninja.Connection) (*PaneLayout, chan (bool)) {

	// Wait till we're paired and have a site
	for {
		config.MustRefresh()
		if config.HasString("siteId") {
			break
		}
		time.Sleep(time.Second * 2)
	}

	go startSearchTasks(conn)

	pane := &PaneLayout{
		gestures: &Tick{
			name: "Gestures/sec",
		},
		wake: make(chan bool),
		log:  logger.GetLogger("PaneLayout"),
	}
	pane.gestures.start()

	if !fakeGestures {
		g, err := gestic.Open()

		if err != nil {
			pane.log.Warningf("Error enabling gestures: %s\n", err)
		} else {
			//defer g.Close()

			pane.log.Infof("Succesfully connected to GestIC device")

			gestures := g.DataStream()

			go func() {
				for gesture := range gestures {
					//pane.log.Debugf("Gesture latency: %s", time.Since(gesture.Time).String())
					go pane.OnGesture(&gesture)
				}
			}()
		}
	}

	// Check for sleep timeout
	go func() {
		for {
			time.Sleep(time.Millisecond * 50)
			if pane.awake && time.Since(pane.lastGesture) > sleepTimeout {
				pane.Sleep()
			}
		}
	}()

	/*if fakeGestures {
		go func() {
			for {
				time.Sleep(time.Millisecond * 5000)
				gesture := gestic.NewGestureData()
				gesture.Gesture.GestureVal = 2
				pane.OnGesture(gesture)
			}
		}()

		go func() {
			for {
				time.Sleep(time.Millisecond * 10)
				gesture := gestic.NewGestureData()
				gesture.Coordinates.X = 100
				gesture.Coordinates.Y = 100
				gesture.Coordinates.Z = 100
				pane.OnGesture(gesture)
			}
		}()
	}*/

	return pane, pane.wake
}

type Pane interface {
	IsEnabled() bool
	Render() (*image.RGBA, error)
	Gesture(*gestic.GestureMessage)
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
		Duration: wakeTransitionDuration, // Alter duration if not starting at 0?
		Ease:     easeOutQuint,
	}
	l.wake <- true
}

func (l *PaneLayout) OnGesture(g *gestic.GestureMessage) {

	//	x, _ := json.Marshal(g)
	//	l.log.Infof("gesture %s", x)

	if logGestures {
		x, _ := json.Marshal(g)
		fmt.Fprint(os.Stdout, string(x)+"\n")
	}

	/*if !g.AirWheel.Active {
		return
	}*/

	//l.log.Infof("gesture : %v", g)

	// Always skip the first gesture if we haven't had any for ignoreFirstGestureAfterDuration
	/*skip := false

	if time.Now().Sub(l.lastGesture) > ignoreFirstGestureAfterDuration {
		log.Printf("Ignoring first gesture")
		skip = true
	}*/

	l.gestures.tick()

	if !enableGestures {
		return
	}

	l.lastGesture = time.Now()

	//if skip {
	//	return
	//}

	//spew.Dump(g)

	// If we're asleep, wake up
	if !l.awake {
		l.Wake()
		return
	}

	// Ignore all gestures while we're fading in or out
	if l.fadeTween == nil {

		if g.Gesture.Gesture == gestic.GestureFlickEastToWest {
			l.panBy(1)
			l.log.Infof("East to west, panning by 1")
		}

		if g.Gesture.Gesture == gestic.GestureFlickWestToEast {
			l.panBy(-1)
			l.log.Infof("West to east, panning by -1")
		}

		// Don't send gestures to panes while we are panning
		if l.panTween == nil {
			l.panes[l.currentPane].Gesture(g)
		}
	}
}

func (l *PaneLayout) Sleep() {
	l.log.Infof("Going to sleep")
	l.awake = false

	l.fadeTween = &Tween{
		From:     1,
		To:       0,
		Start:    time.Now(),
		Duration: sleepTransitionDuration,
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

	if l.fadeTween != nil {
		_, done := l.fadeTween.Update()

		if done {
			l.fadeTween = nil
		}
	}

	if !l.awake && l.fadeTween == nil {
		l.log.Infof("Sending blank frame and wake chan")
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
	l.panLock.Lock()
	defer l.panLock.Unlock()

	l.currentPane = l.targetPane

	target := l.targetPane + delta
	if target < 0 {
		target = (len(l.panes) - 1)
	}
	if target > (len(l.panes) - 1) {
		target = 0
	}

	// XXX: If there are no enabled panes... this will hang.
	// But that's future Elliot's problem. Or some other poor soul
	// who just wants to go home but some people's spheres keep
	// exploding.
	for {
		l.log.Infof("Checking pane %d", target)

		enabled := l.panes[target].IsEnabled()
		if enabled {
			l.log.Infof("Pane %d is enabled", target)
			break
		}
		if forceAllPanes {
			l.log.Infof("Forcing pane %d to display", target)
			break
		}
		l.log.Infof("Skipping unenabled pane %d", target)
		if delta > 0 {
			target++
		} else {
			target--
		}
		if target < 0 {
			target = (len(l.panes) - 1)
		}
		if target > (len(l.panes) - 1) {
			target = 0
		}
	}

	if l.currentPane == target {
		l.log.Infof("Not panning. As we don't have anywhere else to pan to.")
		return
	}

	l.log.Infof("panning from pane %d to %d", l.currentPane, target)

	l.panTween = &Tween{
		From:     0,
		Start:    time.Now(),
		Duration: panDuration,
	}

	if delta > 0 {
		l.panTween.To = width
	} else {
		l.panTween.To = -width
	}

	l.targetPane = target
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
			log.Infof("%s - %d", t.name, t.count)
			t.count = 0
		}
	}()
}
