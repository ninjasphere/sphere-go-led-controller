package main

import (
	"image"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"

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

	// This is used to avoid race conditions on startup
	// used by upstart to emit a READY for this service
	if "1" == os.Getenv("LEDCONTROLLER_RAISESTOP") {
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGSTOP)
	}

	enableControl := config.Bool(false, "enableControl")

	controller.start(enableControl)

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Println("Got signal:", x)

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
