package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	serverPort string
	serverHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve certificate management UI and generated files on the network.",
	Long:  "Serve certificate management UI and generated files on the network.\n\nWARNING: Exposing this dashboard to a network grants access to certificate files and operations without authentication.",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, err := cmd.Flags().GetString("output-dir")
		if err != nil {
			return err
		}

		// Ensure the directory exists
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			return errors.Wrap(err, "Directory does not exist\n")
		}

		// Get absolute path
		absDir, err := filepath.Abs(outputDir)
		if err != nil {
			return errors.Wrap(err, "Failed to get absolute path")
		}

		if serverHost != "localhost" && serverHost != "127.0.0.1" && serverHost != "" {
			log.Printf("WARNING: The dashboard is exposed without authentication. Use trusted networks only.")
		}

		printBanner()
		fmt.Printf("Starting certificate dashboard on http://%s:%s\n", serverHost, serverPort)
		fmt.Printf("File browser available at http://%s:%s/#files\n", serverHost, serverPort)
		fmt.Printf("Serving directory: %s\n", absDir)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			handleDashboard(w, r, absDir)
		})
		mux.HandleFunc("/generate/root", func(w http.ResponseWriter, r *http.Request) {
			handleGenerateRoot(w, r, absDir)
		})
		mux.HandleFunc("/generate/intermediate", func(w http.ResponseWriter, r *http.Request) {
			handleGenerateIntermediate(w, r, absDir)
		})
		mux.HandleFunc("/generate/cert", func(w http.ResponseWriter, r *http.Request) {
			handleGenerateCert(w, r, absDir)
		})
		registerAssetHandlers(mux)
		mux.HandleFunc("/open", func(w http.ResponseWriter, r *http.Request) {
			handleOpenInExplorer(w, r, absDir)
		})
		mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/files/", http.StatusMovedPermanently)
		})
		mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
			trimmed := strings.TrimPrefix(r.URL.Path, "/files")
			if trimmed == "" {
				trimmed = "/"
			}
			if strings.Contains(trimmed, "\\") {
				http.Error(w, "Invalid path", http.StatusBadRequest)
				return
			}
			urlPath := normalizeURLPath(trimmed)
			fsPath := absDir
			if urlPath != "/" {
				fsPath = filepath.Join(absDir, strings.TrimPrefix(urlPath, "/"))
			}
			fsPath = filepath.Clean(fsPath)
			relPath, err := filepath.Rel(absDir, fsPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}
			info, err := os.Stat(fsPath)
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if info.IsDir() {
				target := fileBrowserSectionURL(urlPath)
				if !strings.HasPrefix(target, "/") {
					http.Error(w, "Invalid path", http.StatusBadRequest)
					return
				}
				http.Redirect(w, r, target, http.StatusFound)
				return
			}
			r.URL.Path = urlPath
			serveFileBrowser(w, r, absDir, "/files")
		})

		// Auto-open browser
		serverURL := fmt.Sprintf("http://%s:%s", serverHost, serverPort)
		go func() {
			// Wait for the server to start
			time.Sleep(1 * time.Second)
			if err := openBrowser(serverURL); err != nil {
				log.Printf("Note: Could not auto-open browser: %v", err)
			}
		}()

		return http.ListenAndServe(serverHost+":"+serverPort, mux)
	},
}

func printBanner() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║                CERTIFICATE HELPER                ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println("[!] WARNING: No authentication enabled.")
	fmt.Println("[!] Operate only within trusted environments.")
	fmt.Println()
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		// Try xdg-open for Linux
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", url)
		} else {
			return fmt.Errorf("no browser opener found")
		}
	}

	return cmd.Start()
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVarP(&serverPort, "port", "p", "8000", "Port to serve on")
	serveCmd.Flags().StringVar(&serverHost, "host", "localhost", "Host to serve on (default localhost)")
}
