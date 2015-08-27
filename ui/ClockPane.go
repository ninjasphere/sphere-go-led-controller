package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/clock"
)

var enableClockPane = config.Bool(true, "led.clock.enabled")
var enableAlarm = config.Bool(true, "led.clock.alarmEnabled")
var alarmFlashTimes = config.Int(2, "led.clock.alarmFlashTimes") * 2
var alarmFlashInterval = config.MustDuration("led.clock.alarmFlashInterval")

type ClockPane struct {
	alarm       *time.Time
	timer       *time.Timer
	tapThrottle *throttle
	lights      []*ninja.ServiceClient
	tickTock    bool
}

func NewClockPane() *ClockPane {
	var pane *ClockPane
	pane = &ClockPane{
		timer: time.AfterFunc(time.Minute, func() {
			pane.alarm = nil
			pane.DoAlarm()
		}),
		tapThrottle: &throttle{delay: time.Millisecond * 500},
	}
	pane.timer.Stop()

	if enableAlarm {
		enableAlarm = false

		getChannelServicesContinuous("light", "on-off", nil, func(devices []*ninja.ServiceClient, err error) {
			if err != nil {
				log.Infof("Failed to update on-off devices: %s", err)
				enableAlarm = false
			} else {
				log.Infof("ClockPane got %d lights to alarm", len(devices))
				if len(devices) > 0 {
					pane.lights = devices
					enableAlarm = true
				} else {
					enableAlarm = false
				}
			}

			if !enableAlarm {
				pane.alarm = nil
				pane.timer.Stop()
			}
		})
	}

	go func() {
		tick := time.Tick(time.Second)
		for _ = range tick {
			pane.tickTock = !pane.tickTock
		}
	}()

	return pane
}

func (p *ClockPane) IsEnabled() bool {
	return enableClockPane
}

func (p *ClockPane) KeepAwake() bool {
	return false
}

func (p *ClockPane) Gesture(gesture *gestic.GestureMessage) {
	if !enableAlarm {
		return
	}

	if gesture.Tap.Active() && p.tapThrottle.try() {
		if p.alarm == nil {
			x := time.Now().Add(time.Minute)
			p.alarm = &x
		} else {
			x := p.alarm.Add(time.Minute)
			p.alarm = &x
		}

		p.timer.Reset(p.alarm.Sub(time.Now()))
	}

	if gesture.DoubleTap.Active() {
		p.alarm = nil
		p.timer.Stop()
	}
}

func (p *ClockPane) DoAlarm() {

	log.Infof("Alarm Activated! Flashing %d lights %d times", len(p.lights), alarmFlashTimes)

	for _, device := range p.lights {
		go func(d *ninja.ServiceClient) {
			for i := 0; i < alarmFlashTimes; i++ {
				d.Call("toggle", nil, nil, 0)
				time.Sleep(time.Second * 2)
			}
		}(device)
	}

}

func (p *ClockPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	var text string
	if p.alarm != nil {
		duration := p.alarm.Sub(time.Now())
		text = fmt.Sprintf("%d:%02d", int(duration.Minutes()), int(duration.Seconds())-(int(duration.Minutes())*60))
	} else {

		t := time.Now()

		if timezone != nil {
			t = t.In(timezone)
		}

		text = t.Format("3:04")
		if text[0] == '0' { // 0 is too wide
			text = text[1:]
		}
		if p.tickTock {
			text = strings.Replace(text, ":", ";", 1)
		}
	}

	width := clock.Font.DrawString(img, 0, 0, text, color.Black)
	start := 16 - width

	//log.Infof("text width:%d start:%d text:%s", width, start, text)

	clock.Font.DrawString(img, start, 5, text, color.White)

	return img, nil
}

func (p *ClockPane) IsDirty() bool {
	return true
}
