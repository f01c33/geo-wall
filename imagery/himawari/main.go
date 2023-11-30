package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
)

func main() {
	filePattern := "sample-data/HS_H09_20231130_0030_B03_FLDK_R05_S%02d10.DAT"
	sectionCount := 10
	for i := 1; i <= sectionCount; i++ {
		f, err := os.Open(fmt.Sprintf(filePattern, i))
		if err != nil {
			panic(err)
		}
		h, err := DecodeFile(f)
		_ = f.Close()
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		if i == 1 {
			buf.WriteString(fmt.Sprintf("P3\n%d %d\n255\n", h.DataInfo.NumberOfColumns, int(h.DataInfo.NumberOfLines)*sectionCount))
		}
		fmt.Fprintf(os.Stderr, "%+v\n", h.DataInfo)
		fmt.Fprintf(os.Stderr, "%+v\n", h.CalibrationInfo)
		fmt.Fprintf(os.Stderr, "%+v\n", h.InterCalibrationInfo)
		errCount := h.CalibrationInfo.CountValueOfErrorPixels
		outside := h.CalibrationInfo.CountValueOfPixelsOutsideScanArea
		brightness := 1.
		for n, p := range h.ImageData {
			// Do err and outside scan area logic
			coef := float64(p) / (math.Pow(2., float64(h.CalibrationInfo.ValidNumberOfBitsPerPixel)) - 2.)
			if p == errCount || p == outside {
				coef = 0
			}
			pc := pixel(coef, brightness)
			buf.WriteString(fmt.Sprintf("%d %d %d", pc, pc, pc))
			if n != len(h.ImageData)-1 {
				buf.WriteString(" ")
			}
			if n != 0 && n%int(h.DataInfo.NumberOfColumns) == 0 {
				buf.WriteString("\n")
			}
		}
		_, _ = buf.WriteTo(os.Stdout)
	}

}

func pixel(coef, brig float64) int {
	return int(math.Min(coef*255*brig, 255))
}
