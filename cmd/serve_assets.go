package cmd

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	dashboardTemplateFile   = "templates/dashboard.html"
	fileBrowserTemplateFile = "templates/file_browser.html"
	sharedScriptFile        = "templates/shared.js"
	sharedStylesFile        = "templates/shared.css"
	dashboardScriptFile     = "templates/dashboard.js"
	dashboardStylesFile     = "templates/dashboard.css"
	fileBrowserScriptFile   = "templates/file_browser.js"
	fileBrowserStylesFile   = "templates/file_browser.css"
)

type assetConfig struct {
	file        string
	contentType string
	data        []byte    // Cached data
	etag        string    // ETag for caching
	modTime     time.Time // Last modified time
}

//go:embed templates/*.html templates/*.js templates/*.css
var templateFS embed.FS

var (
	assetConfigs = map[string]*assetConfig{
		"/assets/shared.js":        {file: sharedScriptFile, contentType: "text/javascript; charset=utf-8"},
		"/assets/shared.css":       {file: sharedStylesFile, contentType: "text/css; charset=utf-8"},
		"/assets/dashboard.js":     {file: dashboardScriptFile, contentType: "text/javascript; charset=utf-8"},
		"/assets/dashboard.css":    {file: dashboardStylesFile, contentType: "text/css; charset=utf-8"},
		"/assets/file_browser.js":  {file: fileBrowserScriptFile, contentType: "text/javascript; charset=utf-8"},
		"/assets/file_browser.css": {file: fileBrowserStylesFile, contentType: "text/css; charset=utf-8"},
	}

	// Cache initialization
	assetCacheMutex sync.RWMutex
	assetsCached    bool
)

// initAssetCache pre-loads and caches all assets with ETags
func initAssetCache() error {
	assetCacheMutex.Lock()
	defer assetCacheMutex.Unlock()

	if assetsCached {
		return nil
	}

	var totalSize int
	for _, config := range assetConfigs {
		data, err := templateFS.ReadFile(config.file)
		if err != nil {
			return fmt.Errorf("failed to read asset %s: %w", config.file, err)
		}

		// Cache the data
		config.data = data
		totalSize += len(data)

		// Generate ETag (hash of content)
		hash := sha256.Sum256(data)
		config.etag = `"` + hex.EncodeToString(hash[:16]) + `"`

		// Get file info for modification time
		fileInfo, err := fs.Stat(templateFS, config.file)
		if err == nil {
			config.modTime = fileInfo.ModTime()
		} else {
			// Fallback to current time if stat fails
			config.modTime = time.Now()
		}
	}

	assetsCached = true
	log.Printf("Assets cached: %d files, %d KB total", len(assetConfigs), totalSize/1024)
	return nil
}

// registerAssetHandlers registers all asset handlers with the mux
func registerAssetHandlers(mux *http.ServeMux) {
	// Initialize cache on first registration
	if err := initAssetCache(); err != nil {
		log.Fatalf("Failed to initialize asset cache: %v", err)
	}

	for route, config := range assetConfigs {
		route := route   // Capture for closure
		config := config // Capture for closure

		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			serveAsset(w, r, config)
		})
	}
}

// serveAsset serves a cached asset with proper headers and caching support
func serveAsset(w http.ResponseWriter, r *http.Request, config *assetConfig) {
	// Only allow GET and HEAD methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read from cache
	assetCacheMutex.RLock()
	data := config.data
	etag := config.etag
	modTime := config.modTime
	assetCacheMutex.RUnlock()

	// Validate data is available
	if len(data) == 0 {
		log.Printf("ERROR: Asset not cached: %s", config.file)
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	// Set Content-Type
	w.Header().Set("Content-Type", config.contentType)

	// Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Enhanced CSP
	csp := "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self'; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"
	w.Header().Set("Content-Security-Policy", csp)

	// Set caching headers
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")

	// Check If-None-Match (ETag-based caching)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Check If-Modified-Since (time-based caching)
	if ifModSince := r.Header.Get("If-Modified-Since"); ifModSince != "" {
		if t, err := time.Parse(http.TimeFormat, ifModSince); err == nil {
			if !modTime.Truncate(time.Second).After(t) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	// Set Content-Length
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	// Handle HEAD request
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Write response body for GET requests
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		// Only log real errors (not broken pipe from client disconnect)
		if !isBrokenPipe(err) {
			log.Printf("ERROR: Failed to serve %s: %v", config.file, err)
		}
	}
}

// isBrokenPipe checks if the error is due to a broken pipe (client disconnect)
func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "broken pipe") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "write: connection timed out")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetAssetContent returns the cached content of an asset (useful for templates)
func GetAssetContent(filename string) ([]byte, error) {
	assetCacheMutex.RLock()
	defer assetCacheMutex.RUnlock()

	for _, config := range assetConfigs {
		if config.file == filename {
			if config.data != nil {
				return config.data, nil
			}
			return nil, fmt.Errorf("asset %s not cached", filename)
		}
	}

	return nil, fmt.Errorf("asset %s not found", filename)
}

// ReloadAssetCache reloads all assets (useful for development)
func ReloadAssetCache() error {
	assetCacheMutex.Lock()
	assetsCached = false
	assetCacheMutex.Unlock()

	log.Println("Reloading asset cache...")
	return initAssetCache()
}
