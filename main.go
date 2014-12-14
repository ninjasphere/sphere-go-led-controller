package main

import (
	"os"
	"os/signal"

	"github.com/ninjasphere/go-ninja/logger"

	//"net/http"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
)

//import _ "net/http/pprof"

const drivername = "sphere-led-controller"

func main() {

	/*
		go func() {
			log.Printf("Starting pprof server")
			log.Println(http.ListenAndServe(":6060", nil))
		}()
		//*/

	log := logger.GetLogger("LED-controller")

	conn, err := ninja.Connect(drivername)

	if err != nil {
		log.FatalErrorf(err, "Failed to connect to mqtt")
	}

	controller, err := NewLedController(conn)

	if err != nil {
		log.FatalErrorf(err, "Failed to create led controller")
	}

	enableControl := config.Bool(false, "enableControl")

	controller.start(enableControl)

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Infof("Got signal:", x)

}
