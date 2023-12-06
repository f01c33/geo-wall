package main

import (
	"compress/bzip2"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
)

func main() {
	src := "HS_H09_20231130_0030_B04_FLDK_R10"
	filePattern := "sample-data/%s_S%02d10.DAT.bz2"
	sectionCount := 10
	var img *image.RGBA
	for section := 0; section < sectionCount; section++ {
		// Decode data
		f, err := os.Open(fmt.Sprintf(filePattern, src, section+1))
		if err != nil {
			panic(err)
		}
		h, err := DecodeFile(bzip2.NewReader(f))
		_ = f.Close()
		if err != nil {
			panic(err)
		}
		// Configure variables for the decoding
		errCount := h.CalibrationInfo.CountValueOfErrorPixels
		outside := h.CalibrationInfo.CountValueOfPixelsOutsideScanArea
		brightness := 1.
		width := int(h.DataInfo.NumberOfColumns)
		height := int(h.DataInfo.NumberOfLines)
		scale := 0.1
		scaledWidth := scale * float64(width)
		scaledHeight := scale * float64(height)
		// Start and End Y are the relative positions for the final image based in a section
		startY := int(scaledHeight) * section
		endY := startY + int(scaledHeight)
		// Initialize image if first loop
		if section == 0 {
			img = image.NewRGBA(image.Rect(0, 0, int(scaledWidth), int(scaledHeight)*sectionCount))
		}
		// The amount of bytes to ignore by scaled width
		widthJump := float64(width) / (scaledWidth + 1.0)
		// The amount of bytes to ignore by scaled height
		heightJump := float64(width) * (float64(height) / scaledHeight)
		n := 0.0
		fmt.Printf("Decoding %dx%d from y %d-%d\n", width, height, startY, endY)
		for y, l := startY, 1.0; y < endY; y, l = y+1, l+1 {
			for x := 0; x < int(scaledWidth); x++ {
				// Do err and outside scan area logic
				rawData := h.ImageData[int(n)]
				// Ignore scaled width bytes
				n += widthJump
				if rawData == errCount || rawData == outside {
					img.Set(x, y, color.Black)
					continue
				}
				// Get a number between 0 and 1 from max number of pixels
				// different bands has different number of pixels, e.g., band 03 has 11
				coef := float64(rawData) / (math.Pow(2., float64(h.CalibrationInfo.ValidNumberOfBitsPerPixel)) - 2.)
				pc := pixel(coef, brightness)
				img.Set(x, y, color.RGBA{R: uint8(pc), G: uint8(pc), B: uint8(pc), A: 255})
			}
			// Ignore scaled height bytes
			n = heightJump * l
		}
	}
	fimg, _ := os.Create("image.jpg")
	err := jpeg.Encode(fimg, img, &jpeg.Options{Quality: 90})
	if err != nil {
		panic(err)
	}
	if err = fimg.Close(); err != nil {
		panic(err)
	}
}

// pixel Returns 255*coef clamping at coef, brightness adjusted
func pixel(coef, brig float64) int {
	return int(math.Min(coef*255*brig, 255))
}
