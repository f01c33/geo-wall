package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"golang.org/x/time/rate"
	"gopkg.in/gographics/imagick.v3/imagick"
	"log"
	"matbm.net/geonow/imagery"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Config for the app
	cacheDir       = "geonow-cache" // Folder to store cached images
	updateInterval = time.Minute * 16
	maxWidth       = 3840
	maxHeight      = 2160
	// Rate limit for generating new images (expensive)
	newImageRate  = 0.3
	newImageBurst = 1
	// Rate limit to download cached images (cheap)
	cacheImageRate  = 3
	cacheImageBurst = 3
)

type client struct {
	expensiveLimiter *rate.Limiter
	cheapLimiter     *rate.Limiter
	lastSeen         time.Time
}

var (
	rateMutex sync.Mutex
	clients   = make(map[string]*client)
)

func main() {
	imagick.Initialize()
	defer imagick.Terminate()
	go cleanRateLimits()

	http.HandleFunc("/", imageHandler)
	http.HandleFunc("/redirector", redirectorHandler)
	serveAddr := ":8080"
	log.Printf("Server is running at %s", serveAddr)
	err := http.ListenAndServe(serveAddr, nil)
	if err != nil {
		log.Printf("Failed to serve at %s", serveAddr)
	}
}

//go:embed embed/redirector.html
var redirectorHtml []byte

func redirectorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Write(redirectorHtml)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the IP address from the request.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rateMutex.Lock()
	if _, found := clients[ip]; !found {
		clients[ip] = &client{expensiveLimiter: rate.NewLimiter(newImageRate, newImageBurst), cheapLimiter: rate.NewLimiter(cacheImageRate, cacheImageBurst)}
	}
	clients[ip].lastSeen = time.Now()
	rateMutex.Unlock()

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
	log.Printf("Client request for %s to %dx%d", srcName, width, height)

	// Download latest image if necessary
	// TODO: some source's won't be jpg
	latestImage := imagePath(srcName, "latest.jpg")
	needsRefresh := isDownloadRequired(latestImage)
	needsResize := isResizeRequired(srcName, dimensions)

	// Expensive operation, rate limit it
	if needsRefresh || needsResize {
		rateMutex.Lock()
		if !clients[ip].expensiveLimiter.Allow() {
			rateMutex.Unlock()

			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		rateMutex.Unlock()
	} else {
		// Cheap operation (img serve) shouldn't be too strict
		rateMutex.Lock()
		if !clients[ip].cheapLimiter.Allow() {
			rateMutex.Unlock()

			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		rateMutex.Unlock()
	}

	if needsRefresh {
		log.Printf("Downloading latest %s image", srcName)
		_ = os.Mkdir(cacheDir, 0755)
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
	if needsResize {
		err := resizeImage(imagePath(srcName, "latest-clean.jpg"), width, height, cachedImagePath)
		if err != nil {
			log.Printf("Error processing image %v", err)
			http.Error(w, "Error resizing image", http.StatusInternalServerError)
			return
		}
	}

	http.ServeFile(w, r, cachedImagePath)
}

func isResizeRequired(src string, dimensions string) bool {
	cachedImagePath := imagePath(src, dimensions+".jpg")
	_, err := os.Stat(cachedImagePath)
	return os.IsNotExist(err)
}

func cleanRateLimits() {
	for {
		time.Sleep(time.Minute)
		rateMutex.Lock()
		for key, client := range clients {
			if time.Since(client.lastSeen) > time.Minute*3 {
				delete(clients, key)
			}
		}
		rateMutex.Unlock()
	}
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

	b, err := r.WriteTo(bufio.NewWriter(f))
	if err != nil {
		return err
	}
	log.Printf("%d bytes written to %s", b, dst)

	return nil
}

func parseDimensions(dimensions string) (int, int, error) {
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
	if width < 1 || height < 1 {
		return 0, 0, fmt.Errorf("invalid dimensions")
	}
	if width > maxWidth || height > maxHeight {
		return 0, 0, fmt.Errorf("max dimension is %dx%d", maxWidth, maxHeight)
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
