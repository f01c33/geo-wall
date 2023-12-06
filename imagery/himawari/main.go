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
	for i := 0; i < sectionCount; i++ {
		// Decode data
		f, err := os.Open(fmt.Sprintf(filePattern, src, i+1))
		if err != nil {
			panic(err)
		}
		h, err := DecodeFile(bzip2.NewReader(f))
		_ = f.Close()
		if err != nil {
			panic(err)
		}
		// Initialize image if first loop
		if i == 0 {
			img = image.NewRGBA(image.Rect(0, 0, int(h.DataInfo.NumberOfColumns), int(h.DataInfo.NumberOfLines)*sectionCount))
		}
		errCount := h.CalibrationInfo.CountValueOfErrorPixels
		outside := h.CalibrationInfo.CountValueOfPixelsOutsideScanArea
		brightness := 1.
		width := int(h.DataInfo.NumberOfColumns)
		height := int(h.DataInfo.NumberOfLines)
		startX := 0
		startY := height * i
		endX := width
		endY := startY + height
		n := 0
		fmt.Printf("Decoding %dx%d from x %d-%d to y %d-%d\n", width, height, startX, endX, startY, endY)
		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				// Do err and outside scan area logic
				rawData := h.ImageData[n]
				n++
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
