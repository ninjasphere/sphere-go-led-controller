package util

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"
	"strings"
	"time"
)

// Approx. framerate of the display
const fps = 30

// How long each frame is displayed for
const frameTime = time.Second / fps

// If a frame's delay is under this duration, we will display it
// for a certain number of frames, rather than for a time.
const adjustDelayUnder = time.Millisecond * 300

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
			delay := time.Duration(float64(i.delays[i.pos])*i.delayAdjustment) * 10 * time.Millisecond

			if delay < adjustDelayUnder {

				// Rounded to nearest int
				framesToDisplay := int(math.Floor((float64(delay) / float64(frameTime)) + 0.5))

				// Show for at least one frame
				if framesToDisplay == 0 {
					framesToDisplay = 1
				}

				//log.Printf("Frame wanted a delay of %d so showing for %d frames", delay, framesToDisplay)

				for x := 0; x < framesToDisplay; x++ {
					// Wait till this frame has been taken
					<-i.frameRequest
				}

			} else {
				// Just sleep. At these amounts noone will notice +-1 frame
				//log.Printf("Sleeping frame %d for %d", i.pos, delay)
				time.Sleep(delay)
			}

			// That was the last frame
			if i.pos == len(i.frames)-1 {

				if i.remainingLoops == -1 {
					// Start again, we are looping forever
					i.pos = 0
				} else if i.remainingLoops > 0 {
					// Start again, we still have at least one loop remaining
					i.pos = 0
					i.remainingLoops--
				} else {
					// We're done, this frame gets shown forever *drops mic*.
					break
				}
			} else {
				i.pos++
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
	srcLower := strings.ToLower(src)

	if strings.Contains(srcLower, ".gif") {
		return LoadGif(src)
	} else if strings.Contains(srcLower, ".png") {
		return LoadPng(src)
	} else {
		log.HandleError(fmt.Errorf(src), "Unknown image format")
	}
	return nil
}

func LoadPng(src string) Image {

	file, err := os.Open(src)

	if err != nil {
		log.Fatalf("Could not open png '%s' : %s", src, err)
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Fatalf("PNG decoding failed on image '%s' : %s", src, err)
	}

	return &SingleImage{
		frame: toRGBA(img),
	}
}

func LoadGif(src string) *AnimatedImage {

	file, err := os.Open(src)

	if err != nil {
		log.Fatalf("Could not open gif '%s' : %s", src, err)
	}

	img, err := gif.DecodeAll(file)
	if err != nil {
		log.Fatalf("Gif decoding failed on image '%s' : %s", src, err)
	}

	var frames = []*image.RGBA{}

	for _, frame := range img.Image {
		frames = append(frames, toRGBA(frame))
	}

	loops := img.LoopCount
	if loops == 0 {
		loops = -1
	}

	return &AnimatedImage{
		frames:          frames,
		delays:          img.Delay,
		remainingLoops:  loops,
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

/*

Alt version using different gif decoder... doesn't work

package util

import (
"bytes"
"fmt"
"image"
"image/color"
"image/draw"
"image/png"
"io/ioutil"
"math"
"os"
"strings"
"time"

"github.com/xanthousphoenix/go-gif"
)

// Approx. framerate of the display
const fps = 30

// How long each frame is displayed for
const frameTime = time.Second / fps

// If a frame's delay is under this duration, we will display it
// for a certain number of frames, rather than for a time.
const adjustDelayUnder = time.Millisecond * 300

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
delay := time.Duration(float64(i.delays[i.pos])*i.delayAdjustment) * 10 * time.Millisecond

if delay < adjustDelayUnder {

// Rounded to nearest int
framesToDisplay := int(math.Floor((float64(delay) / float64(frameTime)) + 0.5))

// Show for at least one frame
if framesToDisplay == 0 {
framesToDisplay = 1
}

//log.Printf("Frame wanted a delay of %d so showing for %d frames", delay, framesToDisplay)

for x := 0; x < framesToDisplay; x++ {
// Wait till this frame has been taken
<-i.frameRequest
}

} else {
// Just sleep. At these amounts noone will notice +-1 frame
//log.Printf("Sleeping frame %d for %d", i.pos, delay)
time.Sleep(delay)
}

// That was the last frame
if i.pos == len(i.frames)-1 {

if i.remainingLoops == -1 {
// Start again, we are looping forever
i.pos = 0
} else if i.remainingLoops > 0 {
// Start again, we still have at least one loop remaining
i.pos = 0
i.remainingLoops--
} else {
// We're done, this frame gets shown forever *drops mic*.
break
}
} else {
i.pos++
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
srcLower := strings.ToLower(src)

if strings.Contains(srcLower, ".gif") {
return LoadGif(src)
} else if strings.Contains(srcLower, ".png") {
return LoadPng(src)
} else {
log.HandleError(fmt.Errorf(src), "Unknown image format")
}
return nil
}

func LoadPng(src string) Image {

file, err := os.Open(src)

if err != nil {
log.Fatalf("Could not open png '%s' : %s", src, err)
}

img, err := png.Decode(file)
if err != nil {
log.Fatalf("PNG decoding failed on image '%s' : %s", src, err)
}

return &SingleImage{
frame: toRGBA(img),
}
}

func LoadGif(src string) *AnimatedImage {

b, err := ioutil.ReadFile(src)
if err != nil {
log.Fatalf("Could not open gif '%s' : %s", src, err)
}

reader := bytes.NewReader(b)

img, err := gif.DecodeAll(reader)
if err != nil {
log.Fatalf("Gif decoding failed on image '%s' : %s", src, err)
}

var delays = []int{}
var frames = []*image.RGBA{}

for _, frame := range img.Frames {
delays = append(delays, int(frame.DelayTime))
frames = append(frames, toRGBA(frame.FrameImage))
}

loops := img.Header.LoopCount
if loops == 0 {
loops = -1
}

return &AnimatedImage{
frames:          frames,
delays:          delays,
remainingLoops:  loops,
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


*/
