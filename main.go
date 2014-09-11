package main

import (
	"image"
	"io"
	"log"
	"math"
	"os"
	"os/signal"

	//"net/http"

	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/sphere-go-led-controller/ui"
	"github.com/tarm/goserial"
)

//import _ "net/http/pprof"

const drivername = "sphere-led-controller"

func main() {

	/*	go func() {
		log.Printf("WOOOO")
		log.Println(http.ListenAndServe(":6060", nil))
	}()*/

	conn, err := ninja.Connect(drivername)

	if err != nil {
		log.Fatalf("Failed to connect to mqtt: %s", err)
	}

	statusJob, err := ninja.CreateStatusJob(conn, drivername)
	if err != nil {
		log.Fatalf("Could not setup status job: %s", err)
	}

	statusJob.Start()

	layout, wake := ui.NewPaneLayout(false)

	rpcClient := conn.GetRPCClient()

	if len(os.Getenv("CERTIFICATION")) > 0 {
		layout.AddPane(ui.NewCertPane(conn.GetMqttClient()))
	} else {
		//layout.AddPane(ui.NewTextScrollPane("Exit Music (For A Film)"))

		heaterPane := ui.NewOnOffPane("images/heater-off.png", "images/heater-on.gif", func(state bool) {
			log.Printf("Heater state: %t", state)
		}, rpcClient, "heater")
		layout.AddPane(heaterPane)
	}

	lightPane := ui.NewLightPane("images/light-off.png", "images/light-on.png", func(state bool) {
		log.Printf("Light on-off state: %t", state)
	}, func(state float64) {
		log.Printf("Light color state: %f", state)
	}, rpcClient)
	layout.AddPane(lightPane)

	fanPane := ui.NewOnOffPane("images/fan-off.png", "images/fan-on.gif", func(state bool) {
		log.Printf("Fan state: %t", state)
	}, rpcClient, "fan")
	layout.AddPane(fanPane)

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

	//	layout.AddPane(ui.NewColorPane(color.RGBA{0, 0, 255, 255}))

	log.Println("starting")
	c := &serial.Config{Name: "/dev/tty.ledmatrix", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Printf("No led matrix? Ignoring... %s", err)
	}

	// Send a blank image to the led matrix
	write(image.NewRGBA(image.Rect(0, 0, 16, 16)), s)

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

var cmdWriteBuffer byte = 1
var cmdSwapBuffers byte = 2

// From https://diarmuid.ie/blog/post/pwm-exponential-led-fading-on-arduino-or-other-platforms
var R = (255 * math.Log10(2)) / (math.Log10(255))
var ledAdjust = make(map[uint8]uint8)

func init() {
	for i := 0; i < 256; i++ {
		ledAdjust[uint8(i)] = uint8(math.Pow(2, (float64(i)/R)) - 1)
	}
}

func write(image *image.RGBA, s io.ReadWriteCloser) {

	//spew.Dump("writing image", image)

	var frame [768]byte

	for inPos, outPos := 0, 0; inPos < len(image.Pix); inPos = inPos + 4 {

		outPos = inPos / 4 * 3

		frame[outPos] = ledAdjust[image.Pix[inPos]]
		frame[outPos+1] = ledAdjust[image.Pix[inPos+1]]
		frame[outPos+2] = ledAdjust[image.Pix[inPos+2]]
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

	_, err := s.Write([]byte{cmdWriteBuffer})
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	_, err = s.Write(finalFrame[:])
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	_, err = s.Write([]byte{cmdSwapBuffers})
	if err != nil {
		log.Fatal("Failed writing frame", err)
	}

	//log.Println("Wrote frame", n)
	buf := make([]byte, 1)
	_, err = s.Read(buf)
	if err != nil {
		log.Fatalf("Failed to read char after sending frame : %s", err)
	}
	if buf[0] != byte('F') {
		log.Fatalf("Expected an 'F', got '%q'", buf[0])
	}
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
