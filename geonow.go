package main

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"golang.org/x/time/rate"
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

// TODO move this to a toml config file
const (
	// Config for the app
	disableThumbCache = false
	cacheDir          = "geonow-cache" // Folder to store cached images
	updateInterval    = time.Minute * 16
	maxWidth          = 3840
	maxHeight         = 3840
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
	vips.Startup(nil)
	defer vips.Shutdown()
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
	src, err := imagery.GetSource(srcName, &imagery.Parameters{MaxWidth: maxWidth})
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
	lastRefresh, err := modTime(latestImage)
	if err != nil {
		http.Error(w, "Failed to get last refresh", http.StatusInternalServerError)
		return
	}
	needsRefresh := isDownloadRequired(lastRefresh)
	needsResize := isResizeRequired(lastRefresh, srcName, dimensions) || needsRefresh || disableThumbCache

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
		srcImg, err := os.Open(imagePath(srcName, "latest.jpg"))
		if err != nil {
			log.Printf("Failed to open latest image: %s", err)
			http.Error(w, "Failed to post process image", http.StatusInternalServerError)
		}
		dstImg, err := os.Create(imagePath(srcName, "latest-clean.jpg"))
		if err != nil {
			log.Printf("Failed to create dst image: %s", err)
			http.Error(w, "Failed to post process image", http.StatusInternalServerError)
		}
		err = src.PostProcess(srcImg, dstImg)
		if err != nil {
			log.Printf("Error post processing image: %v", err)
			http.Error(w, "Failed to post process image", http.StatusInternalServerError)
			return
		}
	}

	// Resize or use cached image
	cachedImagePath := imagePath(srcName, dimensions+".jpg")
	stat, err := os.Stat(cachedImagePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		http.Error(w, "Failed to stat cached image", http.StatusInternalServerError)
		return
	}
	needsResize = disableThumbCache || needsRefresh || errors.Is(err, os.ErrNotExist) || stat.ModTime().Before(lastRefresh)
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

func modTime(filePath string) (time.Time, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, nil
	}
	return stat.ModTime(), nil
}

// TODO: some sources may need different update intervals
func isDownloadRequired(t time.Time) bool {
	return t.Before(time.Now().Add(-updateInterval))
}

func isResizeRequired(lastRefresh time.Time, src string, dimensions string) bool {
	cachedImagePath := imagePath(src, dimensions+".jpg")
	stat, err := os.Stat(cachedImagePath)
	return os.IsNotExist(err) || stat.ModTime().Before(lastRefresh) || os.IsNotExist(err)
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

func resizeImage(srcPath string, width, height int, savePath string) error {
	// convert input.jpg -resize 800x600 -background black -gravity center -extent 800x600 output.jpg
	wh := fmt.Sprintf("%dx%d", width, height)
	dst := fmt.Sprintf("%s/%s.jpg", cacheDir, wh)
	img, err := vips.NewImageFromFile(srcPath)
	if err != nil {
		return err
	}
	// Gravity center resize
	var dim, left, top int
	if width > height {
		dim = height
		left = (width - height) / 2
	} else {
		dim = width
		top = (height - width) / 2
	}
	_ = dim
	// FIXME: this expects a 1:1 image, if not, causes warping
	err = img.ThumbnailWithSize(dim, dim, vips.InterestingAttention, vips.SizeForce)
	if err != nil {
		return err
	}
	err = img.EmbedBackground(left, top, width, height, &vips.Color{
		R: 0,
		G: 0,
		B: 0,
	})
	if err != nil {
		return err
	}
	// TODO: maybe it isn't a good idea to write to a buffer? (memory consumption)
	jpeg, metadata, err := img.ExportJpeg(nil)
	if err != nil {
		return err
	}
	err = os.WriteFile(savePath, jpeg, 0660)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	log.Printf("Resize: %s -> %s, %dx%d", srcPath, dst, metadata.Width, metadata.Height)

	return nil
}
