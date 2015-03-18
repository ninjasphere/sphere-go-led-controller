package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
	"time"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

var lightTapInterval = config.MustDuration("led.light.tapInterval")
var colorInterval = config.MustDuration("led.light.colorInterval")

var colorAdjustSpeed = config.MustFloat("led.light.colorSpeed")
var brightnessAdjustSpeed = config.MustFloat("led.light.brightnessSpeed")

var brightnessMinimum = uint8(config.MustInt("led.light.brightnessMinimum"))

type LightPane struct {
	log      *logger.Logger
	conn     *ninja.Connection
	onEnable chan bool

	onOffDevices    *[]*ninja.ServiceClient
	airwheelDevices *[]*ninja.ServiceClient

	onOffState bool
	lastTap    time.Time

	colorMode bool

	airWheelState         float64
	lastSentAirWheelState float64
	airWheelThrottle      *throttle

	lastAirWheelTime time.Time
	lastAirWheel     *uint8

	onImage  util.Image
	offImage util.Image

	gestureSync *sync.Mutex
}

func NewLightPane(colorMode bool /*onOffDevices *[]*ninja.ServiceClient, airwheelDevices *[]*ninja.ServiceClient,*/, offImage string, onImage string, conn *ninja.Connection) *LightPane {

	name := "BrightnessPane"
	if colorMode {
		name = "ColorPane"
	}

	log := logger.GetLogger(name)

	log.Infof("Light rate: %s", colorInterval.String())

	pane := &LightPane{
		onEnable:         make(chan bool),
		colorMode:        colorMode,
		onImage:          util.LoadImage(onImage),
		offImage:         util.LoadImage(offImage),
		log:              log,
		conn:             conn,
		airWheelThrottle: &throttle{delay: colorInterval},
		lastTap:          time.Now(),
		gestureSync:      &sync.Mutex{},
	}

	if colorMode {
		pane.onOffState = true
	}

	listening := make(map[string]bool)

	if !colorMode {
		getChannelServicesContinuous("light", "on-off", /*func(thing *model.Thing) bool {
			isAccent := strings.Contains(strings.ToLower(thing.Name), "accent")
			return isAccent == demoAccentMode
			}*/nil, func(clients []*ninja.ServiceClient, err error) {
				if err != nil {
					log.Infof("Failed to update on-off devices: %s", err)
				} else {
					log.Infof("Got %d on/off devices", len(clients))
					pane.onOffDevices = &clients

					for _, device := range clients {
						if _, ok := listening[device.Topic]; !ok {
							listening[device.Topic] = true

							device.OnEvent("state", func(state *bool, topicKeys map[string]string) bool {
								log.Debugf("Got on-off state: %t", *state)
								// Ignore state updates if its within 500ms of a tap (which will update the display)
								if time.Since(pane.lastTap) > 500*time.Millisecond {
									pane.onOffState = *state
								}

								return true
							})
						}
					}
				}
			})
	}

	//if demoAccentMode {
	getChannelServicesContinuous("light", "core/batching", /*func(thing *model.Thing) bool {
		isAccent := strings.Contains(strings.ToLower(thing.Name), "accent")
		return isAccent == demoAccentMode
		}*/nil, func(clients []*ninja.ServiceClient, err error) {
			if err != nil {
				log.Infof("Failed to update batching devices: %s", err)
			} else {
				log.Infof("Fot %d batching devices", len(clients))
				pane.airwheelDevices = &clients
			}
		})
	//}

	if colorMode {
		getChannelServicesContinuous("light", "color", nil, func(clients []*ninja.ServiceClient, err error) {
			if err != nil {
				log.Infof("Failed to update color devices: %s", err)
			} else {
				for _, device := range clients {
					if _, ok := listening[device.Topic]; !ok {
						listening[device.Topic] = true

						device.OnEvent("state", func(state *channels.ColorState, topicKeys map[string]string) bool {
							log.Debugf("Got color state: %+v", *state)

							if state.Mode != "hue" {
								log.Warningf("Can't handle color mode: %s yet.", state.Mode)
								return true
							}

							// Ignore state updates if its within 2s of an airwheel (which will update the display)
							if time.Since(pane.lastAirWheelTime) > time.Second {
								pane.airWheelState = *state.Hue
							}

							return true
						})
					}
				}
			}
		})
	} else {
		getChannelServicesContinuous("light", "brightness", nil, func(clients []*ninja.ServiceClient, err error) {
			if err != nil {
				log.Infof("Failed to update brightness devices: %s", err)
			} else {
				for _, device := range clients {
					if _, ok := listening[device.Topic]; !ok {
						listening[device.Topic] = true

						device.OnEvent("state", func(state *float64, topicKeys map[string]string) bool {
							log.Infof("Got brightness state: %f", *state)

							// Ignore state updates if its within 2s of an airwheel (which will update the display)
							if time.Since(pane.lastAirWheelTime) > time.Second {
								pane.airWheelState = *state
							}

							return true
						})
					}
				}
			}
		})
	}

	return pane
}

func (p *LightPane) IsEnabled() bool {
	return (p.onOffDevices != nil && len(*p.onOffDevices) > 0) || (p.airwheelDevices != nil && len(*p.airwheelDevices) > 0)
}

