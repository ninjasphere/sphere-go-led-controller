// +build ignore

package main

import (
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/sphere-go-led-controller"
	"github.com/tarm/goserial"
)

var CMD_WRITE_BUFFER byte = 1
var CMD_SWAP_BUFFERS byte = 2

func write(image *image.RGBA, s io.ReadWriteCloser) {

	//spew.Dump("writing image", image)

	var frame [768]byte

	for i := 0; i < len(image.Pix); i = i + 4 {
		//log.Println(i)
		frame[i/4*3] = image.Pix[i]
		frame[(i/4*3)+1] = image.Pix[i+1]
		frame[(i/4*3)+2] = image.Pix[i+2]
	}

	_, err := s.Write([]byte{CMD_WRITE_BUFFER})
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	_, err = s.Write(frame[:])
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	_, err = s.Write([]byte{CMD_SWAP_BUFFERS})
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	//log.Println("Wrote frame", n)
}

func main() {

	layout := led.NewPaneLayout()

	fanPane := led.NewOnOffPane("images/fan-off.png", "images/fan-on.gif", func(state bool) {
		log.Printf("Fan state: %t", state)
	})
	layout.AddPane(fanPane)

	heaterPane := led.NewOnOffPane("images/heater-off.png", "images/heater-on.gif", func(state bool) {
		log.Printf("Heater state: %t", state)
	})
	layout.AddPane(heaterPane)

	// Toggle fan and heater panes every second
	go func() {
		state := false
		for {
			time.Sleep(time.Second * 1)
			state = !state
			fanPane.SetState(state)
			heaterPane.SetState(state)
		}
	}()

	layout.AddPane(led.NewColorPane(color.RGBA{0, 0, 255, 255}))
	layout.AddPane(led.NewColorPane(color.RGBA{255, 0, 0, 255}))
	layout.AddPane(led.NewColorPane(color.RGBA{0, 255, 0, 255}))
	layout.AddPane(led.NewColorPane(color.RGBA{255, 0, 255, 255}))

	log.Println("starting")
	c := &serial.Config{Name: "/dev/tty.ledmatrix", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Printf("No led matrix? Ignoring... %s", err)
	}

	go func() {
		for {
			if s == nil {
				time.Sleep(time.Millisecond * 80)
			}
			//time.Sleep(time.Second / 10)
			image, err := layout.Render()
			if err != nil {
				log.Fatal(err)
			}
			if s != nil {
				write(image, s)
			} else {
				spew.Dump(image)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 4)
			layout.PanLeft()
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
