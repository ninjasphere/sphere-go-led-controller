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
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/sphere-go-led-controller/util"

	"github.com/ninjasphere/go-ninja/logger"
)

var volumeModeReset = config.MustDuration("led.media.volumeModeReset")
var mediaTapTimeout = config.MustDuration("led.media.tapInterval")
var volumeInterval = config.MustDuration("led.media.volumeInterval")
var airWheelReset = config.MustDuration("led.media.airWheelReset")

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

var mediaImages = MediaPaneImages{
	Volume: util.ResolveImagePath(config.MustString("led.media.images.volume")),
	Mute:   util.ResolveImagePath(config.MustString("led.media.images.mute")),
	Play:   util.ResolveImagePath(config.MustString("led.media.images.play")),
	Pause:  util.ResolveImagePath(config.MustString("led.media.images.pause")),
	Stop:   util.ResolveImagePath(config.MustString("led.media.images.stop")),
	Next:   util.ResolveImagePath(config.MustString("led.media.images.next")),
}

func NewMediaPane(conn *ninja.Connection) *MediaPane {
	log := logger.GetLogger("MediaPane")

	pane := &MediaPane{
		log:         log,
		conn:        conn,
		gestureSync: &sync.Mutex{},

		volumeImage: util.LoadImage(mediaImages.Volume),
		muteImage:   util.LoadImage(mediaImages.Mute),
		playImage:   util.LoadImage(mediaImages.Play),
		pauseImage:  util.LoadImage(mediaImages.Pause),
		stopImage:   util.LoadImage(mediaImages.Stop),
		nextImage:   util.LoadImage(mediaImages.Next),

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

	listening := make(map[string]bool)

	getChannelServicesContinuous("mediaplayer", "media-control", nil, func(devices []*ninja.ServiceClient, err error) {

		if err != nil {
			log.Infof("Failed to update control devices: %s", err)
		} else {

			pane.controlDevices = devices

			log.Infof("Got %d media-control devices", len(devices))

			for _, device := range devices {
				if _, ok := listening[device.Topic]; !ok {
					listening[device.Topic] = true

					// New Device
					log.Infof("Got new control device: %s", device.Topic)

					device.OnEvent("playing", e("playing"))
					device.OnEvent("buffering", e("playing"))
					device.OnEvent("paused", e("paused"))
					device.OnEvent("stopped", e("stopped"))
				}
			}
		}

		if len(pane.controlDevices) == 0 {
			pane.volumeMode = true
		}

	})

	getChannelServicesContinuous("mediaplayer", "volume", nil, func(devices []*ninja.ServiceClient, err error) {
		if err != nil {
			log.Infof("Failed to update volume devices: %s", err)
		} else {

			log.Infof("Got %d volume devices", len(devices))

			for _, device := range devices {

				if _, ok := listening[device.Topic]; !ok {
					listening[device.Topic] = true
					// New device
					log.Infof("Got new volume device: %s", device.Topic)

					device.OnEvent("state", func(params *json.RawMessage, values map[string]string) bool {
						if time.Since(pane.lastVolumeTime) > time.Millisecond*500 {

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

		if len(pane.controlDevices) == 0 {
			pane.volumeMode = true
		}
	})

	pane.volumeModeReset = time.AfterFunc(0, func() {
		if len(pane.controlDevices) > 0 {
			pane.volumeMode = false
		}
	})

	pane.ignoreTapTimer = time.AfterFunc(0, func() {
		pane.ignoringTap = false
	})

	return pane
}

func (p *MediaPane) IsEnabled() bool {
	return len(p.volumeDevices) > 0 || len(p.controlDevices) > 0
}

func (p *MediaPane) Gesture(gesture *gestic.GestureMessage) {
	p.gestureSync.Lock()
	defer p.gestureSync.Unlock()

	if len(p.volumeDevices) > 0 && gesture.AirWheel.Counter > 0 && (p.lastAirWheel == nil || gesture.AirWheel.Counter != int(*p.lastAirWheel)) {

		//x, _ := json.Marshal(gesture)
		//p.log.Infof("wheel %s", x)

		p.volumeMode = true
		p.volumeModeReset.Reset(volumeModeReset)

		if time.Since(p.lastAirWheelTime) > airWheelReset {
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

			p.log.Debugf("Airwheel New: %d Offset: %d Last: %d", gesture.AirWheel.Counter, offset, *p.lastAirWheel)

			p.log.Debugf("Current volume %f", p.volume)

			p.log.Debugf("Volume offset %f", float64(offset)/255.0)

			var volume float64 = p.volume + (float64(offset)/255.0)*float64(2)

			volume = math.Max(volume, 0)
			volume = math.Min(volume, 1)

			p.log.Debugf("Adjusted volume %f:", volume)

			p.volume = volume

			if p.lastSentVolume != volume {
				if time.Since(p.lastVolumeTime) < volumeInterval {
					p.log.Debugf("Volume rate limited")
				} else {
					p.lastVolumeTime = time.Now()
					p.log.Debugf("Volume NOT rate limited")
					go p.SendVolume()
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
		p.ignoreTapTimer.Reset(mediaTapTimeout)

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
