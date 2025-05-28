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
	
	// Initialize WebView UI
	webUI := ui.NewWebViewUI(appService, 6996)
	
	// Start the web server
	webUI.StartServer()
	
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
