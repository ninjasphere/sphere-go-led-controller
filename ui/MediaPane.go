package ui

import (
	"encoding/json"
	"image"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ninjasphere/go-gestic"
	"github.com/ninjasphere/go-ninja/api"

	"github.com/ninjasphere/go-ninja/logger"
)

// How long after the last airwheel before we hide the volume display
const volumeModeReset = time.Second
const ignoreTap = time.Millisecond * 300

type MediaPane struct {
	log  *logger.Logger
	conn *ninja.Connection

	volumeMode      bool
	volumeModeReset *time.Timer

	lastAirWheelTime time.Time
	lastAirWheel     *uint8

	volume      float64
	volumeImage *Image
	muteImage   *Image

	ignoringTap    bool
	ignoreTapTimer *time.Timer
	playingState   string
	playImage      *Image
	pauseImage     *Image
	stopImage      *Image
	nextImage      *Image

	gestureSync *sync.Mutex

	controlDevices []*ninja.ServiceClient
	volumeDevices  []*ninja.ServiceClient
}

type MediaPaneImages struct {
	Volume string
	Mute   string
	Play   string
	Pause  string
	Stop   string
	Next   string
}

func NewMediaPane(images *MediaPaneImages, conn *ninja.Connection) *MediaPane {
	log := logger.GetLogger("MediaPane")

	controlDevices, err := getChannelServices("MediaPlayer", "media-control", conn)
	if err != nil {
		log.Fatalf("Failed to get media-control devices: %s", err)
	}
	log.Infof("Pane got %d media-control devices", len(controlDevices))

	if len(controlDevices) > 1 {
		log.Infof("WARNING... MORE THAN ONE MEDIA CONTROL DEVICE.... IT WILL ACT WEIRD")
	}

	volumeDevices, err := getChannelServices("MediaPlayer", "volume", conn)
	if err != nil {
		log.Fatalf("Failed to get volume devices: %s", err)
	}
	log.Infof("Pane got %d volume devices", len(volumeDevices))

	pane := &MediaPane{
		log:            log,
		volumeDevices:  volumeDevices,
		controlDevices: controlDevices,
		conn:           conn,
		gestureSync:    &sync.Mutex{},

		volumeImage: loadImage(images.Volume),
		muteImage:   loadImage(images.Mute),
		playImage:   loadImage(images.Play),
		pauseImage:  loadImage(images.Pause),
		stopImage:   loadImage(images.Stop),
		nextImage:   loadImage(images.Next),

		playingState: "stopped",
		//lastAirWheelTime: time.Now(),
	}

	e := func(state string) func(params *json.RawMessage, values map[string]string) bool {
		return func(params *json.RawMessage, values map[string]string) bool {
			pane.log.Infof("Received control event. Setting playing state to %s", state)
			pane.playingState = state
			return true
		}
	}

	for _, device := range controlDevices {
		device.OnEvent("playing", e("playing"))
		device.OnEvent("buffering", e("playing"))
		device.OnEvent("paused", e("paused"))
		device.OnEvent("stopped", e("stopped"))
	}

	pane.volumeModeReset = time.AfterFunc(0, func() {
		pane.volumeMode = false
	})

	pane.ignoreTapTimer = time.AfterFunc(0, func() {
		pane.ignoringTap = false
	})

	return pane
}

func (p *MediaPane) Gesture(gesture *gestic.GestureData) {
	p.gestureSync.Lock()
	defer p.gestureSync.Unlock()

	if gesture.AirWheel.AirWheelVal > 0 {

		p.volumeMode = true
		p.volumeModeReset.Reset(volumeModeReset)

		if time.Since(p.lastAirWheelTime) > time.Millisecond*300 {
			p.lastAirWheel = nil
		}

		p.lastAirWheelTime = time.Now()

		//p.log.Debugf("Airwheel: %d", gesture.AirWheel.AirWheelVal)

		if p.lastAirWheel != nil {
			offset := int(gesture.AirWheel.AirWheelVal) - int(*p.lastAirWheel)

			if offset > 30 {
				offset -= 255
			}

			if offset < -30 {
				offset += 255
			}

			//p.log.Debugf("Airwheel New: %d Offset: %d Last: %d", gesture.AirWheel.AirWheelVal, offset, *p.lastAirWheel)

			//p.log.Debugf("Current volume %f", p.volume)

			//p.log.Debugf("Volume offset %f", float64(offset)/255.0)

			var volume float64 = p.volume + float64(offset)/255.0

			//p.log.Debugf("Adjusted volume %f:", volume)

			volume = math.Max(volume, 0)
			volume = math.Min(volume, 1)

			if p.volume != volume {
				p.SetVolume(volume)
			}
		}

		val := gesture.AirWheel.AirWheelVal
		p.lastAirWheel = &val
		//spew.Dump("last2", p.lastAirWheel)
	}

	if !p.ignoringTap && strings.Contains(gesture.Touch.Name(), "Tap") {
		p.log.Infof("Tap!")

		p.ignoreTapTimer.Reset(ignoreTap)

		switch p.playingState {
		case "stopped":
			p.SetControlState("playing")
		case "playing":
			p.SetControlState("paused")
		case "paused":
			p.SetControlState("playing")
		}

	}

}

func (p *MediaPane) SetControlState(state string) {

	p.log.Debugf("Setting playing state %s:", state)

	var method = ""
	switch state {
	case "stopped":
		method = "stop"
	case "playing":
		method = "pause"
	case "paused":
		method = "play"
	}

	for _, device := range p.controlDevices {
		device.Call(method, []interface{}{}, nil, time.Second)
	}
	p.playingState = state
}

func (p *MediaPane) SetVolume(volume float64) {
	p.log.Debugf("New volume %f:", volume)
	p.volume = volume
	for _, device := range p.volumeDevices {
		device.Call("set", []interface{}{volume}, nil, time.Second)
	}
	//p.onStateChange(state)
}

func (p *MediaPane) Render() (*image.RGBA, error) {

	if p.volumeMode {
		if p.volume > 0 {
			return p.volumeImage.GetPositionFrame(1 - p.volume), nil
		}

		return p.muteImage.GetNextFrame(), nil
	}

	switch p.playingState {
	case "stopped":
		return p.stopImage.GetNextFrame(), nil
	case "playing":
		return p.playImage.GetNextFrame(), nil
	case "paused":
		return p.pauseImage.GetNextFrame(), nil
	}

	return p.stopImage.GetNextFrame(), nil
}

func (p *MediaPane) IsDirty() bool {
	return true
}
