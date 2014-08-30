// +build ignore

package main

import (
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"os/signal"

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

	rows := split(frame[:], 16*3)

	var orderedRows [][]byte
	for i := 0; i < 8; i++ {
		orderedRows = append(orderedRows, rows[i+8])
		orderedRows = append(orderedRows, rows[i])
	}

	var finalFrame []byte

	for _, line := range orderedRows {
		for i, j := 0, len(line)-1; i < j; i, j = i+1, j-1 {
			line[i], line[j] = line[j], line[i]
		}

		finalFrame = append(finalFrame, line...)
	}

	_, err := s.Write([]byte{CMD_WRITE_BUFFER})
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	_, err = s.Write(finalFrame[:])
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	_, err = s.Write([]byte{CMD_SWAP_BUFFERS})
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	//log.Println("Wrote frame", n)
}

func split(a []byte, size int) [][]byte {
	var out [][]byte
	var i = 0
	for i < len(a) {
		out = append(out, a[i:i+size])
		i += size
	}

	return out
}

func main() {

	layout, wake := led.NewPaneLayout()

	fanPane := led.NewOnOffPane("images/fan-off.png", "images/fan-on.gif", func(state bool) {
		log.Printf("Fan state: %t", state)
	})
	layout.AddPane(fanPane)

	heaterPane := led.NewOnOffPane("images/heater-off.png", "images/heater-on.gif", func(state bool) {
		log.Printf("Heater state: %t", state)
	})
	layout.AddPane(heaterPane)

	marioPane := led.NewOnOffPane("test/mario.gif", "test/mario.gif", func(state bool) {
		log.Printf("Mario state: %t", state)
	})
	layout.AddPane(marioPane)

	// Toggle fan and heater panes every second
	/*go func() {
		state := false
		for {
			time.Sleep(time.Second * 1)
			state = !state
			fanPane.SetState(state)
			heaterPane.SetState(state)
		}
	}()*/

	layout.AddPane(led.NewColorPane(color.RGBA{0, 0, 255, 255}))

	log.Println("starting")
	c := &serial.Config{Name: "/dev/tty.ledmatrix", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Printf("No led matrix? Ignoring... %s", err)
	}

	<-wake

	go func() {
		for {
			//if s == nil {
			//}
			//time.Sleep(time.Second / 10)
			image, wake, err := layout.Render()
			if err != nil {
				log.Fatal(err)
			}

			if s != nil {
				write(image, s)

				buf := make([]byte, 1)
				_, err := s.Read(buf)
				if err != nil {
					log.Fatal(err)
				}
				if buf[0] != byte('F') {
					log.Fatal("Expected an 'F', got '%q'", buf[0])
				}
			} else {
				//	spew.Dump(image)
			}

			if wake != nil {
				log.Println("Waiting as the UI is asleep")
				<-wake
				log.Println("UI woke up!")
			}
		}
	}()

	/*go func() {
		for {
			time.Sleep(time.Second * 4)
			layout.PanLeft()
		}
	}()*/

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
