package imagery

import (
	"bufio"
	"gopkg.in/gographics/imagick.v3/imagick"
	"log"
	"net/http"
)

const (
	baseURL     = "https://cdn.star.nesdis.noaa.gov/GOES16/ABI/FD/GEOCOLOR/"
	latestImage = "latest.jpg" // Name for the latest downloaded image
)

type GoesSource struct{}

// DownloadImage downloads a goes-east, full disk, geo-color, latest image
func (GoesSource) DownloadImage() (*bufio.Reader, error) {
	client := http.Client{}
	resp, err := client.Get(baseURL + latestImage)
	if err != nil {
		return nil, err
	}

	return bufio.NewReader(resp.Body), nil
}

// PostProcess Crop the top/bottom 16px of the GOES image since they are unnecessary
func (GoesSource) PostProcess(src string, dst string) error {
	ret, err := imagick.ConvertImageCommand([]string{
		"magick", src, "-shave", "0x16", "-resize", "x2160", dst,
	})
	if err != nil {
		return err
	}

	log.Printf("Goes post process: %s -> %s, %s", src, dst, ret.Meta)

	return nil
}
