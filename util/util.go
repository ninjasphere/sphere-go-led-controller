package util

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"time"

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

type Image interface {
	GetNextFrame() *image.RGBA
	GetPositionFrame(position float64, blend bool) *image.RGBA
}

type SingleImage struct {
	frame *image.RGBA
}

func NewSingleImage(frame *image.RGBA) *SingleImage {
	return &SingleImage{frame}
}

func (i *SingleImage) GetNextFrame() *image.RGBA {
	return i.frame
}

func (i *SingleImage) GetPositionFrame(position float64, blend bool) *image.RGBA {
	return i.frame
}

type AnimatedImage struct {
	frameRequest    chan bool
	pos             int
	frames          []*image.RGBA
	delays          []int
	remainingLoops  int
	started         bool
	delayAdjustment float64
}

func (i *AnimatedImage) GetNextFrame() *image.RGBA {

	if !i.started {
		i.started = true
		i.start()
	}

	log.Printf("Getting next frame: %d", i.pos)

	frame := i.frames[i.pos]

	select {
	case i.frameRequest <- true:
	default:
	}

	return frame
}

func (i *AnimatedImage) start() {

	go func() {
		for {

			delay := i.delays[i.pos]
			if delay == 0 {
				// Wait till this frame has been taken
				<-i.frameRequest
			} else {

				delayDuration := time.Duration(float64(i.delays[i.pos])*i.delayAdjustment) * 10 * time.Millisecond

				log.Printf("Sleeping frame %d for %d", i.pos, delayDuration)
				time.Sleep(delayDuration)
			}

			i.pos++

			if i.pos >= len(i.frames) {
				i.pos = 0
				i.remainingLoops--
			}

			if i.remainingLoops == 0 {
				continue
			}
		}
	}()

}

// GetPositionFrame returns the frame corresponding to the position given 0....1
func (i *AnimatedImage) GetPositionFrame(position float64, blend bool) *image.RGBA {

	relativePosition := position * float64(len(i.frames)-1)

	previousFrame := int(math.Floor(relativePosition))
	nextFrame := int(math.Ceil(relativePosition))

	framePosition := math.Mod(relativePosition, 1)

	//log.Debugf("GetPositionFrame. Frames:%d Position:%f RelativePosition:%f FramePosition:%f PreviousFrame:%d NextFrame:%d", len(i.frames), position, relativePosition, framePosition, previousFrame, nextFrame)
	if !blend || previousFrame == nextFrame {
		// We can just send back a single frame
		return i.frames[previousFrame]
	}

	maskNext := image.NewUniform(color.Alpha{uint8(255 * framePosition)})

	frame := image.NewRGBA(image.Rect(0, 0, 16, 16))

	draw.Draw(frame, frame.Bounds(), i.frames[previousFrame], image.Point{0, 0}, draw.Src)
	draw.DrawMask(frame, frame.Bounds(), i.frames[nextFrame], image.Point{0, 0}, maskNext, image.Point{0, 0}, draw.Over)

	return frame
}

func LoadImage(src string) Image {
	return loadImage(src)
}

// TODO: Add caching?
func loadImage(src string) Image {
	srcLower := strings.ToLower(src)

	if strings.Contains(srcLower, ".gif") {
		return loadGif(src)
	} else if strings.Contains(srcLower, ".png") {
		return loadPng(src)
	} else {
		log.Printf("Unknown image format: %s", src)
	}
	return nil
}

func LoadPng(src string) Image {
	return loadPng(src)
}

func loadPng(src string) Image {
	file, err := os.Open(src)

	if err != nil {
		log.Printf("Could not open png '%s' : %s", src, err)
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Printf("PNG decoding failed on image '%s' : %s", src, err)
	}

	return &SingleImage{
		frame: toRGBA(img),
	}
}

func LoadGif(src string) *AnimatedImage {
	return loadGif(src)
}

func loadGif(src string) *AnimatedImage {
	file, err := os.Open(src)

	if err != nil {
		log.Printf("Could not open gif '%s' : %s", src, err)
	}

	img, err := gif.DecodeAll(file)
	if err != nil {
		log.Printf("Gif decoding failed on image '%s' : %s", src, err)
	}

	var frames = []*image.RGBA{}

	for _, frame := range img.Image {
		frames = append(frames, toRGBA(frame))
	}

	spew.Dump(img.Delay)

	return &AnimatedImage{
		frames:          frames,
		delays:          img.Delay,
		remainingLoops:  img.LoopCount,
		frameRequest:    make(chan bool),
		delayAdjustment: 1.0,
	}
}

func toRGBA(in image.Image) *image.RGBA {
	bounds := in.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), in, bounds.Min, draw.Over)
	return out
}
