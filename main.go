package main

import (
	"log"
	"os"
	"os/signal"

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

	conn, err := ninja.Connect(drivername)

	if err != nil {
		log.Fatalf("Failed to connect to mqtt: %s", err)
	}

	controller, err := NewLedController(conn)

	if err != nil {
		log.Fatalf("Failed to create led controller: %s", err)
	}

	enableControl := config.Bool(false, "enableControl")

	controller.start(enableControl)

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Println("Got signal:", x)

}
