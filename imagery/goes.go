package imagery

import (
	"bufio"
	"github.com/davidbyttow/govips/v2/vips"
	"log"
	"net/http"
	"os"
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
	img, err := vips.NewImageFromFile(src)
	if err != nil {
		return err
	}
	err = img.ExtractArea(0, 16, img.Width(), img.Height()-32)
	if err != nil {
		return err
	}
	ratio := float64(img.Width()) / float64(img.Height())
	err = img.Thumbnail(3840, int(3840*ratio), vips.InterestingNone)
	if err != nil {
		return err
	}
	jpeg, metadata, err := img.ExportJpeg(nil)

	err = os.WriteFile(dst, jpeg, 0660)
	if err != nil {
		return err
	}
	log.Printf("Goes post process: %s -> %s, %dx%d", src, dst, metadata.Width, metadata.Height)

	return nil
}
