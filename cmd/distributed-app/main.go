package main

import (
	"my-social-network/internal/services"
	"my-social-network/internal/ui"
)

func main() {
	// Initialize application services
	appService := services.NewAppService()

	// Initialize WebView UI
	webUI := ui.NewWebViewUI(appService, 6996)

	// Start the web server
	webUI.StartServer()

	// Start the WebView interface
	webUI.ShowWebView()
}
