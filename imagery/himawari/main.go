package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
)

func main() {
	f, err := os.Open("sample-data/HS_H09_20231130_0030_B03_FLDK_R05_S0510.DAT")
	if err != nil {
		panic(err)
	}
	h, err := DecodeFile(f)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("P3\n%d %d\n255\n", h.DataInfo.NumberOfColumns, h.DataInfo.NumberOfLines))
	fmt.Fprintf(os.Stderr, "%+v\n", h.DataInfo)
	brightness := 40.
	for i, p := range h.ImageData {
		coef := float64(p) / (math.Pow(2., float64(h.DataInfo.NumberOfBitsPerPixel)) - 2.)
		pc := pixel(coef, brightness)
		buf.WriteString(fmt.Sprintf("%d %d %d", pc, pc, pc))
		if i != len(h.ImageData)-1 {
			buf.WriteString(" ")
		}
		if i != 0 && i%int(h.DataInfo.NumberOfColumns) == 0 {
			buf.WriteString("\n")
		}
	}
	_, _ = buf.WriteTo(os.Stdout)

	//for _, p := range h.ImageData {
	//	fmt.Printf("%s ", strconv.FormatInt(int64(p), 2))
	//}

}

func pixel(coef, brig float64) int {
	return int(math.Min(coef*255*brig, 255))
}
