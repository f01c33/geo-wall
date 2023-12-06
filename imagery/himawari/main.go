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
		errCount := h.CalibrationInfo.CountValueOfErrorPixels
		outside := h.CalibrationInfo.CountValueOfPixelsOutsideScanArea
		brightness := 1.
		width := int(h.DataInfo.NumberOfColumns)
		height := int(h.DataInfo.NumberOfLines)
		startX := 0
		endX := width

		scale := 0.5
		scaledWidth := scale * float64(width)
		scaledHeight := scale * float64(height)
		//pace := float64(len(h.ImageData)) / (scaledWidth*scaledHeight + 1.)
		startY := int(scaledHeight) * section
		endY := startY + int(scaledHeight)
		// Initialize image if first loop
		if section == 0 {
			img = image.NewRGBA(image.Rect(0, 0, int(scaledWidth), int(scaledHeight)*sectionCount))
		}
		n := 0.0
		fmt.Printf("Decoding %dx%d from x %d-%d to y %d-%d\n", width, height, startX, endX, startY, endY)
		i := 0
		for y := startY; y < endY; y++ {
			for x := 0; x < int(scaledWidth); x++ {
				// Do err and outside scan area logic
				rawData := h.ImageData[int(n)]
				n += float64(width) / (scaledWidth + 1.0)
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
			i++
			n = float64(width) * (float64(height) / scaledHeight) * float64(i)
			if n >= float64(len(h.ImageData)) {
				break
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
