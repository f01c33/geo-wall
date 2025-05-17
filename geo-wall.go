package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/f01c33/geo-wall/imagery"
	"github.com/reujab/wallpaper"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func main() {
	// Initialize libvips
	vips.Startup(nil)
	defer vips.Shutdown()

	screenWidth := 4096
	screenHeight := 2160
	for range time.NewTicker(time.Minute * 30).C {
		err := setGoesWallpaper(screenWidth, screenHeight)
		if err != nil {
			log.Fatalf("Failed to set GOES wallpaper: %v", err)
		}
	}
}

// setGoesWallpaper downloads the latest GOES image, processes, resizes, and sets it as wallpaper.
// The width and height parameters are the target dimensions for the wallpaper image.
func setGoesWallpaper(targetWidth, targetHeight int) error {
	// 1. Initialize GOES image source
	goesSource := imagery.GoesSource{MaxWidth: 10000}

	// 2. Download the image
	log.Println("Downloading GOES image...")
	imageBuffer, err := goesSource.DownloadImage()
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	exPath, err := getCacheDir()
	if err != nil {
		panic(err)
	}
	downloadedImagePath := filepath.Join(exPath, "goes_downloaded_raw.jpg")
	downloadedFile, err := os.Create(downloadedImagePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for downloaded image: %w", err)
	}
	defer os.Remove(downloadedImagePath) // Clean up

	_, err = imageBuffer.WriteTo(bufio.NewWriter(downloadedFile))
	downloadedFile.Close() // Close file before PostProcess or Resize reads it
	if err != nil {
		return fmt.Errorf("failed to write downloaded image to disk: %w", err)
	}
	log.Printf("Image downloaded to %s", downloadedImagePath)

	// 3. Post-process the downloaded image (replicating the ImageHandler step)
	postprocessedImagePath := filepath.Join(exPath, "goes_postprocessed.jpg")
	rawImgFile, err := os.Open(downloadedImagePath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded image for post-processing: %w", err)
	}
	defer rawImgFile.Close()

	processedFile, err := os.Create(postprocessedImagePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for post-processed image: %w", err)
	}
	defer os.Remove(postprocessedImagePath) // Clean up

	if err := goesSource.PostProcess(rawImgFile, bufio.NewWriter(processedFile)); err != nil {
		processedFile.Close()
		return fmt.Errorf("failed to post-process image: %w", err)
	}
	processedFile.Close() // Ensure data is flushed before resizing
	log.Printf("Image post-processed and saved to %s", postprocessedImagePath)

	// 4. Resize the image (replicating logic from handlers.resizeImage)
	resizedImagePath := filepath.Join(exPath, fmt.Sprintf("goes_resized_%dx%d.jpg", targetWidth, targetHeight))

	log.Printf("Resizing image to %dx%d...", targetWidth, targetHeight)
	vipsImg, err := vips.NewImageFromFile(postprocessedImagePath)
	if err != nil {
		return fmt.Errorf("failed to open post-processed image for resizing: %w", err)
	}
	defer vipsImg.Close()

	// Exact resize logic from handlers.resizeImage:
	// It creates a square thumbnail based on the smaller dimension of the target
	// and then embeds that square into the target canvas.
	var dimForThumbnail, embedLeft, embedTop int
	if targetWidth > targetHeight { // Target canvas is landscape or square (preferring width)
		dimForThumbnail = targetHeight // Create a square thumbnail based on the target height
		embedLeft = (targetWidth - targetHeight) / 2
		embedTop = 0
	} else { // Target canvas is portrait or square (preferring height)
		dimForThumbnail = targetWidth // Create a square thumbnail based on the target width
		embedLeft = 0
		embedTop = (targetHeight - targetWidth) / 2
	}

	// Create a square thumbnail of size dimForThumbnail x dimForThumbnail
	err = vipsImg.ThumbnailWithSize(dimForThumbnail, dimForThumbnail, vips.InterestingAttention, vips.SizeForce)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail: %w", err)
	}

	// Embed this square thumbnail into the final target dimensions (targetWidth x targetHeight)
	// with a black background for letterboxing/pillarboxing.
	err = vipsImg.EmbedBackground(embedLeft, embedTop, targetWidth, targetHeight, &vips.Color{R: 0, G: 0, B: 0})
	if err != nil {
		return fmt.Errorf("failed to embed image: %w", err)
	}

	jpegBytes, _, err := vipsImg.ExportJpeg(nil) // Using default JPEG export options
	if err != nil {
		return fmt.Errorf("failed to export image to JPEG: %w", err)
	}
	err = os.WriteFile(resizedImagePath, jpegBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write resized image to disk: %w", err)
	}
	log.Printf("Image resized and saved to %s", resizedImagePath)

	// 5. Set as wallpaper
	log.Println("Setting wallpaper...")
	err = wallpaper.SetMode(wallpaper.Fit)
	if err != nil {
		return fmt.Errorf("failed to set wallpaper mode: %w", err)
	}
	err = wallpaper.SetFromFile(resizedImagePath)
	if err != nil {
		return fmt.Errorf("failed to set wallpaper from file: %w", err)
	}
	log.Println("Wallpaper set successfully!")
	return nil
}

func getCacheDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(usr.HomeDir, "Library", "Caches"), nil
}
