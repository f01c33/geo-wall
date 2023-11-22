package main

import (
	"bufio"
	"fmt"
	"gopkg.in/gographics/imagick.v3/imagick"
	"log"
	"matbm.net/geonow/imagery"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO: separate based in sources
const (
	cacheDir       = "geonow-cache" // Folder to store cached images
	updateInterval = time.Minute * 16
)

func main() {
	imagick.Initialize()
	defer imagick.Terminate()

	http.HandleFunc("/", imageHandler)
	serveAddr := ":8080"
	log.Printf("Server is running on %s", serveAddr)
	err := http.ListenAndServe(serveAddr, nil)
	if err != nil {
		log.Printf("Failed to serve at %s", serveAddr)
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse client request
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	srcName := parts[1]
	src, err := imagery.GetSource(srcName)
	if err != nil {
		http.Error(w, "Invalid source", http.StatusBadRequest)
		return
	}
	dimensions := parts[2]
	width, height, err := parseDimensions(dimensions)
	if err != nil {
		http.Error(w, "Invalid dimensions", http.StatusBadRequest)
		return
	}

	// Download latest image if necessary
	// TODO: some source's won't be jpg
	latestImage := imagePath(srcName, "latest.jpg")
	needsRefresh := isDownloadRequired(latestImage)
	if needsRefresh {
		log.Printf("Downloading latest %s image", srcName)
		_ = os.Mkdir("geo-cache", 0755)
		err = downloadLatestImage(src, latestImage)
		if err != nil {
			log.Printf("Error downloading %s image: %v", srcName, err)
			http.Error(w, "Failed to download latest image", http.StatusInternalServerError)
			return
		}
		err = src.PostProcess(imagePath(srcName, "latest.jpg"), imagePath(srcName, "latest-clean.jpg"))
		if err != nil {
			log.Printf("Error post processing image: %v", err)
			http.Error(w, "Failed to post process image", http.StatusInternalServerError)
			return
		}
	}

	// Resize or use cached image
	cachedImagePath := imagePath(srcName, dimensions+".jpg")
	if _, err := os.Stat(cachedImagePath); os.IsNotExist(err) || needsRefresh {
		err := resizeImage(imagePath(srcName, "latest-clean.jpg"), width, height, cachedImagePath)
		if err != nil {
			log.Printf("Error processing image %v", err)
			http.Error(w, "Error resizing image", http.StatusInternalServerError)
			return
		}
	}

	http.ServeFile(w, r, cachedImagePath)
}

// imagePath returns the cached image path based in a source
func imagePath(src string, name string) string {
	return cacheDir + "/" + src + "-" + name
}

func isDownloadRequired(filePath string) bool {
	// Check if we need to download the latest image
	stat, err := os.Stat(filePath)
	// If it exists and is less than 2 hours old
	// TODO: some sources may need different update intervals
	if err == nil && stat.ModTime().After(time.Now().Add(-updateInterval)) {
		return false
	}
	return true
}

func downloadLatestImage(src imagery.ImageSource, dst string) error {
	// Download the latest img
	r, err := src.DownloadImage()
	if err != nil {
		return err
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	// TODO: log, written x bytes to...
	_, err = r.WriteTo(bufio.NewWriter(f))
	if err != nil {
		return err
	}

	return nil
}

func parseDimensions(dimensions string) (int, int, error) {
	// TODO: invalidate dimensions larger than 4k
	parts := strings.Split(dimensions, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid dimensions format")
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func resizeImage(src string, width, height int, cachedImagePath string) error {
	// convert input.jpg -resize 800x600 -background black -gravity center -extent 800x600 output.jpg
	wh := fmt.Sprintf("%dx%d", width, height)
	dst := fmt.Sprintf("%s/%s.jpg", cacheDir, wh)
	ret, err := imagick.ConvertImageCommand([]string{
		"convert", src, "-resize", wh, "-background", "black", "-gravity", "center", "-extent", wh, cachedImagePath,
	})
	if err != nil {
		return err
	}
	log.Printf("Resize: %s -> %s, %s", src, dst, ret.Meta)

	return nil
}
