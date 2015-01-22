package ui

import (
	"image"
	"math"
	"math/rand"
	"time"

	"github.com/ninjasphere/gestic-tools/go-gestic-sdk"
	"github.com/ninjasphere/go-ninja/config"
)

var enablePlasmaPane = config.Bool(true, "led.plasma.enabled")

var aSin []float64

func init() {
	aSin = make([]float64, 513)
	for i := 512; i > 0; i-- {
		rad := (float64(i) * 0.703125) * 0.0174532
		aSin[i] = math.Sin(rad) * 1024
	}
}

type PlasmaPane struct {
	t1, t2, t3, t4 int
	p1, p2, p3, p4 int
	rad,
	x,
	idx,
	as, fd, as1, fd1, fd2, ps, ps2 float64
}

func NewPlasmaPane() *PlasmaPane {
	pane := &PlasmaPane{}
	pane.as = 2.6
	pane.fd = 0.4
	pane.as1 = 4.4
	pane.fd1 = 2.2
	pane.ps = -4.4
	pane.ps2 = 3.3

	go func() {
		for {
			time.Sleep(time.Second * 10)
			pane.reset()
		}
	}()

	return pane
}

func (p *PlasmaPane) IsEnabled() bool {
	return enablePlasmaPane
}

func (p *PlasmaPane) reset() {
	p.as = float64(rand.Intn(300) * 5)
	p.fd = float64(rand.Intn(300) * 10)
	p.as1 = float64(rand.Intn(200) * 50)
	p.fd2 = float64(rand.Intn(300) * 50)
	p.ps = float64((rand.Intn(200) * 20) - 10)
	p.ps2 = float64((rand.Intn(200) * 40) - 20)
}

func (p *PlasmaPane) Render() (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	p.t4 = p.p4
	p.t3 = p.p3

	for i := 15; i > 0; i-- {
		p.t1 = p.p1 + 5
		p.t2 = p.p2 + 3

		p.t3 &= 511
		p.t4 &= 511

		for j := 15; j > 0; j-- {
			p.t1 &= 511
			p.t2 &= 511

			x := aSin[p.t1] + aSin[p.t2] + aSin[p.t3] + aSin[p.t4]

			idx := img.PixOffset(i, j)

			//spew.Dump(x/p.as/3, uint8(x/p.as/3))

			img.Pix[idx] = uint8(x / p.as / 3)
			img.Pix[idx+1] = uint8(x / p.fd / 3)
			img.Pix[idx+2] = uint8(x / p.ps / 3)
			img.Pix[idx+3] = 255

			p.t1 += 5
			p.t2 += 3
		}

		p.t4 += int(p.as1)
		p.t3 += int(p.fd1)

	}

	p.p1 += int(p.ps)
	p.p3 += int(p.ps2)

	return img, nil
}

func (p *PlasmaPane) IsDirty() bool {
	return true
}
func (p *PlasmaPane) Gesture(gesture *gestic.GestureMessage) {
	if gesture.Tap.Active() {
		p.reset()
	}
}

/*
ig = document.getElementById("c")
ig.width = 16
ig.height = 16

// dammit opera...
if (!("createImageData" in CanvasRenderingContext2D.prototype)){CanvasRenderingContext2D.prototype.createImageData = function(sw,sh) { return this.getImageData(0,0,sw,sh);}}

var p1 = 0,
p2 = 0,
p3 = 0,
p4 = 0,
t1, t2, t3, t4,
aSin = [],
ti = 15,
cv = ig.getContext('2d'),
cd = cv.createImageData(16, 16),
rad,
i, j, x,
idx,
as = 2.6, fd = 0.4, as1 = 4.4, fd1 = 2.2, ps = -4.4, ps2 = 3.3


function init() {
var i = 512
while (i--) {
rad = (i * 0.703125) * 0.0174532
aSin[i] = Math.sin(rad) * 1024
}
}

function main() {
init()
draw()
}

function rand(va) {
return Math.random(va)
}

document.onclick = function(){
as = rand(300)*5
fd = rand(300)*10
as1 = rand(200)*50
fd2 = rand(300)*50
ps = (rand(200)*20)-10
ps2 = (rand(200)*40)-20
}

function draw() {

cdData = cd.data

t4 = p4
t3 = p3

i = 16; while(i--) {
t1 = p1 + 5
t2 = p2 + 3

t3 &= 511
t4 &= 511

j = 16; while(j--) {
t1 &= 511
t2 &= 511

x = aSin[t1] + aSin[t2] + aSin[t3] + aSin[t4]

idx = (i + j * ig.width) * 4

cdData[idx] = x/as
cdData[idx + 1] = x/fd
cdData[idx + 2] = x/ps
cdData[idx + 3] = 255

t1 += 5
t2 += 3
}

t4 += as1
t3 += fd1

}

cd.data = cdData

cv.putImageData(cd, 0, 0)

p1 += ps
p3 += ps2

setTimeout ( draw, ti)
}

main();*/
