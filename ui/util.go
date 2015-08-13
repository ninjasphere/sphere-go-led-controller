package ui

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
)

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

var sameRoomOnly = config.Bool(true, "homecloud.sameRoomOnly")

var conn *ninja.Connection
var tasks []*request
var thingModel *ninja.ServiceClient

var log = logger.GetLogger("ui")

type request struct {
	thingType string
	protocol  string
	filter    func(thing *model.Thing) bool
	cb        func([]*ninja.ServiceClient, error)
}

var foundLocation = make(chan bool)

var roomID *string

func runTasks() {

	// Find this sphere's location, if we care..
	if sameRoomOnly {
		for _, thing := range allThings {
			if thing.Type == "node" && thing.Device != nil && thing.Device.NaturalID == config.Serial() {
				if thing.Location != nil && (roomID == nil || *roomID != *thing.Location) {
					// Got it.
					log.Infof("Got this sphere's location: %s", thing.Location)
					roomID = thing.Location
					select {
					case foundLocation <- true:
					default:
					}

					break
				}
			}
		}
	}

	for _, task := range tasks {
		go func(t *request) {
			t.cb(getChannelServices(t.thingType, t.protocol, t.filter))
		}(task)
	}

}

var allThings []model.Thing

func fetchAll() error {

	var things []model.Thing

	err := thingModel.Call("fetchAll", []interface{}{}, &things, time.Second*20)
	//err = client.Call("fetch", "c7ac05e0-9999-4d93-bfe3-a0b4bb5e7e78", &thing)

	if err != nil {
		return fmt.Errorf("Failed to get things!: %s", err)
	}

	allThings = things
	runTasks()

	return nil
}

func StartSearchTasks(c *ninja.Connection) {
	startSearchTasks(c)
}

func startSearchTasks(c *ninja.Connection) {
	conn = c
	thingModel = conn.GetServiceClient("$home/services/ThingModel")

	dirty := false

	setDirty := func(params *json.RawMessage, topicKeys map[string]string) bool {
		log.Infof("Devices added/removed/updated. Marking dirty.")
		dirty = true
		return true
	}

	go func() {
		for {
			time.Sleep(time.Second * 30)
			setDirty(nil, nil)
		}
	}()

	go func() {
		for {
			err := fetchAll()
			if err == nil {
				break
			}

			log.Warningf("Failed to get fetch all things: %s", err)
			time.Sleep(time.Second * 3)
		}
	}()

	if sameRoomOnly {
		<-foundLocation
	}

	thingModel.OnEvent("created", setDirty)
	thingModel.OnEvent("updated", setDirty)
	thingModel.OnEvent("deleted", setDirty)

	go func() {
		time.Sleep(time.Second * 10)
		for {
			time.Sleep(time.Second * 5)
			if dirty {
				fetchAll()
				dirty = false
			}
		}
	}()
}

func GetChannelServicesContinuous(thingType string, protocol string, filter func(thing *model.Thing) bool, cb func([]*ninja.ServiceClient, error)) {
	getChannelServicesContinuous(thingType, protocol, filter, cb)
}

func getChannelServicesContinuous(thingType string, protocol string, filter func(thing *model.Thing) bool, cb func([]*ninja.ServiceClient, error)) {

	if filter == nil {
		filter = func(thing *model.Thing) bool {
			return roomID != nil && (thing.Location != nil && *thing.Location == *roomID)
		}
	}

	tasks = append(tasks, &request{thingType, protocol, filter, cb})

	cb(getChannelServices(thingType, protocol, filter))
}

func getChannelServices(thingType string, protocol string, filter func(thing *model.Thing) bool) ([]*ninja.ServiceClient, error) {

	//time.Sleep(time.Second * 3)

	var services []*ninja.ServiceClient

	for _, thing := range allThings {
		if thing.Type == thingType {

			spew.Dump("Found the right thing", thing, "looking for protocol", protocol)

			// Handle more than one channel with same protocol
			channel := getChannel(&thing, protocol)
			if channel != nil {
				if filter(&thing) {
					services = append(services, conn.GetServiceClientFromAnnouncement(channel.ServiceAnnouncement))
				}
			}
		}
	}
	return services, nil
}

func getChannel(thing *model.Thing, protocol string) *model.Channel {

	if thing.Device == nil || thing.Device.Channels == nil {
		return nil
	}

	for _, channel := range *thing.Device.Channels {
		if channel.Protocol == protocol {
			if thing.Device == nil {
				//spew.Dump("NO device on thing!", thing)
				return nil
			}

			return channel
		}
	}

	return nil
}
