# Video Player HTTP Service
A lightweight HTTP service to stream video files in the browser, with QR code support for easy LAN access (mobile-friendly).

## Installation
```bash
# Clone the repository
git clone https://github.com/cjacker/video-player-http.git
cd video-player-http

# Build the binary
go build -o video-player main.go
```

## Usage
```bash
# Use default port (9090)
./video-player path/to/your/video.mp4

# Specify custom port (1-65535)
./video-player -p 8888 path/to/your/video.mp4
./video-player --port 7070 path/to/your/video.mp4
```

## Features
- Stream video files via an HTML5 player (MP4 supported)
- Supports Range requests (enables video seeking)
- Auto-generates QR code for LAN access (mobile/desktop)
- Cross-platform (built with Go)
- User-friendly HTML player interface with responsive design

## Notes
- Ensure the video file path is valid (absolute/relative paths work)
- Mobile devices must be on the same LAN as the server to use the QR code
- Press `Ctrl+C` to stop the service
- Port number must be in the range 1-65535 (default: 9090)
