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
)

func handleOpenInExplorer(w http.ResponseWriter, r *http.Request, baseDir string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	target := r.FormValue("path")
	if target == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		http.Error(w, "Invalid base path", http.StatusInternalServerError)
		return
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	if _, err := os.Stat(absTarget); err != nil {
		http.Error(w, "Path not found", http.StatusNotFound)
		return
	}
	if err := openInExplorer(absTarget); err != nil {
		http.Error(w, "Failed to open path", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func openInExplorer(target string) error {
	var cmd *exec.Cmd
	safeTarget := filepath.Clean(target)
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", safeTarget)
	case "windows":
		cmd = exec.Command("explorer", safeTarget)
	default:
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return fmt.Errorf("xdg-open not found: %w", err)
		}
		cmd = exec.Command("xdg-open", safeTarget)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open %s in file manager: %w", safeTarget, err)
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("File explorer exit: %v", err)
		}
	}()
	return nil
}
