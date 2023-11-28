package ratelimit

import (
	"fmt"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	rateMutex sync.Mutex
	clients   = make(map[string]*Client)
)

const (
	// Rate limit for generating new images (expensive)
	newImageRate  = 0.3
	newImageBurst = 1
	// Rate limit to download cached images (cheap)
	cacheImageRate  = 3
	cacheImageBurst = 3
)

type Client struct {
	expensiveLimiter *rate.Limiter
	cheapLimiter     *rate.Limiter
	lastSeen         time.Time
}

// GetClient extracts the IP of a client and returns it
func GetClient(r *http.Request) (*Client, error) {
	rateMutex.Lock()
	defer rateMutex.Unlock()
	// Extract the IP address from the request.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get IP from request")
	}
	if _, found := clients[ip]; !found {
		clients[ip] = &Client{expensiveLimiter: rate.NewLimiter(newImageRate, newImageBurst), cheapLimiter: rate.NewLimiter(cacheImageRate, cacheImageBurst)}
	}
	clients[ip].lastSeen = time.Now()

	return clients[ip], nil
}

// CleanRateLimits infinite loop that cleanups rate limit cache every period of time
func CleanRateLimits() {
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

// AllowsCheap returns if a client is allowed to request a cheap operation
func (c *Client) AllowsCheap() bool {
	rateMutex.Lock()
	defer rateMutex.Unlock()
	return c.cheapLimiter.Allow()
}

// AllowsExpensive returns if a client is allowed to request an expensive operation, such as image processing operations
func (c *Client) AllowsExpensive() bool {
	rateMutex.Lock()
	defer rateMutex.Unlock()
	return c.expensiveLimiter.Allow()
}
