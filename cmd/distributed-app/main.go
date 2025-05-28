package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"my-social-network/internal/services"
	"my-social-network/internal/ui"
)

func main() {
	// Initialize application services
	appService := services.NewAppService()
	defer func() {
		if err := appService.Close(); err != nil {
			log.Printf("Error closing app service: %v", err)
		}
	}()
	
	// Initialize WebView UI with automatic port discovery
	webUI, err := ui.NewWebViewUI(appService, 6996)
	if err != nil {
		log.Fatalf("Failed to create WebView UI: %v", err)
	}
	
	// Start the web server
	webUI.StartServer()
	
	// Start file system monitoring
	if err := appService.StartMonitoring(); err != nil {
		log.Printf("Warning: Failed to start file monitoring: %v", err)
	}
	
	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		appService.Close()
		os.Exit(0)
	}()
	
	// Start the WebView interface
	webUI.ShowWebView()
}
