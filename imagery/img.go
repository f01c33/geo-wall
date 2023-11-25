package imagery

import (
	"bufio"
	"fmt"
	"io"
)

type ImageSource interface {
	// DownloadImage Downloads an image to a reader
	DownloadImage() (*bufio.Reader, error)
	// PostProcess Can be used to clean an image, expects that dst is written somewhere
	// src and dst are file paths
	PostProcess(src io.Reader, dst io.Writer) error
}

type Parameters struct {
	// MaxWidth defines what is the max width of the images
	MaxWidth int
}

func GetSource(src string, p *Parameters) (ImageSource, error) {
	if src == "goes" {
		return GoesSource{p.MaxWidth}, nil
	}

	return nil, fmt.Errorf("invalid source")
}
