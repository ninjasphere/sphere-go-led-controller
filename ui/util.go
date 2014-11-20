package ui

import (
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
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
func (i *Image) GetPositionFrame(position float64, blend bool) *image.RGBA {

	relativePosition := position * float64(len(i.frames)-1)

	previousFrame := int(math.Floor(relativePosition))
	nextFrame := int(math.Ceil(relativePosition))

	framePosition := math.Mod(relativePosition, 1)

	//log.Debugf("GetPositionFrame. Frames:%d Position:%f RelativePosition:%f FramePosition:%f PreviousFrame:%d NextFrame:%d", len(i.frames), position, relativePosition, framePosition, previousFrame, nextFrame)
	if !blend || previousFrame == nextFrame {
		// We can just send back a single frame
		return i.frames[previousFrame]
	}

	maskNext := image.NewUniform(color.Alpha{uint8(255 * framePosition)})

	frame := image.NewRGBA(image.Rect(0, 0, 16, 16))

	draw.Draw(frame, frame.Bounds(), i.frames[previousFrame], image.Point{0, 0}, draw.Src)
	draw.DrawMask(frame, frame.Bounds(), i.frames[nextFrame], image.Point{0, 0}, maskNext, image.Point{0, 0}, draw.Over)

	return frame
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
		log.Printf("Unknown image format: %s", src)
	}
	return nil
}

func loadPng(src string) *Image {
	file, err := os.Open(src)

	if err != nil {
		log.Printf("Could not open png '%s' : %s", src, err)
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Printf("PNG decoding failed on image '%s' : %s", src, err)
	}

	return &Image{
		frames: []*image.RGBA{toRGBA(img)},
	}
}

func loadGif(src string) *Image {
	file, err := os.Open(src)

	if err != nil {
		log.Printf("Could not open gif '%s' : %s", src, err)
	}

	img, err := gif.DecodeAll(file)
	if err != nil {
		log.Printf("Gif decoding failed on image '%s' : %s", src, err)
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

var conn *ninja.Connection
var tasks []*request
var thingModel *ninja.ServiceClient

type request struct {
	thingType string
	protocol  string
	cb        func([]*ninja.ServiceClient, error)
}

var dirty = false

func runTasks() {

	for _, task := range tasks {
		go func(t *request) {
			t.cb(getChannelServices(t.thingType, t.protocol))
		}(task)
	}

}

func startSearchTasks(c *ninja.Connection) {
	conn = c
	thingModel = conn.GetServiceClient("$home/services/ThingModel")

	setDirty := func(params *json.RawMessage, topicKeys map[string]string) bool {
		log.Printf("Devices added/removed/updated. Marking dirty.")
		dirty = true
		return true
	}

	thingModel.OnEvent("created", setDirty)
	thingModel.OnEvent("updated", setDirty)
	thingModel.OnEvent("deleted", setDirty)

	go func() {
		time.Sleep(time.Second * 20)
		for {
			time.Sleep(time.Second * 5)
			if dirty {
				runTasks()
				dirty = false
			}
		}
	}()
}

func getChannelServicesContinuous(thingType string, protocol string, cb func([]*ninja.ServiceClient, error)) {

	tasks = append(tasks, &request{thingType, protocol, cb})

	cb(getChannelServices(thingType, protocol))
}

func getChannelServices(thingType string, protocol string) ([]*ninja.ServiceClient, error) {

	//time.Sleep(time.Second * 3)

	var things []model.Thing

	err := thingModel.Call("fetchByType", []interface{}{thingType}, &things, time.Second*10)
	//err = client.Call("fetch", "c7ac05e0-9999-4d93-bfe3-a0b4bb5e7e78", &thing)

	if err != nil {
		log.Printf("Failed calling fetchByType method %s ", err)
	}

	//spew.Dump(things)

	var services []*ninja.ServiceClient

	for _, thing := range things {

		// Handle more than one channel with same protocol
		channelTopic := getChannelTopic(&thing, protocol)
		if channelTopic != "" {
			services = append(services, conn.GetServiceClient(channelTopic))
		}
	}
	return services, nil
}

/*func listenToEvents(topic string, conn *mqtt.MqttClient) {

	filter, err := mqtt.NewTopicFilter(topic+"/event/+", 0)
	if err != nil {
		log.Fatalf("Boom, no good", err)
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
		log.Fatalf("Boom, no good", err)
	}

	<-receipt

}*/

func getChannelTopic(thing *model.Thing, protocol string) string {

	if thing.Device == nil || thing.Device.Channels == nil {
		return ""
	}

	for _, channel := range *thing.Device.Channels {
		if channel.Protocol == protocol {
			if thing.Device == nil {
				//spew.Dump("NO device on thing!", thing)
				return ""
			} else {
				return "$device/" + thing.Device.ID + "/channel/" + channel.ID
			}
		}
	}

	return ""
}
