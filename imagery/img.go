package imagery

import (
	"bufio"
	"fmt"
)

type ImageSource interface {
	// DownloadImage Downloads an image to a reader
	DownloadImage() (*bufio.Reader, error)
	// PostProcess Can be used to clean an image, expects that dst is written somewhere
	// src and dst are file paths
	PostProcess(src string, dst string) error
}

func GetSource(src string) (ImageSource, error) {
	if src == "goes" {
		return GoesSource{}, nil
	}

	return nil, fmt.Errorf("invalid source")
}
