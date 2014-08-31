package ui

import (
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"net/rpc"
	"os"
	"strings"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/rpc2"
)

type Image struct {
	frame  int
	frames []*image.RGBA
}

func (i *Image) GetFrame() *image.RGBA {
	i.frame++
	if i.frame >= len(i.frames) {
		i.frame = 0
	}
	return i.frames[i.frame]
}

func loadImage(src string) *Image {
	srcLower := strings.ToLower(src)

	if strings.Contains(srcLower, ".gif") {
		return loadGif(src)
	} else if strings.Contains(srcLower, ".png") {
		return loadPng(src)
	} else {
		log.Fatalf("Unknown image format: %s", src)
	}
	return nil
}

func loadPng(src string) *Image {
	file, err := os.Open(src)

	if err != nil {
		log.Fatalf("Could not open png '%s' : %s", src, err)
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Fatalf("PNG decoding failed on image '%s' : %s", src, err)
	}

	return &Image{
		frames: []*image.RGBA{toRGBA(img)},
	}
}

func loadGif(src string) *Image {
	file, err := os.Open(src)

	if err != nil {
		log.Fatalf("Could not open gif '%s' : %s", src, err)
	}

	img, err := gif.DecodeAll(file)
	if err != nil {
		log.Fatalf("Gif decoding failed on image '%s' : %s", src, err)
	}

	var frames = []*image.RGBA{}

	for _, frame := range img.Image {
		frames = append(frames, toRGBA(frame))
	}

	return &Image{
		frames: frames,
	}
}

func toRGBA(in image.Image) *image.RGBA {
	bounds := in.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), in, bounds.Min, draw.Over)
	return out
}

type Thing struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Device Device
}

type Device struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	IDType   string `json:"idType"`
	Guid     string `json:"guid"`
	Channels []Channel
}

type Channel struct {
	Protocol string `json:"protocol"`
	Name     string `json:"channel"`
	ID       string `json:"id"`
}

func getChannelClients(thingType string, protocol string, mqtt *mqtt.MqttClient) ([]*rpc.Client, error) {
	// You need to export the mqtt connection here if you want to test it.
	client, err := rpc2.GetClient("$home/services/ThingModel", mqtt)

	if err != nil {
		log.Fatalf("Failed getting rpc2 client %s", err)
	}

	//time.Sleep(time.Second * 3)

	var things []Thing

	err = client.Call("fetchByType", thingType, &things)
	//err = client.Call("fetch", "c7ac05e0-9999-4d93-bfe3-a0b4bb5e7e78", &thing)

	if err != nil {
		log.Fatalf("Failed calling fetch method: %s", err)
	}

	log.Printf("Done")
	spew.Dump(things)

	var clients []*rpc.Client

	for _, thing := range things {
		client, err := getChannelClient(&thing, protocol, mqtt)
		if err != nil {
			log.Fatalf("Failed getting %s client for thing %s: %s", protocol, thing.ID, err)
		}

		if client != nil {
			log.Printf("Found %s on thing %s", protocol, thing.ID)
			clients = append(clients, client)
			//	_ = onOffClient.Go("turnOn", nil, nil, nil)

		}
	}
	return clients, nil
}

func getChannelClient(thing *Thing, protocol string, mqtt *mqtt.MqttClient) (*rpc.Client, error) {

	for _, channel := range thing.Device.Channels {
		if channel.Protocol == protocol {
			topic := "$device/" + thing.Device.Guid + "/channel/" + channel.ID + "/" + protocol
			return rpc2.GetClient(topic, mqtt)
		}
	}

	return nil, nil
}
