// +build !linux !arm

package util

import (
	"image"
	"io"
)

func GetLEDConnection() (io.ReadWriteCloser, error) {
	panic("not implemented")
}

func WriteLEDMatrix(image *image.RGBA, s io.ReadWriteCloser) {
	panic("not implemented")
}