func (p *LightPane) KeepAwake() bool {
	return false
}

func (p *LightPane) Gesture(gesture *gestic.GestureMessage) {

	p.gestureSync.Lock()
	defer p.gestureSync.Unlock()

	if !p.colorMode && gesture.Tap.Active() && time.Since(p.lastTap) > lightTapInterval {
		p.lastTap = time.Now()

		p.SetOnOffState(!p.onOffState)
	}

	if time.Since(gesture.Time) > time.Millisecond*100 {
		// Too old for wheeling, don't care
		return
	}

	//	x, _ := json.Marshal(gesture)
	//	p.log.Infof("Color gesture %s", x)
	/*
		col := p.airwheelState + colorRotateSpeed
		if col >= 1 {
			col = 0
		}
		p.airwheelState = col

		if !onOffRate.busy && colorRate.try() {

			p.SetColorState(col)
			p.log.Infof("Color wheel %f", col)

		} else {
			p.log.Infof("Ignoring Color wheel")
		}*/

	if gesture.AirWheel.Counter > 0 && (p.lastAirWheel == nil || gesture.AirWheel.Counter != int(*p.lastAirWheel)) {

		/*x, _ := json.Marshal(gesture)
		p.log.Infof("wheel %s", x)*/

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

			p.log.Debugf("Airwheel New: %d Offset: %d Last: %d", gesture.AirWheel.Counter, offset, *p.lastAirWheel)

			if p.colorMode {

				p.log.Debugf("Current color %f", p.airWheelState)

				p.log.Debugf("Color offset %f", float64(offset)/255.0)

				var color = p.airWheelState + (float64(offset)/255.0)*colorAdjustSpeed

				if color > 1 {
					color--
				}

				if color < 0 {
					color++
				}

				p.log.Debugf("Adjusted color %f:", color)

				p.airWheelState = color
			} else {
				// Brightness

				p.log.Debugf("Current brightness %f", p.airWheelState)

				p.log.Debugf("Brightness offset %f", float64(offset)/255.0)

				var brightness = p.airWheelState + (float64(offset)/255.0)*brightnessAdjustSpeed

				// Limit to between 0 and 1
				brightness = math.Min(brightness, 1)
				brightness = math.Max(brightness, 0)

				p.log.Debugf("Adjusted brightness %f:", brightness)

				p.airWheelState = brightness
			}

			if p.lastSentAirWheelState != p.airWheelState {
				if p.airWheelThrottle.try() {
					p.log.Debugf("Airwheel NOT rate limited")
					if p.colorMode {
						go p.SendColorToDevices()
					} else {
						go p.SendBrightnessToDevices()
					}
				} else {
					p.log.Debugf("Airwheel rate limited")
				}
			}
		}

		val := uint8(gesture.AirWheel.Counter)
		p.lastAirWheel = &val
	}

}

func (p *LightPane) SetOnOffState(state bool) {
	p.onOffState = state
	p.SendOnOffToDevices()
}

func (p *LightPane) SendOnOffToDevices() {

	if p.onOffState {
		p.log.Infof("Turning lights on")
	} else {
		p.log.Infof("Turning lights off")
	}

	for _, device := range *p.onOffDevices {

		if p.onOffState {
			device.Call("turnOn", nil, nil, 0)
		} else {
			device.Call("turnOff", nil, nil, 0)
		}

	}
}

func (p *LightPane) SendColorToDevices() {
	sat := 0.6

	for _, device := range *p.airwheelDevices {

		airwheelState := &channels.ColorState{
			Mode:       "hue",
			Hue:        &p.airWheelState,
			Saturation: &sat,
		}
		transition := 100

		device.Call("setBatch", &devices.LightDeviceState{
			Color:      airwheelState,
			Transition: &transition,
		}, nil, 0)

	}
}

func (p *LightPane) SendBrightnessToDevices() {

	for _, device := range *p.airwheelDevices {
		transition := 100

		device.Call("setBatch", &devices.LightDeviceState{
			Transition: &transition,
			Brightness: &p.airWheelState,
		}, nil, 0)

	}
}

func (p *LightPane) Render() (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))

	if p.colorMode {
		c := colorful.Hsv(p.airWheelState*360, 1, 1)
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.RGBA{uint8(c.R * 255), uint8(c.G * 255), uint8(c.B * 255), 255}}, image.ZP, draw.Src)
	} else {

		brightness := uint8(p.airWheelState * 255)
		if brightness < brightnessMinimum {
			brightness = brightnessMinimum
		}

		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.RGBA{brightness, brightness, brightness, 255}}, image.ZP, draw.Src)
	}

	var frame *image.RGBA
	if p.onOffState {
		frame = p.onImage.GetNextFrame()
	} else {
		frame = p.offImage.GetNextFrame()
	}

	draw.Draw(canvas, canvas.Bounds(), frame, image.ZP, draw.Over)

	return canvas, nil
}

func (p *LightPane) IsDirty() bool {
	return true
}

type throttle struct {
	delay time.Duration
	busy  bool
}

func (t *throttle) try() bool {
	if t.busy {
		return false
	}

	t.busy = true
	go func() {
		time.Sleep(t.delay)
		t.busy = false
	}()
	return true
}
