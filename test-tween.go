// +build ignore

package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/ninjasphere/sphere-go-led-controller"
)

func main() {

	tween := &led.Tween{
		From:     0,
		To:       100,
		Duration: time.Second * 5,
		Start:    time.Now(),
	}

	go func() {
		for {
			val, done := tween.Update()

			log.Printf("Value: %d", val)

			if done {
				break
			}

		}
	}()

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Println("Got signal:", x)

	/*	buf := make([]byte, 128)
		n, err = s.Read(buf)
		if err != nil {
		  log.Fatal(err)
		}
		log.Print("%q", buf[:n])*/
}
