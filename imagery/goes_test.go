package imagery

import (
	"bytes"
	"image/jpeg"
	"os"
	"testing"
)

func TestPostProcess(t *testing.T) {
	src := "./testdata/goes-east-geocolor.jpg"
	dest := t.TempDir() + "/dest.jpg"
	maxWidth := 256

	err := GoesSource{
		MaxWidth: maxWidth,
	}.PostProcess(src, dest)
	if err != nil {
		t.Errorf("Failed to post process in test: %s", err)
	}

	file, err := os.ReadFile(dest)
	if err != nil {
		t.Errorf("Failed to read %s: %s", dest, err)
	}
	img, err := jpeg.Decode(bytes.NewReader(file))
	if err != nil {
		t.Errorf("Failed to decode %s: %s", dest, err)
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
