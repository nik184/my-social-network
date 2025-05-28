package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/webview/webview_go"
)

type App struct {
	DirectoryPath string
	FolderInfo    *FolderInfo
	Node          *NetworkNode
}

type FolderInfo struct {
	Path     string    `json:"path"`
	Files    []string  `json:"files"`
	LastScan time.Time `json:"lastScan"`
}

type NetworkNode struct {
	ID       string `json:"id"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	LastSeen time.Time `json:"lastSeen"`
}

func NewApp() *App {
	homeDir, _ := os.UserHomeDir()
	dirPath := filepath.Join(homeDir, "space184")
	
	return &App{
		DirectoryPath: dirPath,
		Node: &NetworkNode{
			ID:   generateNodeID(),
			IP:   getLocalIP(),
			Port: 8080,
		},
	}
}

func (a *App) CreateDirectory() error {
	err := os.MkdirAll(a.DirectoryPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func (a *App) ScanDirectory() error {
	files, err := os.ReadDir(a.DirectoryPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	a.FolderInfo = &FolderInfo{
		Path:     a.DirectoryPath,
		Files:    fileNames,
		LastScan: time.Now(),
	}

	return nil
}

func (a *App) StartWebServer() {
	http.HandleFunc("/api/info", a.handleGetInfo)
	http.HandleFunc("/api/scan", a.handleScan)
	http.HandleFunc("/api/create", a.handleCreate)
	http.HandleFunc("/api/discover", a.handleDiscover)
	
	go func() {
		log.Printf("Starting web server on port %d", a.Node.Port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", a.Node.Port), nil); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
}

func (a *App) handleGetInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folderInfo": a.FolderInfo,
		"node":       a.Node,
	})
}

func (a *App) handleScan(w http.ResponseWriter, r *http.Request) {
	if err := a.ScanDirectory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (a *App) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := a.CreateDirectory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "directory created"})
}

func (a *App) handleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		IP string `json:"ip"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	nodeInfo, err := a.discoverNode(req.IP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

func (a *App) discoverNode(ip string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	resp, err := client.Get(fmt.Sprintf("http://%s:8080/api/info", ip))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result, nil
}

func (a *App) StartWebView() {
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>Distributed Social Network</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 20px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; color: #333; margin-bottom: 30px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .button { background-color: #007bff; color: white; padding: 10px 15px; border: none; border-radius: 5px; cursor: pointer; margin: 5px; }
        .button:hover { background-color: #0056b3; }
        .input { padding: 8px; margin: 5px; border: 1px solid #ddd; border-radius: 3px; width: 200px; }
        .result { background-color: #f8f9fa; padding: 10px; margin: 10px 0; border-radius: 3px; font-family: monospace; white-space: pre-wrap; }
        .status { padding: 10px; margin: 10px 0; border-radius: 3px; }
        .success { background-color: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .error { background-color: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌐 Distributed Social Network</h1>
            <p>Manage and share your space184 directory across the network</p>
        </div>
        
        <div class="section">
            <h3>📁 Directory Management</h3>
            <button class="button" onclick="createDirectory()">Create space184 Directory</button>
            <button class="button" onclick="scanDirectory()">Scan Directory</button>
            <div id="directoryStatus"></div>
            <div id="directoryInfo" class="result"></div>
        </div>
        
        <div class="section">
            <h3>🔍 Network Discovery</h3>
            <input type="text" id="ipInput" class="input" placeholder="Enter IP address" value="127.0.0.1">
            <button class="button" onclick="discoverNode()">Discover Node</button>
            <div id="discoveryStatus"></div>
            <div id="discoveryResult" class="result"></div>
        </div>
        
        <div class="section">
            <h3>ℹ️ Current Node Info</h3>
            <button class="button" onclick="getNodeInfo()">Refresh Info</button>
            <div id="nodeInfo" class="result"></div>
        </div>
    </div>

    <script>
        function showStatus(elementId, message, isError = false) {
            const element = document.getElementById(elementId);
            element.innerHTML = message;
            element.className = 'status ' + (isError ? 'error' : 'success');
        }

        function showResult(elementId, data) {
            document.getElementById(elementId).textContent = JSON.stringify(data, null, 2);
        }

        async function createDirectory() {
            try {
                const response = await fetch('/api/create', { method: 'POST' });
                const data = await response.json();
                showStatus('directoryStatus', 'Directory created successfully!');
            } catch (error) {
                showStatus('directoryStatus', 'Error creating directory: ' + error.message, true);
            }
        }

        async function scanDirectory() {
            try {
                const response = await fetch('/api/scan', { method: 'POST' });
                const data = await response.json();
                showStatus('directoryStatus', 'Directory scanned successfully!');
                getNodeInfo(); // Refresh the info
            } catch (error) {
                showStatus('directoryStatus', 'Error scanning directory: ' + error.message, true);
            }
        }

        async function discoverNode() {
            const ip = document.getElementById('ipInput').value;
            if (!ip) {
                showStatus('discoveryStatus', 'Please enter an IP address', true);
                return;
            }

            try {
                const response = await fetch('/api/discover', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: ip })
                });
                
                if (!response.ok) {
                    throw new Error('Network response was not ok');
                }
                
                const data = await response.json();
                showStatus('discoveryStatus', 'Node discovered successfully!');
                showResult('discoveryResult', data);
            } catch (error) {
                showStatus('discoveryStatus', 'Error discovering node: ' + error.message, true);
                document.getElementById('discoveryResult').textContent = '';
            }
        }

        async function getNodeInfo() {
            try {
                const response = await fetch('/api/info');
                const data = await response.json();
                showResult('nodeInfo', data);
            } catch (error) {
                showStatus('nodeInfo', 'Error getting node info: ' + error.message, true);
            }
        }

        // Load initial node info
        window.onload = function() {
            getNodeInfo();
        };
    </script>
</body>
</html>`

	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("Distributed Social Network")
	w.SetSize(900, 700, webview.HintNone)
	w.Navigate("data:text/html," + htmlContent)
	w.Run()
}

func generateNodeID() string {
	return fmt.Sprintf("node_%d", time.Now().Unix())
}

func getLocalIP() string {
	return "127.0.0.1"
}

func main() {
	app := NewApp()
	
	// Start the web server
	app.StartWebServer()
	
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Start the WebView UI
	app.StartWebView()
}