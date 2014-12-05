package ui

import (
	"encoding/json"
	"image"
	"math"
	"sync"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/sphere-go-led-controller/util"

	"github.com/ninjasphere/go-ninja/logger"
)

// How long after the last airwheel before we hide the volume display
const volumeModeReset = time.Second
const ignoreTap = time.Millisecond * 300

const volumeRate = 2 // Max number of volume calls per second

type MediaPane struct {
	log  *logger.Logger
	conn *ninja.Connection

	volumeMode      bool
	volumeModeReset *time.Timer

	lastAirWheelTime time.Time
	lastAirWheel     *uint8

	volume         float64
	volumeImage    util.Image
	muteImage      util.Image
	lastVolumeTime time.Time
	lastSentVolume float64

	ignoringTap    bool
	ignoreTapTimer *time.Timer
	playingState   string
	playImage      util.Image
	pauseImage     util.Image
	stopImage      util.Image
	nextImage      util.Image

	gestureSync *sync.Mutex

	controlDevices map[string]*ninja.ServiceClient
	volumeDevices  map[string]*ninja.ServiceClient
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

	pane := &MediaPane{
		log:            log,
		volumeDevices:  make(map[string]*ninja.ServiceClient),
		controlDevices: make(map[string]*ninja.ServiceClient),
		conn:           conn,
		gestureSync:    &sync.Mutex{},

		volumeImage: util.LoadImage(images.Volume),
		muteImage:   util.LoadImage(images.Mute),
		playImage:   util.LoadImage(images.Play),
		pauseImage:  util.LoadImage(images.Pause),
		stopImage:   util.LoadImage(images.Stop),
		nextImage:   util.LoadImage(images.Next),

		playingState: "stopped",

		lastVolumeTime: time.Now(),
		//lastAirWheelTime: time.Now(),
	}

	e := func(state string) func(params *json.RawMessage, values map[string]string) bool {
		return func(params *json.RawMessage, values map[string]string) bool {
			if !pane.ignoringTap {
				pane.log.Infof("Received control event. Setting playing state to %s", state)
				pane.playingState = state
			}
			return true
		}
	}

	getChannelServicesContinuous("mediaplayer", "media-control", func(devices []*ninja.ServiceClient, err error) {

		if err != nil {
			log.Infof("Failed to update control devices: %s", err)
		} else {
			for _, device := range devices {

				if _, ok := pane.controlDevices[device.Topic]; !ok {
					// New Device
					log.Infof("Got control device: %s", device.Topic)

					pane.controlDevices[device.Topic] = device

					device.OnEvent("playing", e("playing"))
					device.OnEvent("buffering", e("playing"))
					device.OnEvent("paused", e("paused"))
					device.OnEvent("stopped", e("stopped"))
				}
			}
		}

	})

	getChannelServicesContinuous("mediaplayer", "volume", func(devices []*ninja.ServiceClient, err error) {
		if err != nil {
			log.Infof("Failed to update volume devices: %s", err)
		} else {
			for _, device := range devices {

				if _, ok := pane.volumeDevices[device.Topic]; !ok {
					// New device
					log.Infof("Got volume device: %s", device.Topic)

					pane.volumeDevices[device.Topic] = device

					device.OnEvent("state", func(params *json.RawMessage, values map[string]string) bool {
						if time.Since(pane.lastVolumeTime) > time.Millisecond*300 {

							var volume channels.VolumeState
							err := json.Unmarshal(*params, &volume)
							if err != nil {
								pane.log.Infof("Failed to unmarshal volume from %s error:%s", *params, err)
							}
							pane.volume = *volume.Level
						}
						return true
					})
				}
			}
		}
	})

	pane.volumeModeReset = time.AfterFunc(0, func() {
		pane.volumeMode = false
	})

	pane.ignoreTapTimer = time.AfterFunc(0, func() {
		pane.ignoringTap = false
	})

	return pane
}

func (p *MediaPane) Gesture(gesture *gestic.GestureMessage) {
	p.gestureSync.Lock()
	defer p.gestureSync.Unlock()

	if p.lastAirWheel == nil || gesture.AirWheel.Counter != int(*p.lastAirWheel) {

		p.volumeMode = true
		p.volumeModeReset.Reset(volumeModeReset)

		if time.Since(p.lastAirWheelTime) > time.Millisecond*300 {
			p.lastAirWheel = nil
		}

		p.lastAirWheelTime = time.Now()

		//p.log.Debugf("Airwheel: %d", gesture.AirWheel.AirWheelVal)

		if p.lastAirWheel != nil {
			offset := int(gesture.AirWheel.Counter) - int(*p.lastAirWheel)

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

			volume = math.Max(volume, 0)
			volume = math.Min(volume, 1)

			p.log.Debugf("Adjusted volume %f:", volume)

			p.volume = volume

			if p.lastSentVolume != volume {
				if time.Since(p.lastVolumeTime) < time.Millisecond*500 {
					p.log.Debugf("Volume rate limited")
				} else {
					p.lastVolumeTime = time.Now()
					p.log.Debugf("Volume NOT rate limited")
					p.SendVolume()
				}
			}
		}

		val := uint8(gesture.AirWheel.Counter)
		p.lastAirWheel = &val
		//spew.Dump("last2", p.lastAirWheel)
	}

	if !p.ignoringTap && gesture.Tap.Active() {
		p.log.Infof("Tap!")

		p.ignoringTap = true
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
		method = "play"
	case "paused":
		method = "pause"
	}

	for _, device := range p.controlDevices {
		device.Call(method, nil, nil, 0)
	}
	p.playingState = state
}

func (p *MediaPane) SendVolume() {
	p.log.Debugf("New volume %f:", p.volume)
	//	p.volume = volume

	p.lastSentVolume = p.volume
	for _, device := range p.volumeDevices {
		device.Call("set", channels.VolumeState{Level: &p.volume}, nil, 0)
	}
	//p.onStateChange(state)
}

func (p *MediaPane) Render() (*image.RGBA, error) {

	if p.volumeMode {
		if p.volume > 0 {
			return p.volumeImage.GetPositionFrame(1-p.volume, true), nil
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
