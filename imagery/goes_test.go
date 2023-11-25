package imagery

import (
	"bytes"
	"image/jpeg"
	"io"
	"os"
	"testing"
)

func TestPostProcess(t *testing.T) {
	fSrc, err := os.Open("./testdata/goes-east-geocolor.jpg")
	if err != nil {
		t.Errorf("Failed to open the source file: %s", err)
	}

	reader := io.Reader(fSrc)
	destData := &bytes.Buffer{}
	writer := io.Writer(destData)
	maxWidth := 256

	err = GoesSource{
		MaxWidth: maxWidth,
	}.PostProcess(reader, writer)
	if err != nil {
		t.Errorf("Failed to post process in test: %s", err)
	}

	img, err := jpeg.Decode(bytes.NewReader(destData.Bytes()))
	if err != nil {
		t.Errorf("Failed to decode destination image: %s", err)
	}

	// Assert that we cut at y (removing metadata)
	rect := img.Bounds()
	if rect.Dx() != maxWidth {
		t.Errorf("Expected %d for img x size", maxWidth)
	}
	if rect.Dy() >= maxWidth {
		t.Errorf("Expected y dimension to be cut")
	}
}
