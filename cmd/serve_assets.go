package cmd

import (
	"embed"
	"net/http"
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
}

//go:embed templates/*.html templates/*.js templates/*.css
var templateFS embed.FS

var assetConfigs = map[string]assetConfig{
	"/assets/shared.js":        {file: sharedScriptFile, contentType: "text/javascript; charset=utf-8"},
	"/assets/shared.css":       {file: sharedStylesFile, contentType: "text/css; charset=utf-8"},
	"/assets/dashboard.js":     {file: dashboardScriptFile, contentType: "text/javascript; charset=utf-8"},
	"/assets/dashboard.css":    {file: dashboardStylesFile, contentType: "text/css; charset=utf-8"},
	"/assets/file_browser.js":  {file: fileBrowserScriptFile, contentType: "text/javascript; charset=utf-8"},
	"/assets/file_browser.css": {file: fileBrowserStylesFile, contentType: "text/css; charset=utf-8"},
}

func registerAssetHandlers(mux *http.ServeMux) {
	for route, config := range assetConfigs {
		route := route
		config := config
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			serveAsset(w, r, config)
		})
	}
}

func serveAsset(w http.ResponseWriter, r *http.Request, config assetConfig) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := templateFS.ReadFile(config.file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", config.contentType)
	_, _ = w.Write(data)
}
