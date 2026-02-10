package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mdp/qrterminal/v3" // Updated to v3 version
)

// Use ascii blocks to form the QR Code
const BLACK_WHITE = "▄"
const BLACK_BLACK = " "
const WHITE_BLACK = "▀"
const WHITE_WHITE = "█"

// Global variables to store video file path and service port
var (
	videoFilePath string
	serverPort    int // Changed to int type for easier parameter parsing
)

// Print help information
func printHelp() {
	helpText := `
Video Playback HTTP Service
Usage:
  video-player [options] [video file path]

Features:
  Start an HTTP service to play the specified video file in a browser, 
  supporting access via mobile phone QR code scanning

Options:
  -p, --port int   Specify service port (default 9090, range 1-65535)

Examples:
  video-player ./movie.mp4                # Use default port 9090
  video-player -p 8888 ./movie.mp4        # Use port 8888
  video-player --port 7070 /home/video.mp4

Parameters:
  video file path    Absolute/relative path of the video file to play

Access Methods:
  1. Local access: http://localhost:port
  2. Mobile/LAN device: Scan the terminal QR code, or visit http://[local IP]:port
`
	fmt.Print(helpText)
}

// Get local LAN IP (for mobile QR code access)
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		// Filter loopback addresses and IPv6 addresses, only take IPv4 LAN addresses
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "localhost"
}

// Handle HTTP requests for video files (supports Range partial content)
func videoHandler(w http.ResponseWriter, r *http.Request) {
	// Open video file
	file, err := os.Open(videoFilePath)
	if err != nil {
		http.Error(w, "Failed to open video file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file information
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file information: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fileSize := fileInfo.Size()

	// Handle Range requests (partial content)
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		// Parse Range header, format like "bytes=0-1023"
		rangeParts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		start, err := strconv.ParseInt(rangeParts[0], 10, 64)
		if err != nil {
			http.Error(w, "Invalid Range request", http.StatusBadRequest)
			return
		}

		end := fileSize - 1
		if len(rangeParts) > 1 && rangeParts[1] != "" {
			end, err = strconv.ParseInt(rangeParts[1], 10, 64)
			if err != nil {
				http.Error(w, "Invalid Range request", http.StatusBadRequest)
				return
			}
		}

		// Ensure end does not exceed file size
		if end > fileSize-1 {
			end = fileSize - 1
		}

		// Set response headers
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusPartialContent)

		// Seek file pointer and write data
		_, err = file.Seek(start, io.SeekStart)
		if err != nil {
			http.Error(w, "Failed to seek file", http.StatusInternalServerError)
			return
		}
		io.CopyN(w, file, end-start+1)
		return
	}

	// Non-Range request, return entire file directly
	w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	io.Copy(w, file)
}

// Provide HTML page with embedded video player
func playerHandler(w http.ResponseWriter, r *http.Request) {
	// Get video file name (for page title)
	filename := filepath.Base(videoFilePath)

	// HTML page with embedded HTML5 video player
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>%s - Video Player</title>
    <style>
        body {
            margin: 0;
            padding: 20px;
            background-color: #f0f0f0;
            display: flex;
            flex-direction: column;
            align-items: center;
            font-family: Arial, sans-serif;
        }
        h1 {
            color: #333;
            margin-bottom: 20px;
        }
        video {
            width: 90%%;
            max-width: 1200px;
            height: auto;
            border-radius: 8px;
            box-shadow: 0 4px 8px rgba(0,0,0,0.2);
        }
    </style>
</head>
<body>
    <h1>%s</h1>
    <video controls autoplay preload="metadata">
        <source src="/video" type="video/mp4">
        Your browser does not support HTML5 video playback. Please upgrade your browser.
    </video>
</body>
</html>
`, filename, filename)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func main() {
	// Define port parameter, default 9090
	flag.IntVar(&serverPort, "port", 9090, "Specify service port (default 9090)")
	flag.IntVar(&serverPort, "p", 9090, "Specify service port (short)")
	flag.Usage = printHelp
	flag.Parse()

	// Validate port validity (1-65535)
	if serverPort < 1 || serverPort > 65535 {
		fmt.Printf("Error: Port number %d is invalid, must be in the range 1-65535\n", serverPort)
		os.Exit(1)
	}

	// Get non-flag arguments (video file path)
	args := flag.Args()
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	// Verify video file exists
	videoFilePath = args[0]
	if _, err := os.Stat(videoFilePath); os.IsNotExist(err) {
		fmt.Printf("Error: Video file does not exist -> %s\n", videoFilePath)
		os.Exit(1)
	}

	// Register HTTP routes
	http.HandleFunc("/", playerHandler)
	http.HandleFunc("/video", videoHandler)

	// Get local LAN IP
	localIP := getLocalIP()
	accessURL := fmt.Sprintf("http://%s:%d", localIP, serverPort)

	// Start HTTP service (asynchronous to avoid blocking QR code generation)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil)
		if err != nil {
			fmt.Printf("\nService startup failed: %s\n", err.Error())
			os.Exit(1)
		}
	}()

	// Output startup information
	fmt.Printf("======================\n")
	fmt.Printf("Video file: %s\n", videoFilePath)
	fmt.Printf("Local access: http://localhost:%d\n", serverPort)
	fmt.Printf("LAN access: %s\n", accessURL)
	fmt.Println("======================")
	fmt.Println("Scan QR code to access (mobile phone and computer must be on the same LAN):")

	// Generate and print QR code (adapted to v3 API)
	config := qrterminal.Config{
		Level:          qrterminal.M,
		Writer:         os.Stdout,
		HalfBlocks:     true,
		BlackChar:      BLACK_BLACK,
		WhiteBlackChar: WHITE_BLACK,
		WhiteChar:      WHITE_WHITE,
		BlackWhiteChar: BLACK_WHITE,
		QuietZone:      1,
	}

	qrterminal.GenerateWithConfig(accessURL, config)

	fmt.Println("\nPress Ctrl+C to stop the service")
	// Block main thread
	select {}
}
