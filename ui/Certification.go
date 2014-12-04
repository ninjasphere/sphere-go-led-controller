package ui

import (
	"encoding/json"
	"fmt"

	"image"
	"image/color"
	"regexp"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/bus"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
)

type CertPane struct {
	log       *logger.Logger
	waypoints int
	rssi      string
	tag       string
}

func NewCertPane(conn bus.Bus) *CertPane {

	log := logger.GetLogger("CertPane")

	pane := &CertPane{
		log: log,
	}

	_, err := conn.Subscribe("$location/waypoints", func(topic string, payload []byte) {
		var waypoints int
		err := json.Unmarshal(payload, &waypoints)
		if err != nil {
			log.Infof("Failed to parse incoming waypoints json %s, from %s", err, payload)
		} else {
			log.Infof("Number of waypoints: %d", waypoints)
		}
		pane.waypoints = waypoints
	})

	if err != nil {
		log.HandleError(err, "Could not start subscription to waypoint topic")
	}

	pane.StartRssi(conn)

	return pane
}

func (p *CertPane) Gesture(gesture *gestic.GestureMessage) {

}

var rssiRegex = regexp.MustCompile(`rssi":-(\d*)`)
var nameRegex = regexp.MustCompile(`name":"(.{3})"`)

func (p *CertPane) StartRssi(conn bus.Bus) {

	_, err := conn.Subscribe("$device/+/TEMPPATH/rssi", func(topic string, payload []byte) {
		nameFind := nameRegex.FindAllStringSubmatch(string(payload), -1)
		rssiFind := rssiRegex.FindAllStringSubmatch(string(payload), -1)

		if nameFind == nil {
			// Not a sticknfind
		} else {
			name := nameFind[0][1]
			rssi := rssiFind[0][1]
			spew.Dump("name", name, "rssi", rssi)

			p.tag = name
			p.rssi = rssi
		}

	})

	if err != nil {
		p.log.HandleError(err, "")
	}

}

func (p *CertPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	O4b03b.Font.DrawString(img, 0, 0, "wp", color.RGBA{255, 0, 0, 255})

	O4b03b.Font.DrawString(img, 12, 0, fmt.Sprintf("%d", p.waypoints), color.RGBA{255, 255, 255, 255})

	if p.tag != "" {
		O4b03b.Font.DrawString(img, 0, 8, "t", color.RGBA{255, 0, 0, 255})

		O4b03b.Font.DrawString(img, 4, 8, "-"+p.rssi, color.RGBA{255, 255, 255, 255})
	}

	return img, nil
}

func (p *CertPane) IsDirty() bool {
	return true
}
