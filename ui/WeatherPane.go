package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"sync/atomic"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

const weatherDetailDuration = 5 * time.Second

// timezone is referenced by ClockPane. Originally populated by the old
// site-model lookup that this pane no longer does — we leave it at nil so
// ClockPane falls back to the system local time, which is correct on both
// spheres' OS clocks.
var timezone *time.Location

// WeatherState mirrors the payload of "state" events on the "weather" channel
// emitted by ha-bridge (current condition + today's hi/lo + day-or-night).
type WeatherState struct {
	Condition string  `json:"condition"`
	TempC     float64 `json:"temp_c"`
	HighC     float64 `json:"temp_high_c"`
	LowC      float64 `json:"temp_low_c"`
	IsDay     bool    `json:"is_day"`
}

// HA condition string → OWM-style icon code (sans d/n suffix). The icon assets
// in `images/weather/{01..50}{d|n}.png` are kept verbatim from upstream — we
// only changed the data source.
var weatherIconMap = map[string]string{
	"sunny":           "01",
	"clear-night":     "01",
	"partlycloudy":    "02",
	"cloudy":          "04",
	"rainy":           "10",
	"pouring":         "09",
	"lightning":       "11",
	"lightning-rainy": "11",
	"snowy":           "13",
	"snowy-rainy":     "13",
	"hail":            "13",
	"fog":             "50",
	"windy":           "04",
	"windy-variant":   "04",
	"exceptional":     "11",
}

type WeatherPane struct {
	log     *logger.Logger
	devices []*ninja.ServiceClient

	state WeatherState
	have  bool

	image    util.Image
	imageKey string // current "01d" / "10n" etc — used to skip reloads when state spam doesn't change the icon

	detailUntil      time.Time
	ignoringGestures atomic.Bool
}

func NewWeatherPane(conn *ninja.Connection) *WeatherPane {
	log := logger.GetLogger("WeatherPane")
	pane := &WeatherPane{
		log:   log,
		image: util.LoadImage(util.ResolveImagePath("weather/loading.gif")),
	}

	listening := map[string]bool{}
	getChannelServicesContinuous("weather", "weather", nil, func(clients []*ninja.ServiceClient, err error) {
		if err != nil {
			log.Infof("Failed to update weather devices: %s", err)
			return
		}
		log.Infof("WeatherPane got %d weather devices", len(clients))
		pane.devices = clients
		for _, device := range clients {
			if listening[device.Topic] {
				continue
			}
			listening[device.Topic] = true
			device.OnEvent("state", func(params *json.RawMessage, _ map[string]string) bool {
				var st WeatherState
				if err := json.Unmarshal(*params, &st); err != nil {
					log.Warningf("weather state unmarshal: %s", err)
					return true
				}
				pane.state = st
				pane.have = true
				pane.refreshIcon()
				return true
			})
		}
	})

	return pane
}

// refreshIcon picks the right asset for current condition + day/night and
// reloads only when the choice actually changes.
func (p *WeatherPane) refreshIcon() {
	code, ok := weatherIconMap[p.state.Condition]
	if !ok {
		code = "01" // unknown condition → fall back to clear
	}
	suffix := "d"
	if p.state.Condition == "clear-night" || !p.state.IsDay {
		suffix = "n"
	}
	key := code + suffix
	if key == p.imageKey {
		return
	}
	p.imageKey = key
	p.image = util.LoadImage(util.ResolveImagePath("weather/" + key + ".png"))
}

func (p *WeatherPane) IsEnabled() bool { return len(p.devices) > 0 && p.have }
func (p *WeatherPane) KeepAwake() bool { return false }
func (p *WeatherPane) IsDirty() bool   { return true }

func (p *WeatherPane) Gesture(gesture *gestic.GestureMessage) {
	if p.ignoringGestures.Load() {
		return
	}
	if gesture.Tap.Active() {
		p.log.Infof("Tap! showing weather detail")
		p.detailUntil = time.Now().Add(weatherDetailDuration)
		p.ignoringGestures.Store(true)
		go func() {
			time.Sleep(300 * time.Millisecond)
			p.ignoringGestures.Store(false)
		}()
	}
}

func (p *WeatherPane) Render() (*image.RGBA, error) {
	if !p.have {
		return p.image.GetNextFrame(), nil
	}
	if time.Now().Before(p.detailUntil) {
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		p.renderDetail(img)
		return img, nil
	}
	return p.image.GetNextFrame(), nil
}

func (p *WeatherPane) renderDetail(img *image.RGBA) {
	drawText := func(text string, col color.RGBA, top int) {
		// Measure on a throwaway canvas so we can right-align.
		width := O4b03b.Font.DrawString(image.NewRGBA(image.Rect(0, 0, 16, 16)), 0, 0, text, color.Black)
		start := 16 - width - 2
		if start < 0 {
			start = 0
		}
		O4b03b.Font.DrawString(img, start, top, text, col)
	}
	drawText(fmt.Sprintf("%dC", int(p.state.HighC+0.5)), color.RGBA{253, 151, 32, 255}, 1)
	drawText(fmt.Sprintf("%dC", int(p.state.LowC+0.5)), color.RGBA{69, 175, 249, 255}, 8)
}
