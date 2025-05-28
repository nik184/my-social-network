package ui

import (
	"fmt"
	"log"
	"net/http"
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

// NewWebViewUI creates a new WebView UI manager
func NewWebViewUI(appService *services.AppService, port int) *WebViewUI {
	return &WebViewUI{
		appService: appService,
		handler:    handlers.NewHandler(appService),
		port:       port,
	}
}

// StartServer starts the HTTP server for the API and static files
func (w *WebViewUI) StartServer() {
	// Register API routes
	w.handler.RegisterRoutes()
	
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("web/static/")))
	
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