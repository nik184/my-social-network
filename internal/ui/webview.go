package ui

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	webview "github.com/webview/webview_go"
	"my-social-network/internal/handlers"
	"my-social-network/internal/services"
)

// Removed embed directive for now - we'll serve files directly

// WebViewUI manages the WebView interface
type WebViewUI struct {
	appService *services.AppService
	handler    *handlers.Handler
	port       int
}

// NewWebViewUI creates a new WebView UI manager with automatic port discovery
func NewWebViewUI(appService *services.AppService, preferredPort int) (*WebViewUI, error) {
	// Find an available port starting from the preferred port
	availablePort, err := services.FindAvailablePort(preferredPort)
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	
	log.Printf("🌐 Using port %d for HTTP server (preferred: %d)", availablePort, preferredPort)
	
	return &WebViewUI{
		appService: appService,
		handler:    handlers.NewHandler(appService),
		port:       availablePort,
	}, nil
}

// GetPort returns the port being used by the HTTP server
func (w *WebViewUI) GetPort() int {
	return w.port
}

// StartServer starts the HTTP server for the API and static files
func (w *WebViewUI) StartServer() {
	// Register API routes
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
	
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir(staticDir)))
	
	go func() {
		log.Printf("Starting web server on port %d", w.port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", w.port), nil); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
	
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
}

// ShowWebView displays the WebView window
func (w *WebViewUI) ShowWebView() {
	wv := webview.New(true)
	defer wv.Destroy()
	
	wv.SetTitle("Distributed Social Network")
	wv.SetSize(900, 700, webview.HintNone)
	wv.Navigate(fmt.Sprintf("http://localhost:%d/index.html", w.port))
	wv.Run()
}