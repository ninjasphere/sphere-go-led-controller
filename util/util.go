package util

import (
	"image"
	"io"
	"log"
	"math"

	"github.com/davecgh/go-spew/spew"
)

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

func ConvertImage(image *image.RGBA) []byte {

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

	return finalFrame
}

// Write an image into the led matrix
func WriteLEDMatrix(image *image.RGBA, s io.ReadWriteCloser) {

	//spew.Dump("writing image", image)

	finalFrame := ConvertImage(image)

	_, err := s.Write([]byte{cmdWriteBuffer})
	if err != nil {
		log.Printf("Failed writing frame", err)
	}

	_, err = s.Write(finalFrame[:])
	if err != nil {
		log.Printf("Failed writing frame", err)
	}

	_, err = s.Write([]byte{cmdSwapBuffers})
	if err != nil {
		log.Printf("Failed writing frame", err)
	}

	//log.Println("Wrote frame", n)
	buf := make([]byte, 1)
	_, err = s.Read(buf)
	if err != nil {
		log.Printf("Failed to read char after sending frame : %s", err)
	}
	if buf[0] != byte('F') {
		log.Printf("Expected an 'F', got '%q'", buf[0])
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

func compress(frame []byte) []byte {
	compressed := make([]byte, 0)
	for i := 0; i < len(frame); i++ {

		val := frame[i]
		if val == 0 {

			count := 0
			for j := i + 1; j < len(frame) && frame[j] == val; j++ {
				count++
			}

			compressed = append(compressed, val, byte(count))
			i += count
		} else {
			compressed = append(compressed, val)
		}
	}
	spew.Dump("from", frame, compressed)
	return compressed
}
