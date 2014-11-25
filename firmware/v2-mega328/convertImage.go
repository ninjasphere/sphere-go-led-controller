package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"os"
	"strings"

	"github.com/ninjasphere/sphere-go-led-controller/util"
)

func main() {
	//log.Printf("Loading image", os.Args[1])

	image := LoadImage(os.Args[1])

	data := make([]string, 0)

	for _, frame := range image.frames {
		bytes := util.ConvertImage(frame)

		chunks := make([]string, len(bytes))
		for j, b := range bytes {
			chunks[j] = fmt.Sprintf("0x%x", b)
		}

		frameStr := "\t{" + strings.Join(chunks, ", ") + "}"

		data = append(data, frameStr)

	}

	fmt.Println(fmt.Sprintf("const uint8_t loading[%d][768] = ", len(data)) + "{\n" + strings.Join(data, ",\n") + "\n};")

	//spew.Dump(image)
}

type Image struct {
	pos    int
	frames []*image.RGBA
}

func (i *Image) GetNextFrame() *image.RGBA {
	i.pos++
	if i.pos >= len(i.frames) {
		i.pos = 0
	}
	return i.frames[i.pos]
}

func (i *Image) GetNumFrames() int {
	return len(i.frames)
}

func (i *Image) GetFrame(frame int) *image.RGBA {
	return i.frames[frame]
}

func LoadImage(src string) *Image {
	return loadImage(src)
}

func loadImage(src string) *Image {
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

func loadPng(src string) *Image {
	file, err := os.Open(src)

	if err != nil {
		log.Printf("Could not open png '%s' : %s", src, err)
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Printf("PNG decoding failed on image '%s' : %s", src, err)
	}

	return &Image{
		frames: []*image.RGBA{toRGBA(img)},
	}
}

func loadGif(src string) *Image {
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

	return &Image{
		frames: frames,
	}
}

func toRGBA(in image.Image) *image.RGBA {
	bounds := in.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), in, bounds.Min, draw.Over)
	return out
}
