//go:build linux || darwin
// +build linux darwin

package ui

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"my-social-network/internal/handlers"
	"my-social-network/internal/services"
)

// WebViewUI manages the WebView interface on Unix systems
type WebViewUI struct {
	appService      *services.AppService
	templateService *services.TemplateService
	handler         *handlers.Handler
	port            int
}

// NewWebViewUI creates a new WebView UI manager with automatic port discovery
func NewWebViewUI(appService *services.AppService, preferredPort int) (*WebViewUI, error) {
	// Find an available port starting from the preferred port
	availablePort, err := services.FindAvailablePort(preferredPort)
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	
	log.Printf("🌐 Using port %d for HTTP server (preferred: %d)", availablePort, preferredPort)
	
	// Find the project root directory
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			log.Fatal("Could not find project root")
		}
		wd = parent
	}
	
	// Initialize template service
	templateDir := filepath.Join(wd, "web", "templates")
	templateService, err := services.NewTemplateService(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create template service: %w", err)
	}
	
	return &WebViewUI{
		appService:      appService,
		templateService: templateService,
		handler:         handlers.NewHandler(appService, templateService),
		port:            availablePort,
	}, nil
}

// GetPort returns the port being used by the HTTP server
func (w *WebViewUI) GetPort() int {
	return w.port
}

// StartServer starts the HTTP server for the API and static files
func (w *WebViewUI) StartServer() {
	// Register API routes and page handlers
	w.handler.RegisterRoutes()
	
	// Find the project root directory
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			log.Fatal("Could not find project root")
		}
		wd = parent
	}
	
	staticDir := filepath.Join(wd, "web", "static")
	log.Printf("Serving static files from: %s", staticDir)
	
	// Serve static files under specific paths only
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(filepath.Join(staticDir, "css")))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(filepath.Join(staticDir, "js")))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	
	go func() {
		log.Printf("Starting web server on port %d", w.port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", w.port), nil); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
	
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
}

// ShowWebView displays the WebView window using system browser
func (w *WebViewUI) ShowWebView() {
	url := fmt.Sprintf("http://localhost:%d/network", w.port)
	log.Printf("🌐 Opening application in system browser: %s", url)
	
	// Try to open in system browser
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		log.Printf("Please open your browser and navigate to: %s", url)
		return
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser automatically: %v", err)
		log.Printf("Please open your browser and navigate to: %s", url)
	}
	
	// Keep the application running
	log.Printf("💡 Application is running. Press Ctrl+C to stop.")
	select {} // Block forever
}