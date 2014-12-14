package ui

import (
	"encoding/json"
	"log"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
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

var conn *ninja.Connection
var tasks []*request
var thingModel *ninja.ServiceClient

type request struct {
	thingType string
	protocol  string
	filter    func(thing *model.Thing) bool
	cb        func([]*ninja.ServiceClient, error)
}

var dirty = false

func runTasks() {

	for _, task := range tasks {
		go func(t *request) {
			t.cb(getChannelServices(t.thingType, t.protocol, t.filter))
		}(task)
	}

}

var roomID *string

func startSearchTasks(c *ninja.Connection) {
	conn = c
	thingModel = conn.GetServiceClient("$home/services/ThingModel")

	setDirty := func(params *json.RawMessage, topicKeys map[string]string) bool {
		log.Printf("Devices added/removed/updated. Marking dirty.")
		dirty = true
		return true
	}

	foundLocation := make(chan bool)

	go func() {

		for {
			// Find this sphere's thing, so we know what room it's in
			var nodes []model.Thing

			err := thingModel.Call("fetchByType", []interface{}{"node"}, &nodes, time.Second*10)

			gotIt := false

			if err != nil {
				log.Printf("Failed finding this sphere %s ", err)
			} else {

				for _, thing := range nodes {
					if thing.Device != nil && thing.Device.NaturalID == config.Serial() {

						if thing.Location != nil && (roomID == nil || *roomID != *thing.Location) {
							// Got it.
							log.Printf("Got this sphere's location: %s", thing.Location)
							roomID = thing.Location
							dirty = true
							gotIt = true
							select {
							case foundLocation <- true:
							default:
							}
						}
					}
				}

				if gotIt {
					time.Sleep(time.Second * 20)
				} else {
					log.Printf("Didn't find the sphere's location")
					time.Sleep(time.Second * 5)
				}
			}

		}
	}()

	<-foundLocation

	thingModel.OnEvent("created", setDirty)
	thingModel.OnEvent("updated", setDirty)
	thingModel.OnEvent("deleted", setDirty)

	go func() {
		time.Sleep(time.Second * 10)
		for {
			time.Sleep(time.Second * 5)
			if dirty {
				runTasks()
				dirty = false
			}
		}
	}()
}

func getChannelServicesContinuous(thingType string, protocol string, filter func(thing *model.Thing) bool, cb func([]*ninja.ServiceClient, error)) {

	tasks = append(tasks, &request{thingType, protocol, filter, cb})

	if filter == nil {
		filter = func(thing *model.Thing) bool {
			return roomID == nil || (thing.Location != nil && *thing.Location == *roomID)
		}
	}

	cb(getChannelServices(thingType, protocol, filter))
}

func getChannelServices(thingType string, protocol string, filter func(thing *model.Thing) bool) ([]*ninja.ServiceClient, error) {

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
			if filter(&thing) {
				services = append(services, conn.GetServiceClient(channelTopic))
			}
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
			}

			return "$device/" + thing.Device.ID + "/channel/" + channel.ID
		}
	}

	return ""
}
