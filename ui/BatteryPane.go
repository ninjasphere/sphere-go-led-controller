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
)

const batteryDetailDuration = 5 * time.Second

// BatteryState is the payload of a "state" event on the "battery" channel.
type BatteryState struct {
	Level    int  `json:"level"`    // 0..100
	Charging bool `json:"charging"`
}

type BatteryPane struct {
	log     *logger.Logger
	devices []*ninja.ServiceClient

	state BatteryState
	have  bool

	detailUntil      time.Time
	pulseFrame       int
	ignoringGestures atomic.Bool
}

func NewBatteryPane(conn *ninja.Connection) *BatteryPane {
	log := logger.GetLogger("BatteryPane")
	pane := &BatteryPane{log: log, devices: []*ninja.ServiceClient{}}

	listening := map[string]bool{}

	getChannelServicesContinuous("battery", "battery", nil, func(clients []*ninja.ServiceClient, err error) {
		if err != nil {
			log.Infof("Failed to update devices: %s", err)
			return
		}
		log.Infof("BatteryPane got %d battery devices", len(clients))
		pane.devices = clients
		for _, device := range clients {
			if listening[device.Topic] {
				continue
			}
			listening[device.Topic] = true
			device.OnEvent("state", func(params *json.RawMessage, _ map[string]string) bool {
				var st BatteryState
				if err := json.Unmarshal(*params, &st); err != nil {
					log.Warningf("battery state unmarshal: %s", err)
					return true
				}
				pane.state = st
				pane.have = true
				return true
			})
		}
	})

	return pane
}

func (p *BatteryPane) IsEnabled() bool { return len(p.devices) > 0 }
func (p *BatteryPane) KeepAwake() bool { return false }
func (p *BatteryPane) IsDirty() bool   { return true }

func (p *BatteryPane) Gesture(gesture *gestic.GestureMessage) {
	if p.ignoringGestures.Load() {
		return
	}
	if gesture.Tap.Active() {
		p.log.Infof("Tap! showing detail")
		p.detailUntil = time.Now().Add(batteryDetailDuration)
		p.ignoringGestures.Store(true)
		go func() {
			time.Sleep(300 * time.Millisecond)
			p.ignoringGestures.Store(false)
		}()
	}
}

func batteryColor(level int) color.RGBA {
	switch {
	case level >= 50:
		return color.RGBA{0, 200, 60, 255} // green
	case level >= 20:
		return color.RGBA{255, 180, 0, 255} // amber
	default:
		return color.RGBA{255, 40, 0, 255} // red
	}
}

func (p *BatteryPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	if !p.have {
		drawBatteryBody(img, color.RGBA{60, 60, 60, 255})
		return img, nil
	}

	// Detail view (tap-triggered) auto-reverts after batteryDetailDuration.
	if time.Now().Before(p.detailUntil) {
		p.renderDetail(img)
		return img, nil
	}
	p.renderBars(img)
	return img, nil
}

func (p *BatteryPane) renderBars(img *image.RGBA) {
	drawBatteryBody(img, color.RGBA{180, 180, 180, 255})

	filled := (p.state.Level + 19) / 20 // ceil(level/20)
	if filled > 5 {
		filled = 5
	}
	c := batteryColor(p.state.Level)

	// 5 bars, bottom-up. Bar i (0=bottom) at row 13 - 2i, cols 6..9.
	for i := 0; i < 5; i++ {
		row := 13 - 2*i
		if i < filled {
			for col := 6; col <= 9; col++ {
				img.SetRGBA(col, row, c)
			}
		}
	}

	// Charging: pulse the next-empty bar in white (or pulse cap when full).
	if p.state.Charging {
		p.pulseFrame++
		on := (p.pulseFrame/15)%2 == 0
		if on {
			pulse := color.RGBA{255, 255, 255, 255}
			if filled < 5 {
				row := 13 - 2*filled
				for col := 6; col <= 9; col++ {
					img.SetRGBA(col, row, pulse)
				}
			} else {
				// Full + still charging — pulse the cap.
				for row := 1; row <= 2; row++ {
					for col := 6; col <= 9; col++ {
						img.SetRGBA(col, row, pulse)
					}
				}
			}
		}
	}
}

func (p *BatteryPane) renderDetail(img *image.RGBA) {
	c := batteryColor(p.state.Level)
	text := fmt.Sprintf("%d", p.state.Level)

	// Measure with a throwaway draw in transparent black (off-screen y).
	width := O4b03b.Font.DrawString(image.NewRGBA(image.Rect(0, 0, 16, 16)), 0, 0, text, color.Black)
	start := (16 - width) / 2
	if start < 0 {
		start = 0
	}
	// O4b03b sits ~6px tall; centre vertically biased toward bottom so arrow has room above.
	O4b03b.Font.DrawString(img, start, 8, text, c)

	// Arrow above the number — up if charging, down otherwise.
	white := color.RGBA{255, 255, 255, 255}
	if p.state.Charging {
		drawArrow(img, white, true)
	} else {
		drawArrow(img, white, false)
	}
}

// drawBatteryBody renders the vertical battery outline (cap on top).
// Body: cols 5..10, rows 3..14 (6w x 12h). Cap: cols 6..9, rows 1..2.
func drawBatteryBody(img *image.RGBA, c color.RGBA) {
	// Cap
	for row := 1; row <= 2; row++ {
		for col := 6; col <= 9; col++ {
			img.SetRGBA(col, row, c)
		}
	}
	// Body top + bottom edges
	for col := 5; col <= 10; col++ {
		img.SetRGBA(col, 3, c)
		img.SetRGBA(col, 14, c)
	}
	// Body sides
	for row := 4; row <= 13; row++ {
		img.SetRGBA(5, row, c)
		img.SetRGBA(10, row, c)
	}
}

// drawArrow draws a 6-wide chevron at the top of the panel (rows 1..3).
// up=true: pointing up (charging).
func drawArrow(img *image.RGBA, c color.RGBA, up bool) {
	if up {
		// row 1: cols 7,8
		for _, col := range []int{7, 8} {
			img.SetRGBA(col, 1, c)
		}
		// row 2: cols 6..9
		for col := 6; col <= 9; col++ {
			img.SetRGBA(col, 2, c)
		}
		// row 3: cols 5..10
		for col := 5; col <= 10; col++ {
			img.SetRGBA(col, 3, c)
		}
	} else {
		// row 1: cols 5..10
		for col := 5; col <= 10; col++ {
			img.SetRGBA(col, 1, c)
		}
		// row 2: cols 6..9
		for col := 6; col <= 9; col++ {
			img.SetRGBA(col, 2, c)
		}
		// row 3: cols 7,8
		for _, col := range []int{7, 8} {
			img.SetRGBA(col, 3, c)
		}
	}
}
