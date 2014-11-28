package ui

import (
	"encoding/json"
	"fmt"

	"image"
	"image/color"
	"regexp"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-gestic"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/org.eclipse.paho.mqtt.golang"
	"github.com/ninjasphere/sphere-go-led-controller/fonts/O4b03b"
)

type CertPane struct {
	log       *logger.Logger
	waypoints int
	rssi      string
	tag       string
}

func NewCertPane(conn *mqtt.MqttClient) *CertPane {

	log := logger.GetLogger("CertPane")

	pane := &CertPane{
		log: log,
	}

	filter, err := mqtt.NewTopicFilter("$location/waypoints", 0)
	if err != nil {
		log.Fatalf("Boom, no good", err)
	}

	receipt, err := conn.StartSubscription(func(client *mqtt.MqttClient, message mqtt.Message) {
		var waypoints int
		err := json.Unmarshal(message.Payload(), &waypoints)
		if err != nil {
			log.Infof("Failed to parse incoming waypoints json %s, from %s", err, message.Payload())
		} else {
			log.Infof("Number of waypoints: %d", waypoints)
		}
		pane.waypoints = waypoints
	}, filter)

	if err != nil {
		log.HandleError(err, "Could not start subscription to waypoint topic")
	}

	<-receipt

	pane.StartRssi(conn)

	return pane
}

func (p *CertPane) Gesture(gesture *gestic.GestureData) {

}

var rssiRegex = regexp.MustCompile(`rssi":-(\d*)`)
var nameRegex = regexp.MustCompile(`name":"(.{3})"`)

func (p *CertPane) StartRssi(conn *mqtt.MqttClient) {

	filter, err := mqtt.NewTopicFilter("$device/+/TEMPPATH/rssi", 0)
	if err != nil {
		p.log.HandleError(err, "Could not subscribe to $device/+/TEMPPATH/rssi ")
	}

	receipt, err := conn.StartSubscription(func(client *mqtt.MqttClient, message mqtt.Message) {
		nameFind := nameRegex.FindAllStringSubmatch(string(message.Payload()), -1)
		rssiFind := rssiRegex.FindAllStringSubmatch(string(message.Payload()), -1)

		if nameFind == nil {
			// Not a sticknfind
		} else {
			name := nameFind[0][1]
			rssi := rssiFind[0][1]
			spew.Dump("name", name, "rssi", rssi)

			p.tag = name
			p.rssi = rssi
		}

	}, filter)

	if err != nil {
		p.log.HandleError(err, "")
	}

	<-receipt

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
