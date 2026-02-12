package cmd

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ctere1/cert-helper/internal"
)

// normalizeURLPath expects forward-slash separators; serveFileBrowser validates this before calling.
func normalizeURLPath(rawPath string) string {
	cleaned := path.Clean(rawPath)
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

func fileBrowserSectionURL(rawPath string) string {
	cleaned := normalizeURLPath(rawPath)
	if cleaned == "/" {
		return "/#files"
	}
	return fmt.Sprintf("/?files=%s#files", url.QueryEscape(cleaned))
}

func buildDashboardFileBrowser(baseDir, rawPath string) (PageData, error) {
	if rawPath == "" {
		rawPath = "/"
	}
	if strings.HasPrefix(rawPath, "/files") {
		rawPath = strings.TrimPrefix(rawPath, "/files")
		if rawPath == "" {
			rawPath = "/"
		}
	}
	if !strings.HasPrefix(rawPath, "/") {
		rawPath = "/" + rawPath
	}
	if strings.Contains(rawPath, "\\") {
		return PageData{}, fmt.Errorf("invalid path")
	}
	urlPath := normalizeURLPath(rawPath)
	fsPath := baseDir
	if urlPath != "/" {
		fsPath = filepath.Join(baseDir, strings.TrimPrefix(urlPath, "/"))
	}
	fsPath = filepath.Clean(fsPath)
	relPath, err := filepath.Rel(baseDir, fsPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return PageData{}, fmt.Errorf("access denied")
	}
	info, err := os.Stat(fsPath)
	if err != nil {
		return PageData{}, err
	}
	if !info.IsDir() {
		urlPath = normalizeURLPath(path.Dir(urlPath))
		fsPath = filepath.Dir(fsPath)
	}

	fileInfos, err := listFileInfos(fsPath, urlPath)
	if err != nil {
		return PageData{}, err
	}

	var parentPath string
	if urlPath != "/" {
		parentPath = fileBrowserSectionURL(path.Dir(urlPath))
	}
	for i := range fileInfos {
		browserPath := fileBrowserSectionURL(fileInfos[i].Path)
		browserFolderPath := fileBrowserSectionURL(fileInfos[i].FolderPath)
		fileInfos[i].BrowserPath = browserPath
		fileInfos[i].BrowserFolderPath = browserFolderPath
		fileInfos[i].Path = "/files" + fileInfos[i].Path
		fileInfos[i].FolderPath = "/files" + fileInfos[i].FolderPath
	}

	data := PageData{
		CurrentPath: strings.TrimPrefix(urlPath, "/"),
		ParentPath:  parentPath,
		Files:       fileInfos,
		Title:       "Certificate Browser - " + urlPath,
		RootPath:    fileBrowserSectionURL("/"),
		Summary:     buildFileSummary(fileInfos),
	}
	return data, nil
}

func listFileInfos(fsPath, urlPath string) ([]FileInfo, error) {
	files, err := os.ReadDir(fsPath)
	if err != nil {
		return nil, err
	}

	var fileInfos []FileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		filePath := path.Join(urlPath, file.Name())
		folderPath := filePath
		if !file.IsDir() {
			folderPath = path.Dir(filePath)
		}
		folderPath = normalizeURLPath(folderPath)
		systemPath := filepath.Join(fsPath, file.Name())
		systemFolderPath := systemPath
		if !file.IsDir() {
			systemFolderPath = filepath.Dir(systemPath)
		}
		fileInfos = append(fileInfos, FileInfo{
			Name:             file.Name(),
			Size:             info.Size(),
			ModTime:          info.ModTime(),
			IsDir:            file.IsDir(),
			Path:             filePath,
			SystemPath:       systemPath,
			FolderPath:       folderPath,
			SystemFolderPath: systemFolderPath,
		})
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		if fileInfos[i].IsDir != fileInfos[j].IsDir {
			return fileInfos[i].IsDir
		}
		return fileInfos[i].Name < fileInfos[j].Name
	})

	return fileInfos, nil
}

func serveFileBrowser(w http.ResponseWriter, r *http.Request, baseDir, urlPrefix string) {
	// Clean the URL path
	// Reject malformed URLs with backslashes to prevent inconsistent URL vs. filesystem path interpretation.
	rawPath := r.URL.Path
	if strings.Contains(rawPath, "\\") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	urlPath := normalizeURLPath(rawPath)

	// Convert URL path to file system path
	var fsPath string
	if urlPath == "/" {
		fsPath = baseDir
	} else {
		fsPath = filepath.Join(baseDir, strings.TrimPrefix(urlPath, "/"))
	}

	// Security check: ensure we're not serving outside the base directory
	fsPath = filepath.Clean(fsPath)
	relPath, err := filepath.Rel(baseDir, fsPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if path exists
	info, err := os.Stat(fsPath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If it's a file, serve it for download
	if !info.IsDir() {
		// Set appropriate headers for certificate files
		ext := strings.ToLower(filepath.Ext(fsPath))
		switch ext {
		case ".pem":
			w.Header().Set("Content-Type", "application/x-pem-file")
		case ".key":
			w.Header().Set("Content-Type", "application/x-pem-file")
		case ".pfx":
			w.Header().Set("Content-Type", "application/x-pkcs12")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fsPath)))
		http.ServeFile(w, r, fsPath)
		return
	}

	// It's a directory, show file listing
	fileInfos, err := listFileInfos(fsPath, urlPath)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	var parentPath string
	if urlPath != "/" {
		parentPath = normalizeURLPath(path.Dir(urlPath))
	}

	rootPath := "/"
	prefix := strings.TrimSuffix(urlPrefix, "/")
	if prefix != "" && prefix != "/" {
		for i := range fileInfos {
			fileInfos[i].Path = prefix + fileInfos[i].Path
			fileInfos[i].FolderPath = prefix + fileInfos[i].FolderPath
		}
		if parentPath != "" {
			parentPath = prefix + parentPath
		}
		rootPath = prefix + "/"
	}

	data := PageData{
		CurrentPath: strings.TrimPrefix(urlPath, "/"),
		ParentPath:  parentPath,
		Files:       fileInfos,
		Title:       "Certificate Browser - " + urlPath,
		RootPath:    rootPath,
		Summary:     buildFileSummary(fileInfos),
	}

	// Create template with helper functions
	tmpl := template.New("browser").Funcs(template.FuncMap{
		"formatSize": internal.FormatSize,
		"fileExt":    filepath.Ext,
	})

	tmpl, err = tmpl.ParseFS(templateFS, fileBrowserTemplateFile)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, filepath.Base(fileBrowserTemplateFile), data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func buildFileSummary(fileInfos []FileInfo) FileSummary {
	summary := FileSummary{
		Total: len(fileInfos),
	}
	for _, info := range fileInfos {
		if info.IsDir {
			summary.Directories++
			continue
		}
		summary.TotalSize += info.Size
		switch strings.ToLower(filepath.Ext(info.Name)) {
		case ".pem":
			summary.Certificates++
		case ".key":
			summary.Keys++
		case ".pfx":
			summary.Bundles++
		}
	}
	return summary
}
