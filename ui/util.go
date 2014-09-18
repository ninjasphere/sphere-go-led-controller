package ui

import (
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"math"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-ninja/rpc"
)

type Image struct {
	pos    int
	frames []*image.RGBA
}

func (i *Image) GetNextFrame() *image.RGBA {
	i.pos++
	if i.pos >= len(i.frames) {
		i.pos = 0
	}
	return i.frames[i.pos]
}

// GetPositionFrame returns the frame corresponding to the position given 0....1
func (i *Image) GetPositionFrame(position float64) *image.RGBA {
	frameNumber := math.Min(float64(len(i.frames)-1), math.Floor(position*float64(len(i.frames))))
	return i.frames[int(frameNumber)]
}

func (i *Image) GetNumFrames() int {
	return len(i.frames)
}

func (i *Image) GetFrame(frame int) *image.RGBA {
	return i.frames[frame]
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

/*
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
}*/

func getChannelIds(thingType string, protocol string, rpcClient *rpc.Client) ([]string, error) {

	//time.Sleep(time.Second * 3)

	var things []model.Thing

	call, err := rpcClient.Call("$home/services/ThingModel", "fetchByType", []interface{}{thingType}, &things)
	//err = client.Call("fetch", "c7ac05e0-9999-4d93-bfe3-a0b4bb5e7e78", &thing)

	if err != nil {
		log.Fatalf("Failed calling fetchByType method: %s", err)
	}

	<-call.Done
	spew.Dump(things)

	var topics []string

	for _, thing := range things {

		// Handle more than one channel with same protocol
		thingTopic := getChannelTopic(&thing, protocol)
		if thingTopic != "" {
			topics = append(topics, thingTopic)
		}
	}
	return topics, nil
}

func getChannelTopic(thing *model.Thing, protocol string) string {

	for _, channel := range *thing.Device.Channels {
		if channel.Protocol == protocol {
			return "$device/" + thing.Device.Guid + "/channel/" + channel.ID + "/" + protocol
		}
	}

	return ""
}
